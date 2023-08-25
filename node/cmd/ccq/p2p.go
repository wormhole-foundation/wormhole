package ccq

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/certusone/wormhole/node/pkg/p2p"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/query"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/multiformats/go-multiaddr"
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
	sub   *pubsub.Subscription
	topic *pubsub.Topic
	host  host.Host
}

func runP2P(ctx context.Context, priv crypto.PrivKey, port uint, networkID, bootstrap, ethRpcUrl, ethCoreAddr string, pendingResponses *PendingResponses, logger *zap.Logger) (*P2PSub, error) {
	// p2p setup
	components := p2p.DefaultComponents()
	components.Port = port
	h, err := libp2p.New(
		// Use the keypair we generated
		libp2p.Identity(priv),

		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			components.ListeningAddresses()...,
		),

		// Enable TLS security as the only security protocol.
		libp2p.Security(libp2ptls.ID, libp2ptls.New),

		// Enable QUIC transport as the only transport.
		libp2p.Transport(libp2pquic.NewTransport),

		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(components.ConnMgr),

		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			logger.Info("Connecting to bootstrap peers", zap.String("bootstrap_peers", bootstrap))
			bootstrappers := make([]peer.AddrInfo, 0)
			for _, addr := range strings.Split(bootstrap, ",") {
				if addr == "" {
					continue
				}
				ma, err := multiaddr.NewMultiaddr(addr)
				if err != nil {
					logger.Error("Invalid bootstrap address", zap.String("peer", addr), zap.Error(err))
					continue
				}
				pi, err := peer.AddrInfoFromP2pAddr(ma)
				if err != nil {
					logger.Error("Invalid bootstrap address", zap.String("peer", addr), zap.Error(err))
					continue
				}
				if pi.ID == h.ID() {
					logger.Info("We're a bootstrap node")
					continue
				}
				bootstrappers = append(bootstrappers, *pi)
			}
			idht, err := dht.New(ctx, h, dht.Mode(dht.ModeServer),
				// This intentionally makes us incompatible with the global IPFS DHT
				dht.ProtocolPrefix(protocol.ID("/"+networkID)),
				dht.BootstrapPeers(bootstrappers...),
			)
			return idht, err
		}),
	)

	if err != nil {
		return nil, err
	}

	topicName := fmt.Sprintf("%s/%s", networkID, "broadcast")

	logger.Info("Subscribing pubsub topic", zap.String("topic", topicName))
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		return nil, err
	}

	topic, err := ps.Join(topicName)
	if err != nil {
		logger.Error("failed to join topic", zap.Error(err))
		return nil, err
	}

	sub, err := topic.Subscribe()
	if err != nil {
		logger.Error("failed to subscribe topic", zap.Error(err))
		return nil, err
	}

	logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
		zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))

	// Wait for peers
	for len(topic.ListPeers()) < 1 {
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
				logger.Error("Failed to read next pubsub message", zap.Error(err))
				break
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
						pendingResponse.ch <- s
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
		sub:   sub,
		topic: topic,
		host:  h,
	}, nil
}
