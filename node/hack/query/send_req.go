// This tool can be used to send various queries to the p2p gossip network.
// It is meant for testing purposes only.

package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"
	"time"

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
	"github.com/tendermint/tendermint/libs/rand"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/gagliardetto/solana-go"
)

// this script has to be run inside kubernetes since it relies on UDP
// https://github.com/kubernetes/kubernetes/issues/47862
// kubectl --namespace=wormhole exec -it spy-0 -- sh -c "cd node/hack/query/ && go run send_req.go"
// one way to iterate inside the container
// kubectl --namespace=wormhole exec -it spy-0 -- bash
// apt update
// apt install nano
// cd node/hack/query
// echo "" > send_req.go
// nano send_req.go
// [paste, ^x, y, enter]
// go run send_req.go

func main() {

	//
	// BEGIN SETUP
	//

	p2pNetworkID := "/wormhole/dev"
	var p2pPort uint = 8998 // don't collide with spy so we can run from the same container in tilt
	p2pBootstrap := "/dns4/guardian-0.guardian/udp/8996/quic/p2p/12D3KooWL3XJ9EMCyZvmmGXL2LMiVBtrVa2BuESsJiXkSj7333Jw"
	nodeKeyPath := "./querier.key"

	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	signingKeyPath := string("./dev.guardian.key")

	logger.Info("Loading signing key", zap.String("signingKeyPath", signingKeyPath))
	sk, err := common.LoadGuardianKey(signingKeyPath, true)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}
	logger.Info("Signing key loaded", zap.String("publicKey", ethCrypto.PubkeyToAddress(sk.PublicKey).Hex()))

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

	//
	// END SETUP
	//

	//
	// Solana Tests
	//

	{
		logger.Info("Running Solana account test")

		// Start of query creation...
		account1, err := solana.PublicKeyFromBase58("Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o")
		if err != nil {
			panic("solana account1 is invalid")
		}
		account2, err := solana.PublicKeyFromBase58("B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE")
		if err != nil {
			panic("solana account2 is invalid")
		}
		callRequest := &query.SolanaAccountQueryRequest{
			Commitment:      "finalized",
			DataSliceOffset: 0,
			DataSliceLength: 100,
			Accounts:        [][query.SolanaPublicKeyLength]byte{account1, account2},
		}

		queryRequest := &query.QueryRequest{
			Nonce: rand.Uint32(),
			PerChainQueries: []*query.PerChainQueryRequest{
				{
					ChainId: 1,
					Query:   callRequest,
				},
			},
		}
		sendSolanaQueryAndGetRsp(queryRequest, sk, th_req, ctx, logger, sub)
	}

	{
		logger.Info("Running Solana PDA test")

		// Start of query creation...
		callRequest := &query.SolanaPdaQueryRequest{
			Commitment:      "finalized",
			DataSliceOffset: 0,
			DataSliceLength: 100,
			PDAs: []query.SolanaPDAEntry{
				query.SolanaPDAEntry{
					ProgramAddress: ethCommon.HexToHash("0x02c806312cbe5b79ef8aa6c17e3f423d8fdfe1d46909fb1f6cdf65ee8e2e6faa"), // Devnet core bridge
					Seeds: [][]byte{
						[]byte("GuardianSet"),
						make([]byte, 4),
					},
				},
			},
		}

		queryRequest := &query.QueryRequest{
			Nonce: rand.Uint32(),
			PerChainQueries: []*query.PerChainQueryRequest{
				{
					ChainId: 1,
					Query:   callRequest,
				},
			},
		}
		sendSolanaQueryAndGetRsp(queryRequest, sk, th_req, ctx, logger, sub)
	}

	logger.Info("Solana tests complete!")
	// return

	//
	// EVM Tests
	//

	wethAbi, err := abi.JSON(strings.NewReader("[{\"constant\":true,\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"name\":\"\",\"type\":\"string\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"},{\"constant\":true,\"inputs\":[],\"name\":\"totalSupply\",\"outputs\":[{\"name\":\"\",\"type\":\"uint256\"}],\"payable\":false,\"stateMutability\":\"view\",\"type\":\"function\"}]"))
	if err != nil {
		panic(err)
	}

	methods := []string{"name", "totalSupply"}
	callData := []*query.EthCallData{}
	to, _ := hex.DecodeString("DDb64fE46a91D46ee29420539FC25FD07c5FEa3E")

	for _, method := range methods {
		data, err := wethAbi.Pack(method)
		if err != nil {
			panic(err)
		}

		callData = append(callData, &query.EthCallData{
			To:   to,
			Data: data,
		})
	}

	// Fetch the latest block number
	//url := "https://localhost:8545"
	url := "http://eth-devnet:8545"
	logger.Info("Querying for latest block height", zap.String("url", url))
	blockNum, err := utils.FetchLatestBlockNumberFromUrl(ctx, url)
	if err != nil {
		logger.Fatal("Failed to fetch latest block number", zap.Error(err))
	}

	logger.Info("latest block", zap.String("num", blockNum.String()), zap.String("encoded", hexutil.EncodeBig(blockNum)))

	// block := "0x28d9630"
	// block := "latest"
	// block := "0x9999bac44d09a7f69ee7941819b0a19c59ccb1969640cc513be09ef95ed2d8e2"

	// Start of query creation...
	callRequest := &query.EthCallQueryRequest{
		BlockId:  hexutil.EncodeBig(blockNum),
		CallData: callData,
	}

	// Send 2 individual requests for the same thing but 5 blocks apart
	// First request...
	logger.Info("calling sendQueryAndGetRsp for ", zap.String("blockNum", blockNum.String()))
	queryRequest := createQueryRequest(callRequest)
	sendQueryAndGetRsp(queryRequest, sk, th_req, ctx, logger, sub, wethAbi, methods)

	// This is just so that when I look at the output, it is easier for me. (Paul)
	logger.Info("sleeping for 5 seconds")
	time.Sleep(time.Second * 5)

	// Second request...
	blockNum = blockNum.Sub(blockNum, big.NewInt(5))
	callRequest2 := &query.EthCallQueryRequest{
		BlockId:  hexutil.EncodeBig(blockNum),
		CallData: callData,
	}
	queryRequest2 := createQueryRequest(callRequest2)
	logger.Info("calling sendQueryAndGetRsp for ", zap.String("blockNum", blockNum.String()))
	sendQueryAndGetRsp(queryRequest2, sk, th_req, ctx, logger, sub, wethAbi, methods)

	// Now, want to send a single query with multiple requests...
	logger.Info("Starting multiquery test in 5...")
	time.Sleep(time.Second * 5)
	multiCallRequest := []*query.EthCallQueryRequest{callRequest, callRequest2}
	multQueryRequest := createQueryRequestWithMultipleRequests(multiCallRequest)
	sendQueryAndGetRsp(multQueryRequest, sk, th_req, ctx, logger, sub, wethAbi, methods)

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
		logger.Fatal("Error closing the host", zap.Error(err))
	}

	//
	// END SHUTDOWN
	//

	logger.Info("Success! All tests passed!")
}

