package ccq

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
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

func runP2P(
	ctx context.Context,
	priv crypto.PrivKey,
	port uint,
	networkID string,
	bootstrapPeers string,
	ethRpcUrl string,
	ethCoreAddr string,
	pendingResponses *PendingResponses,
	logger *zap.Logger,
	monitorPeers bool,
	loggingMap *LoggingMap,
	gossipAdvertiseAddress string,
	protectedPeers []string,
) (*P2PSub, error) {
	// p2p setup
	components := p2p.DefaultComponents()
	components.Port = port
	components.GossipAdvertiseAddress = gossipAdvertiseAddress

	h, err := p2p.NewHost(logger, ctx, networkID, bootstrapPeers, components, priv)
	if err != nil {
		return nil, err
	}

	if len(protectedPeers) != 0 {
		for _, peerId := range protectedPeers {
			components.ConnMgr.Protect(peer.ID(peerId), "configured")
		}
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

	bootstrappers, _ := p2p.BootstrapAddrs(logger, bootstrapPeers, h.ID())
	successes := p2p.ConnectToPeers(ctx, logger, h, bootstrappers)
	logger.Info("Connected to bootstrap peers", zap.Int("num", successes))

	// Wait for peers
	for len(th_req.ListPeers()) < 1 {
		time.Sleep(time.Millisecond * 100)
	}
	logger.Info("Found peers", zap.Int("numPeers", len(th_req.ListPeers())))

	if monitorPeers {
		logger.Info("Will monitor for missing peers once per minute.")
		go func() {
			t := time.NewTicker(time.Minute)
			for {
				select {
				case <-ctx.Done():
					logger.Info("Context cancelled, exiting peer monitoring.")
				case <-t.C:
					peers := th_req.ListPeers()
					logger.Info("current peers", zap.Int("numPeers", len(peers)), zap.Any("peers", peers))
					peerMap := map[string]struct{}{}
					for _, peer := range peers {
						peerMap[peer.String()] = struct{}{}
					}
					for _, p := range bootstrappers {
						if _, exists := peerMap[p.ID.String()]; !exists {
							logger.Info("attempting to reconnect to peer", zap.String("peer", p.ID.String()))
							if err := h.Connect(ctx, p); err != nil {
								logger.Error("failed to reconnect to peer", zap.String("peer", p.ID.String()), zap.Error(err))
							} else {
								logger.Info("Reconnected to peer", zap.String("peer", p.ID.String()))
								peerMap[p.ID.String()] = struct{}{}
								successfulReconnects.Inc()
							}
						}
					}
				}
			}
		}()
	}

	// Fetch the initial current guardian set
	guardianSet, err := FetchCurrentGuardianSet(ctx, ethRpcUrl, ethCoreAddr)
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
				logger.Error("Failed to read next pubsub message", zap.Error(err))
				return
			}
			var msg gossipv1.GossipMessage
			err = proto.Unmarshal(envelope.Data, &msg)
			if err != nil {
				logger.Error("received invalid message", zap.Binary("data", envelope.Data), zap.String("from", envelope.GetFrom().String()))
				inboundP2pError.WithLabelValues("failed_to_unmarshal_gossip_msg").Inc()
				continue
			}
			switch m := msg.Message.(type) {
			case *gossipv1.GossipMessage_SignedQueryResponse:
				logger.Debug("query response received", zap.Any("response", m.SignedQueryResponse))
				peerId := envelope.GetFrom().String()
				queryResponsesReceived.WithLabelValues(peerId).Inc()
				var queryResponse query.QueryResponsePublication
				err := queryResponse.Unmarshal(m.SignedQueryResponse.QueryResponse)
				if err != nil {
					logger.Error("failed to unmarshal response", zap.Error(err))
					inboundP2pError.WithLabelValues("failed_to_unmarshal_response").Inc()
					continue
				}
				for _, pcr := range queryResponse.PerChainResponses {
					queryResponsesReceivedByChainAndPeerID.WithLabelValues(pcr.ChainId.String(), peerId).Inc()
				}
				requestSignature := hex.EncodeToString(queryResponse.Request.Signature)
				logger.Info("query response received from gossip", zap.String("peerId", peerId), zap.Any("requestId", requestSignature))
				if loggingMap.ShouldLogResponse(requestSignature) {
					var queryRequest query.QueryRequest
					if err := queryRequest.Unmarshal(queryResponse.Request.QueryRequest); err == nil {
						logger.Info("logging response", zap.String("peerId", peerId), zap.Any("requestId", requestSignature), zap.Any("request", queryRequest), zap.Any("response", queryResponse))
					} else {
						logger.Error("logging response (failed to unmarshal request)", zap.String("peerId", peerId), zap.Any("requestId", requestSignature), zap.Any("response", queryResponse))
					}
				}
				// Check that we're handling the request for this response
				pendingResponse := pendingResponses.Get(requestSignature)
				if pendingResponse == nil {
					// This will happen for responses that come in after quorum is reached.
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
					inboundP2pError.WithLabelValues("failed_to_verify_signature").Inc()
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
					numSigners := len(responses[requestSignature][digest])
					if numSigners >= quorum {
						s := &SignedResponse{
							Response:   &queryResponse,
							Signatures: responses[requestSignature][digest],
						}
						delete(responses, requestSignature)
						select {
						case pendingResponse.ch <- s:
							logger.Info("quorum reached, forwarded query response",
								zap.String("peerId", peerId),
								zap.String("userId", pendingResponse.userName),
								zap.Any("requestId", requestSignature),
								zap.Int("numSigners", numSigners),
								zap.Int("quorum", quorum),
							)
						default:
							logger.Error("failed to write query response to channel, dropping it", zap.String("peerId", peerId), zap.Any("requestId", requestSignature))
							// Leave the request in the pending map. It will get cleaned up if it times out.
						}
					} else {
						// Proxy should return early if quorum is no longer possible - i.e maxMatchingResponses + outstandingResponses < quorum
						var totalSigners, maxMatchingResponses int
						for _, signers := range responses[requestSignature] {
							totalSigners += len(signers)
							if len(signers) > maxMatchingResponses {
								maxMatchingResponses = len(signers)
							}
						}
						outstandingResponses := len(guardianSet.Keys) - totalSigners
						pendingResponse.updateStats(maxMatchingResponses, outstandingResponses, quorum)
						if maxMatchingResponses+outstandingResponses < quorum {
							quorumNotMetByUser.WithLabelValues(pendingResponse.userName).Inc()
							failedQueriesByUser.WithLabelValues(pendingResponse.userName).Inc()
							delete(responses, requestSignature)
							select {
							case pendingResponse.errCh <- &ErrorEntry{err: errors.New("quorum not met"), status: http.StatusBadRequest}:
								logger.Info("query failed, quorum not met",
									zap.String("peerId", peerId),
									zap.String("userId", pendingResponse.userName),
									zap.Any("requestId", requestSignature),
									zap.Int("numSigners", numSigners),
									zap.Int("maxMatchingResponses", maxMatchingResponses),
									zap.Int("outstandingResponses", outstandingResponses),
									zap.Int("quorum", quorum),
								)
							default:
								logger.Error("failed to write query error response to channel, dropping it", zap.String("peerId", peerId), zap.Any("requestId", requestSignature))
								// Leave the request in the pending map. It will get cleaned up if it times out.
							}
						} else {
							logger.Info("waiting for more query responses",
								zap.String("peerId", peerId),
								zap.String("userId", pendingResponse.userName),
								zap.Any("requestId", requestSignature),
								zap.Int("numSigners", numSigners),
								zap.Int("maxMatchingResponses", maxMatchingResponses),
								zap.Int("outstandingResponses", outstandingResponses),
								zap.Int("quorum", quorum),
							)
						}
					}
				} else {
					logger.Warn("received observation by unknown guardian - is our guardian set outdated?",
						zap.String("digest", digest.Hex()), zap.String("address", signerAddress.Hex()),
					)
					inboundP2pError.WithLabelValues("unknown_guardian").Inc()
				}
			default:
				// Since CCQ gossip is isolated, this really shouldn't happen.
				logger.Debug("unexpected gossip message type", zap.Any("msg", m))
				inboundP2pError.WithLabelValues("unexpected_gossip_msg_type").Inc()
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
