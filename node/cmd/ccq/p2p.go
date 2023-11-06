package ccq

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type GuardianSignature struct {
	Index     int
	Signature string
}

type SignedResponse struct {
	Response   *query.QueryResponsePublication
	Signatures []GuardianSignature
}

type P2PSub struct {
	sub        *pubsub.Subscription
	topic_req  *pubsub.Topic
	topic_resp *pubsub.Topic
	host       host.Host
}

func runP2P(ctx context.Context, priv crypto.PrivKey, port uint, networkID, bootstrapPeers, ethRpcUrl, ethCoreAddr string, pendingResponses *PendingResponses, logger *zap.Logger) (*P2PSub, error) {
	// p2p setup
	components := p2p.DefaultComponents()
	components.Port = port

	h, err := p2p.NewHost(logger, ctx, networkID, bootstrapPeers, components, priv)
	if err != nil {
		return nil, err
	}

	topic_req := fmt.Sprintf("%s/%s", networkID, "ccq_req")
	topic_resp := fmt.Sprintf("%s/%s", networkID, "ccq_resp")

	logger.Info("Subscribing pubsub topic", zap.String("topic_req", topic_req), zap.String("topic_resp", topic_resp))

	// Comment from security team in PR #2981: CCQServers should have a parameter of D = 36, Dlo = 19, Dhi = 40, Dout = 18 such that they can reach all Guardians directly.
	gossipParams := pubsub.DefaultGossipSubParams()
	gossipParams.D = 36
	gossipParams.Dlo = 19
	gossipParams.Dhi = 40
	gossipParams.Dout = 18

	ps, err := pubsub.NewGossipSub(ctx, h, pubsub.WithGossipSubParams(gossipParams))
	if err != nil {
		logger.Error("failed to create gossip subscription", zap.Error(err))
		return nil, err
	}

	th_req, err := ps.Join(topic_req)
	if err != nil {
		logger.Error("failed to join request topic", zap.String("topic_req", topic_req), zap.Error(err))
		return nil, err
	}

	th_resp, err := ps.Join(topic_resp)
	if err != nil {
		logger.Error("failed to join response topic", zap.String("topic_resp", topic_resp), zap.Error(err))
		return nil, err
	}

	sub, err := th_resp.Subscribe()
	if err != nil {
		logger.Error("failed to subscribe to response topic", zap.Error(err))
		return nil, err
	}

	logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
		zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))

	// Wait for peers
	for len(th_req.ListPeers()) < 1 {
		time.Sleep(time.Millisecond * 100)
	}

	// Fetch the initial current guardian set
	guardianSet, err := FetchCurrentGuardianSet(ethRpcUrl, ethCoreAddr)
	if err != nil {
		logger.Fatal("Failed to fetch current guardian set", zap.Error(err))
	}
	quorum := vaa.CalculateQuorum(len(guardianSet.Keys))

	// Listen to the p2p network for query responses
	go func() {
		// Maps the request signature to a map of response digests which maps to a list of guardian signatures.
		// A request could have responses with different digests, because the guardians could have
		// different results returned for the query in the event of a rollback.
		responses := make(map[string]map[ethCommon.Hash][]GuardianSignature)
		for {
			envelope, err := sub.Next(ctx)
			if err != nil {
				logger.Fatal("Failed to read next pubsub message", zap.Error(err))
			}
			var msg gossipv1.GossipMessage
			err = proto.Unmarshal(envelope.Data, &msg)
			if err != nil {
				logger.Error("received invalid message", zap.Binary("data", envelope.Data),
					zap.String("from", envelope.GetFrom().String()))
				continue
			}
			switch m := msg.Message.(type) {
			case *gossipv1.GossipMessage_SignedQueryResponse:
				logger.Debug("query response received", zap.Any("response", m.SignedQueryResponse))
				var queryResponse query.QueryResponsePublication
				err := queryResponse.Unmarshal(m.SignedQueryResponse.QueryResponse)
				if err != nil {
					logger.Error("failed to unmarshal response", zap.Error(err))
					continue
				}
				requestSignature := hex.EncodeToString(queryResponse.Request.Signature)
				// Check that we're handling the request for this response
				pendingResponse := pendingResponses.Get(requestSignature)
				if pendingResponse == nil {
					logger.Debug("skipping query response for unknown request", zap.String("signature", requestSignature))
					continue
				}
				// Make sure that the request bytes match
				if !bytes.Equal(queryResponse.Request.QueryRequest, pendingResponse.req.QueryRequest) ||
					!bytes.Equal(queryResponse.Request.Signature, pendingResponse.req.Signature) {
					continue
				}
				digest := query.GetQueryResponseDigestFromBytes(m.SignedQueryResponse.QueryResponse)
				signerBytes, err := ethCrypto.Ecrecover(digest.Bytes(), m.SignedQueryResponse.Signature)
				if err != nil {
					logger.Error("failed to verify signature on response",
						zap.String("digest", digest.Hex()),
						zap.String("signature", hex.EncodeToString(m.SignedQueryResponse.Signature)),
						zap.Error(err))
					continue
				}
				signerAddress := ethCommon.BytesToAddress(ethCrypto.Keccak256(signerBytes[1:])[12:])
				keyIdx, hasKeyIdx := guardianSet.KeyIndex(signerAddress)

				if hasKeyIdx {
					if _, ok := responses[requestSignature]; !ok {
						responses[requestSignature] = make(map[ethCommon.Hash][]GuardianSignature)
					}
					found := false
					for _, gs := range responses[requestSignature][digest] {
						if gs.Index == keyIdx {
							found = true
							break
						}
					}
					if found {
						// Already handled the response from this guardian
						continue
					}
					responses[requestSignature][digest] = append(responses[requestSignature][digest], GuardianSignature{
						Index:     keyIdx,
						Signature: hex.EncodeToString(m.SignedQueryResponse.Signature),
					})
					// quorum is reached when a super-majority of guardians have signed a response with the same digest
					if len(responses[requestSignature][digest]) >= quorum {
						s := &SignedResponse{
							Response:   &queryResponse,
							Signatures: responses[requestSignature][digest],
						}
						delete(responses, requestSignature)
						select {
						case pendingResponse.ch <- s:
							logger.Debug("forwarded query response")
						default:
							logger.Error("failed to write query response to channel, dropping it")
						}
					}
				} else {
					logger.Warn("received observation by unknown guardian - is our guardian set outdated?",
						zap.String("digest", digest.Hex()), zap.String("address", signerAddress.Hex()),
					)
				}
			}
		}
	}()

	return &P2PSub{
		sub:        sub,
		topic_req:  th_req,
		topic_resp: th_resp,
		host:       h,
	}, nil
}