const (
	GuardianKeyArmoredBlock = "WORMHOLE GUARDIAN PRIVATE KEY"
)

func createQueryRequest(callRequest *query.EthCallQueryRequest) *query.QueryRequest {
	queryRequest := &query.QueryRequest{
		Nonce: rand.Uint32(),
		PerChainQueries: []*query.PerChainQueryRequest{
			{
				ChainId: 2,
				Query:   callRequest,
			},
		},
	}
	return queryRequest
}

func createQueryRequestWithMultipleRequests(callRequests []*query.EthCallQueryRequest) *query.QueryRequest {
	perChainQueries := []*query.PerChainQueryRequest{}
	for _, req := range callRequests {
		perChainQueries = append(perChainQueries, &query.PerChainQueryRequest{
			ChainId: 2,
			Query:   req,
		})
	}

	queryRequest := &query.QueryRequest{
		Nonce:           rand.Uint32(),
		PerChainQueries: perChainQueries,
	}
	return queryRequest
}

func sendQueryAndGetRsp(queryRequest *query.QueryRequest, sk *ecdsa.PrivateKey, th *pubsub.Topic, ctx context.Context, logger *zap.Logger, sub *pubsub.Subscription, wethAbi abi.ABI, methods []string) {
	queryRequestBytes, err := queryRequest.Marshal()
	if err != nil {
		panic(err)
	}
	numQueries := len(queryRequest.PerChainQueries)

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

	err = th.Publish(ctx, b)
	if err != nil {
		panic(err)
	}

	logger.Info("Waiting for message...")
	// TODO: max wait time
	// TODO: accumulate signatures to reach quorum
	for {
		envelope, err := sub.Next(ctx)
		if err != nil {
			logger.Panic("failed to receive pubsub message", zap.Error(err))
		}
		var msg gossipv1.GossipMessage
		err = proto.Unmarshal(envelope.Data, &msg)
		if err != nil {
			logger.Info("received invalid message",
				zap.Binary("data", envelope.Data),
				zap.String("from", envelope.GetFrom().String()))
			continue
		}
		var isMatchingResponse bool
		switch m := msg.Message.(type) {
		case *gossipv1.GossipMessage_SignedQueryResponse:
			logger.Info("query response received", zap.Any("response", m.SignedQueryResponse),
				zap.String("responseBytes", hexutil.Encode(m.SignedQueryResponse.QueryResponse)),
				zap.String("sigBytes", hexutil.Encode(m.SignedQueryResponse.Signature)))
			var response query.QueryResponsePublication
			err := response.Unmarshal(m.SignedQueryResponse.QueryResponse)
			if err != nil {
				logger.Warn("failed to unmarshal response", zap.Error(err))
				break
			}
			if bytes.Equal(response.Request.QueryRequest, queryRequestBytes) && bytes.Equal(response.Request.Signature, sig) {
				// TODO: verify response signature
				isMatchingResponse = true

				if len(response.PerChainResponses) != numQueries {
					logger.Warn("unexpected number of per chain query responses", zap.Int("expectedNum", numQueries), zap.Int("actualNum", len(response.PerChainResponses)))
					break
				}
				// Do double loop over responses
				for index := range response.PerChainResponses {
					logger.Info("per chain query response index", zap.Int("index", index))

					var localCallData []*query.EthCallData
					switch ecq := queryRequest.PerChainQueries[index].Query.(type) {
					case *query.EthCallQueryRequest:
						localCallData = ecq.CallData
					default:
						panic("unsupported query type")
					}

					var localResp *query.EthCallQueryResponse
					switch ecq := response.PerChainResponses[index].Response.(type) {
					case *query.EthCallQueryResponse:
						localResp = ecq
					default:
						panic("unsupported query type")
					}

					if len(localResp.Results) != len(localCallData) {
						logger.Warn("unexpected number of results", zap.Int("expectedNum", len(localCallData)), zap.Int("expectedNum", len(localResp.Results)))
						break
					}

					for idx, resp := range localResp.Results {
						result, err := wethAbi.Methods[methods[idx]].Outputs.Unpack(resp)
						if err != nil {
							logger.Warn("failed to unpack result", zap.Error(err))
							break
						}

						resultStr := hexutil.Encode(resp)
						logger.Info("found matching response", zap.Int("idx", idx), zap.Uint64("number", localResp.BlockNumber), zap.String("hash", localResp.Hash.String()), zap.String("time", localResp.Time.String()), zap.String("method", methods[idx]), zap.Any("resultDecoded", result), zap.String("resultStr", resultStr))
					}
				}
			}
		default:
			continue
		}
		if isMatchingResponse {
			break
		}
	}
}

