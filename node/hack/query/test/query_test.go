package query_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/certusone/wormhole/node/hack/query/utils"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/ethereum/go-ethereum/accounts/abi"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

func TestCrossChainQuery(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test, set environment variable INTEGRATION")
	}

	p2pNetworkID := "/wormhole/dev"
	var p2pPort uint = 8997
	p2pBootstrap := "/dns4/guardian-0.guardian/udp/8996/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw"
	nodeKeyPath := "../querier.key"

	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	signingKeyPath := string("../dev.guardian.key")

	logger.Info("Loading signing key", zap.String("signingKeyPath", signingKeyPath))
	sk, err := common.LoadGuardianKey(signingKeyPath, true)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}
	logger.Info("Signing key loaded", zap.String("publicKey", ethCrypto.PubkeyToAddress(sk.PublicKey).Hex()))

	// Fetch the current guardian set
	idx, sgs, err := utils.FetchCurrentGuardianSet(common.GoTest)
	if err != nil {
		logger.Fatal("Failed to fetch current guardian set", zap.Error(err))
	}
	logger.Info("Fetched guardian set", zap.Any("keys", sgs.Keys))
	gs := common.GuardianSet{
		Keys:  sgs.Keys,
		Index: idx,
	}

	// Fetch the latest block number
	blockNum, err := utils.FetchLatestBlockNumber(ctx, common.GoTest)
	if err != nil {
		logger.Fatal("Failed to fetch latest block number", zap.Error(err))
	}

	// Load p2p private key
	var priv crypto.PrivKey
	priv, err = common.GetOrCreateNodeKey(logger, nodeKeyPath)
	if err != nil {
		logger.Fatal("Failed to load node key", zap.Error(err))
	}

	// Manual p2p setup
	components := p2p.DefaultComponents()
	components.Port = p2pPort
	bootstrapPeers := p2pBootstrap
	networkID := p2pNetworkID + "/ccq"

	h, err := p2p.NewHost(logger, ctx, networkID, bootstrapPeers, components, priv)
	if err != nil {
		panic(err)
	}

	topic_req := fmt.Sprintf("%s/%s", networkID, "ccq_req")
	topic_resp := fmt.Sprintf("%s/%s", networkID, "ccq_resp")

	logger.Info("Subscribing pubsub topic", zap.String("topic_req", topic_req), zap.String("topic_resp", topic_resp))
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	th_req, err := ps.Join(topic_req)
	if err != nil {
		logger.Panic("failed to join request topic", zap.String("topic_req", topic_req), zap.Error(err))
	}

	th_resp, err := ps.Join(topic_resp)
	if err != nil {
		logger.Panic("failed to join response topic", zap.String("topic_resp", topic_resp), zap.Error(err))
	}

	sub, err := th_resp.Subscribe()
	if err != nil {
		logger.Panic("failed to subscribe to response topic", zap.Error(err))
	}

	logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
		zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))

	// Wait for peers
	for len(th_req.ListPeers()) < 1 {
		time.Sleep(time.Millisecond * 100)
	}

	logger.Info("Detected peers")

	wethAbi, err := abi.JSON(strings.NewReader("[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"))
	if err != nil {
		panic(err)
	}

	methodName := "name"
	data, err := wethAbi.Pack(methodName)
	if err != nil {
		panic(err)
	}
	to, _ := hex.DecodeString("DDb64fE46a91D46ee29420539FC25FD07c5FEa3E") // WETH

	callData := []*query.EthCallData{
		{
			To:   to,
			Data: data,
		},
	}

	callRequest := &query.EthCallQueryRequest{
		BlockId:  hexutil.EncodeBig(blockNum),
		CallData: callData,
	}

	queryRequest := &query.QueryRequest{
		Nonce: 1,
		PerChainQueries: []*query.PerChainQueryRequest{
			{
				ChainId: 2,
				Query:   callRequest,
			},
		},
	}

	queryRequestBytes, err := queryRequest.Marshal()
	if err != nil {
		panic(err)
	}

	// Sign the query request using our private key.
	digest := query.QueryRequestDigest(common.UnsafeDevNet, queryRequestBytes)
	sig, err := ethCrypto.Sign(digest.Bytes(), sk)
	if err != nil {
		panic(err)
	}

	signedQueryRequest := &gossipv1.SignedQueryRequest{
		QueryRequest: queryRequestBytes,
		Signature:    sig,
	}

	msg := gossipv1.GossipMessage{
		Message: &gossipv1.GossipMessage_SignedQueryRequest{
			SignedQueryRequest: signedQueryRequest,
		},
	}

	b, err := proto.Marshal(&msg)
	if err != nil {
		panic(err)
	}

	err = th_req.Publish(ctx, b)
	if err != nil {
		panic(err)
	}

	logger.Info("Waiting for message...")
	var success bool
	signers := map[int]bool{}
	// The guardians can retry for up to a minute so we have to wait longer than that.
	subCtx, cancel := context.WithTimeout(ctx, 75*time.Second)
	defer cancel()
	for {
		envelope, err := sub.Next(subCtx)
		if err != nil {
			break
		}
		var msg gossipv1.GossipMessage
		err = proto.Unmarshal(envelope.Data, &msg)
		if err != nil {
			logger.Fatal("received invalid message",
				zap.Binary("data", envelope.Data),
				zap.String("from", envelope.GetFrom().String()))
		}
		switch m := msg.Message.(type) {
		case *gossipv1.GossipMessage_SignedQueryResponse:
			logger.Info("query response received", zap.Any("response", m.SignedQueryResponse))
			var response query.QueryResponsePublication
			err := response.Unmarshal(m.SignedQueryResponse.QueryResponse)
			if err != nil {
				logger.Fatal("failed to unmarshal response", zap.Error(err))
			}
			if bytes.Equal(response.Request.QueryRequest, queryRequestBytes) && bytes.Equal(response.Request.Signature, sig) {
				digest := query.GetQueryResponseDigestFromBytes(m.SignedQueryResponse.QueryResponse)
				signerBytes, err := ethCrypto.Ecrecover(digest.Bytes(), m.SignedQueryResponse.Signature)
				if err != nil {
					logger.Fatal("failed to verify signature on response",
						zap.String("digest", digest.Hex()),
						zap.String("signature", hex.EncodeToString(m.SignedQueryResponse.Signature)),
						zap.Error(err))
				}
				signerAddress := ethCommon.BytesToAddress(ethCrypto.Keccak256(signerBytes[1:])[12:])
				if keyIdx, ok := gs.KeyIndex(signerAddress); !ok {
					logger.Fatal("received observation by unknown guardian - is our guardian set outdated?",
						zap.String("digest", digest.Hex()),
						zap.String("address", signerAddress.Hex()),
						zap.Uint32("index", gs.Index),
						zap.Any("keys", gs.KeysAsHexStrings()),
					)
				} else {
					signers[keyIdx] = true
				}
				quorum := vaa.CalculateQuorum(len(gs.Keys))
				if len(signers) < quorum {
					logger.Sugar().Infof("not enough signers, have %d need %d", len(signers), quorum)
					continue
				}

				if len(response.PerChainResponses) != 1 {
					logger.Warn("unexpected number of per chain query responses", zap.Int("expectedNum", 1), zap.Int("actualNum", len(response.PerChainResponses)))
					break
				}

				var pcq *query.EthCallQueryResponse
				switch ecq := response.PerChainResponses[0].Response.(type) {
				case *query.EthCallQueryResponse:
					pcq = ecq
				default:
					panic("unsupported query type")
				}

				if len(pcq.Results) == 0 {
					logger.Warn("response did not contain any results", zap.Error(err))
					break
				}

				for idx, resp := range pcq.Results {
					result, err := wethAbi.Methods[methodName].Outputs.Unpack(resp)
					if err != nil {
						logger.Warn("failed to unpack result", zap.Error(err))
						break
					}

					resultStr := hexutil.Encode(resp)
					logger.Info("found matching response", zap.Int("idx", idx), zap.Uint64("number", pcq.BlockNumber), zap.String("hash", pcq.Hash.String()), zap.String("time", pcq.Time.String()), zap.Any("resultDecoded", result), zap.String("resultStr", resultStr))
				}

				success = true
			}
		default:
			continue
		}
		if success {
			break
		}
	}

	assert.True(t, success)

	// Cleanly shutdown
	// Without this the same host won't properly discover peers until some timeout
	sub.Cancel()
	if err := th_req.Close(); err != nil {
		logger.Fatal("Error closing the request topic", zap.Error(err))
	}
	if err := th_resp.Close(); err != nil {
		logger.Fatal("Error closing the response topic", zap.Error(err))
	}
	if err := h.Close(); err != nil {
		logger.Error("Error closing the host", zap.Error(err))
	}
}

const (
	GuardianKeyArmoredBlock = "WORMHOLE GUARDIAN PRIVATE KEY"
)