func sendSolanaQueryAndGetRsp(queryRequest *query.QueryRequest, sk *ecdsa.PrivateKey, th *pubsub.Topic, ctx context.Context, logger *zap.Logger, sub *pubsub.Subscription) {
	queryRequestBytes, err := queryRequest.Marshal()
	if err != nil {
		panic(err)
	}
	numQueries := len(queryRequest.PerChainQueries)

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

	err = th.Publish(ctx, b)
	if err != nil {
		panic(err)
	}

	logger.Info("Waiting for message...")
	// TODO: max wait time
	// TODO: accumulate signatures to reach quorum
	for {
		envelope, err := sub.Next(ctx)
		if err != nil {
			logger.Panic("failed to receive pubsub message", zap.Error(err))
		}
		var msg gossipv1.GossipMessage
		err = proto.Unmarshal(envelope.Data, &msg)
		if err != nil {
			logger.Info("received invalid message",
				zap.Binary("data", envelope.Data),
				zap.String("from", envelope.GetFrom().String()))
			continue
		}
		var isMatchingResponse bool
		switch m := msg.Message.(type) {
		case *gossipv1.GossipMessage_SignedQueryResponse:
			logger.Info("query response received", zap.Any("response", m.SignedQueryResponse),
				zap.String("responseBytes", hexutil.Encode(m.SignedQueryResponse.QueryResponse)),
				zap.String("sigBytes", hexutil.Encode(m.SignedQueryResponse.Signature)))
			isMatchingResponse = true

			var response query.QueryResponsePublication
			err := response.Unmarshal(m.SignedQueryResponse.QueryResponse)
			if err != nil {
				logger.Warn("failed to unmarshal response", zap.Error(err))
				break
			}
			if bytes.Equal(response.Request.QueryRequest, queryRequestBytes) && bytes.Equal(response.Request.Signature, sig) {
				// TODO: verify response signature
				isMatchingResponse = true

				if len(response.PerChainResponses) != numQueries {
					logger.Warn("unexpected number of per chain query responses", zap.Int("expectedNum", numQueries), zap.Int("actualNum", len(response.PerChainResponses)))
					break
				}
				// Do double loop over responses
				for index := range response.PerChainResponses {
					switch r := response.PerChainResponses[index].Response.(type) {
					case *query.SolanaAccountQueryResponse:
						logger.Info("solana account query per chain response", zap.Int("index", index), zap.Any("pcr", r))
					case *query.SolanaPdaQueryResponse:
						logger.Info("solana pda query per chain response", zap.Int("index", index), zap.Any("pcr", r))
					default:
						panic(fmt.Sprintf("unsupported query type, should be solana, index: %d", index))
					}
				}
			}
		default:
			continue
		}
		if isMatchingResponse {
			break
		}
	}
}
