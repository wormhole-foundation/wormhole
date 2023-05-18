package p2p

import (
	"context"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/accountant"
	node_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/version"
	"github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
)

const DefaultPort = 8999

var (
	p2pHeartbeatsSent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_heartbeats_sent_total",
			Help: "Total number of p2p heartbeats sent",
		})
	p2pMessagesSent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_broadcast_messages_sent_total",
			Help: "Total number of p2p pubsub broadcast messages sent",
		})
	p2pMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_broadcast_messages_received_total",
			Help: "Total number of p2p pubsub broadcast messages received",
		}, []string{"type"})
	p2pReceiveChannelOverflow = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_receive_channel_overflow",
			Help: "Total number of p2p received messages dropped due to channel overflow",
		}, []string{"type"})
)

var heartbeatMessagePrefix = []byte("heartbeat|")

var signedObservationRequestPrefix = []byte("signed_observation_request|")

// heartbeatMaxTimeDifference specifies the maximum time difference between the local clock and the timestamp in incoming heartbeat messages. Heartbeats that are this old or this much into the future will be dropped. This value should encompass clock skew and network delay.
var heartbeatMaxTimeDifference = time.Minute * 15

func heartbeatDigest(b []byte) common.Hash {
	return ethcrypto.Keccak256Hash(append(heartbeatMessagePrefix, b...))
}

func signedObservationRequestDigest(b []byte) common.Hash {
	return ethcrypto.Keccak256Hash(append(signedObservationRequestPrefix, b...))
}

type Components struct {
	// P2PIDInHeartbeat determines if the guardian will put it's libp2p node ID in the authenticated heartbeat payload
	P2PIDInHeartbeat           bool
	ListeningAddressesPatterns []string
	// Port on which the Guardian is going to bind
	Port uint
	// ConnMgr is the ConnectionManager that the Guardian is going to use
	ConnMgr *connmgr.BasicConnMgr
	// ProtectedHostByGuardianKey is used to ensure that only one p2p peer can be protected by any given known guardian key
	ProtectedHostByGuardianKey map[common.Address]peer.ID
	// ProtectedHostByGuardianKeyLock is only useful to prevent a race condition in test as ProtectedHostByGuardianKey
	// is only accessed by a single routine at any given time in a running Guardian.
	ProtectedHostByGuardianKeyLock sync.Mutex
}

func (f *Components) ListeningAddresses() []string {
	la := make([]string, 0, len(f.ListeningAddressesPatterns))
	for _, pattern := range f.ListeningAddressesPatterns {
		la = append(la, fmt.Sprintf(pattern, f.Port))
	}
	return la
}

func DefaultComponents() *Components {
	mgr, err := DefaultConnectionManager()
	if err != nil {
		panic(err)
	}

	return &Components{
		P2PIDInHeartbeat: true,
		ListeningAddressesPatterns: []string{
			// Listen on QUIC only.
			// https://github.com/libp2p/go-libp2p/issues/688
			"/ip4/0.0.0.0/udp/%d/quic",
			"/ip6/::/udp/%d/quic",
		},
		Port:                       DefaultPort,
		ConnMgr:                    mgr,
		ProtectedHostByGuardianKey: make(map[common.Address]peer.ID),
	}
}

const LowWaterMarkDefault = 100
const HighWaterMarkDefault = 400

func DefaultConnectionManager() (*connmgr.BasicConnMgr, error) {
	return connmgr.NewConnManager(
		LowWaterMarkDefault,
		HighWaterMarkDefault,

		// GracePeriod set to 0 means that new peers are not protected by a grace period
		connmgr.WithGracePeriod(0),
	)
}

func Run(
	obsvC chan<- *gossipv1.SignedObservation,
	obsvReqC chan<- *gossipv1.ObservationRequest,
	obsvReqSendC <-chan *gossipv1.ObservationRequest,
	gossipSendC chan []byte,
	signedInC chan<- *gossipv1.SignedVAAWithQuorum,
	priv crypto.PrivKey,
	gk *ecdsa.PrivateKey,
	gst *node_common.GuardianSetState,
	networkID string,
	bootstrapPeers string,
	nodeName string,
	disableHeartbeatVerify bool,
	rootCtxCancel context.CancelFunc,
	acct *accountant.Accountant,
	gov *governor.ChainGovernor,
	signedGovCfg chan *gossipv1.SignedChainGovernorConfig,
	signedGovSt chan *gossipv1.SignedChainGovernorStatus,
	components *Components,
	ibcFeatures *string,
	signedQueryReqC chan<- *gossipv1.SignedQueryRequest,
) func(ctx context.Context) error {
	if components == nil {
		components = DefaultComponents()
	}

	return func(ctx context.Context) (re error) {
		p2pReceiveChannelOverflow.WithLabelValues("observation").Add(0)
		p2pReceiveChannelOverflow.WithLabelValues("signed_vaa_with_quorum").Add(0)
		p2pReceiveChannelOverflow.WithLabelValues("signed_observation_request").Add(0)

		logger := supervisor.Logger(ctx)

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
				logger.Info("Connecting to bootstrap peers", zap.String("bootstrap_peers", bootstrapPeers))
				bootstrappers := make([]peer.AddrInfo, 0)
				for _, addr := range strings.Split(bootstrapPeers, ",") {
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
				// TODO(leo): Persistent data store (i.e. address book)
				idht, err := dht.New(ctx, h, dht.Mode(dht.ModeServer),
					// This intentionally makes us incompatible with the global IPFS DHT
					dht.ProtocolPrefix(protocol.ID("/"+networkID)),
					dht.BootstrapPeers(bootstrappers...),
				)
				return idht, err
			}),
		)

		if err != nil {
			panic(err)
		}

		defer func() {
			// TODO: libp2p cannot be cleanly restarted (https://github.com/libp2p/go-libp2p/issues/992)
			logger.Error("p2p routine has exited, cancelling root context...", zap.Error(re))
			rootCtxCancel()
		}()

		nodeIdBytes, err := h.ID().Marshal()
		if err != nil {
			panic(err)
		}

		topic := fmt.Sprintf("%s/%s", networkID, "broadcast")

		logger.Info("Subscribing pubsub topic", zap.String("topic", topic))
		ps, err := pubsub.NewGossipSub(ctx, h)
		if err != nil {
			panic(err)
		}

		th, err := ps.Join(topic)
		if err != nil {
			return fmt.Errorf("failed to join topic: %w", err)
		}

		sub, err := th.Subscribe()
		if err != nil {
			return fmt.Errorf("failed to subscribe topic: %w", err)
		}

		logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
			zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))

		bootTime := time.Now()

		// Periodically run guardian state set cleanup.
		go func() {
			ticker := time.NewTicker(15 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					gst.Cleanup()
				case <-ctx.Done():
					return
				}
			}
		}()

		go func() {
			// Disable heartbeat when no node name is provided (spy mode)
			if nodeName == "" {
				return
			}
			ourAddr := ethcrypto.PubkeyToAddress(gk.PublicKey)

			ctr := int64(0)
			tick := time.NewTicker(15 * time.Second)
			defer tick.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:

					// create a heartbeat
					b := func() []byte {
						DefaultRegistry.mu.Lock()
						defer DefaultRegistry.mu.Unlock()
						networks := make([]*gossipv1.Heartbeat_Network, 0, len(DefaultRegistry.networkStats))
						for _, v := range DefaultRegistry.networkStats {
							errCtr := DefaultRegistry.GetErrorCount(vaa.ChainID(v.Id))
							v.ErrorCount = errCtr
							networks = append(networks, v)
						}

						features := make([]string, 0)
						if gov != nil {
							features = append(features, "governor")
						}
						if acct != nil {
							features = append(features, acct.FeatureString())
						}
						if ibcFeatures != nil && *ibcFeatures != "" {
							features = append(features, *ibcFeatures)
						}

						heartbeat := &gossipv1.Heartbeat{
							NodeName:      nodeName,
							Counter:       ctr,
							Timestamp:     time.Now().UnixNano(),
							Networks:      networks,
							Version:       version.Version(),
							GuardianAddr:  ourAddr.String(),
							BootTimestamp: bootTime.UnixNano(),
							Features:      features,
						}

						if components.P2PIDInHeartbeat {
							heartbeat.P2PNodeId = nodeIdBytes
						}

						if err := gst.SetHeartbeat(ourAddr, h.ID(), heartbeat); err != nil {
							panic(err)
						}
						collectNodeMetrics(ourAddr, h.ID(), heartbeat)

						if gov != nil {
							gov.CollectMetrics(heartbeat, gossipSendC, gk, ourAddr)
						}

						msg := gossipv1.GossipMessage{
							Message: &gossipv1.GossipMessage_SignedHeartbeat{
								SignedHeartbeat: createSignedHeartbeat(gk, heartbeat),
							},
						}

						b, err := proto.Marshal(&msg)
						if err != nil {
							panic(err)
						}
						return b
					}()

					err = th.Publish(ctx, b)
					if err != nil {
						logger.Warn("failed to publish heartbeat message", zap.Error(err))
					}

					p2pHeartbeatsSent.Inc()
					ctr += 1
				}
			}
		}()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-gossipSendC:
					err := th.Publish(ctx, msg)
					p2pMessagesSent.Inc()
					if err != nil {
						logger.Error("failed to publish message from queue", zap.Error(err))
					}
				case msg := <-obsvReqSendC:
					b, err := proto.Marshal(msg)
					if err != nil {
						panic(err)
					}

					// Sign the observation request using our node's guardian key.
					digest := signedObservationRequestDigest(b)
					sig, err := ethcrypto.Sign(digest.Bytes(), gk)
					if err != nil {
						panic(err)
					}

					sReq := &gossipv1.SignedObservationRequest{
						ObservationRequest: b,
						Signature:          sig,
						GuardianAddr:       ethcrypto.PubkeyToAddress(gk.PublicKey).Bytes(),
					}

					envelope := &gossipv1.GossipMessage{
						Message: &gossipv1.GossipMessage_SignedObservationRequest{
							SignedObservationRequest: sReq}}

					b, err = proto.Marshal(envelope)
					if err != nil {
						panic(err)
					}

					// Send to local observation request queue (the loopback message is ignored)
					obsvReqC <- msg

					err = th.Publish(ctx, b)
					p2pMessagesSent.Inc()
					if err != nil {
						logger.Error("failed to publish observation request", zap.Error(err))
					} else {
						logger.Info("published signed observation request", zap.Any("signed_observation_request", sReq))
					}
				}
			}
		}()

		for {
			envelope, err := sub.Next(ctx)
			if err != nil {
				return fmt.Errorf("failed to receive pubsub message: %w", err)
			}

			var msg gossipv1.GossipMessage
			err = proto.Unmarshal(envelope.Data, &msg)
			if err != nil {
				logger.Info("received invalid message",
					zap.Binary("data", envelope.Data),
					zap.String("from", envelope.GetFrom().String()))
				p2pMessagesReceived.WithLabelValues("invalid").Inc()
				continue
			}

			if envelope.GetFrom() == h.ID() {
				logger.Debug("received message from ourselves, ignoring",
					zap.Any("payload", msg.Message))
				p2pMessagesReceived.WithLabelValues("loopback").Inc()
				continue
			}

			logger.Debug("received message",
				zap.Any("payload", msg.Message),
				zap.Binary("raw", envelope.Data),
				zap.String("from", envelope.GetFrom().String()))

			switch m := msg.Message.(type) {
			case *gossipv1.GossipMessage_SignedHeartbeat:
				s := m.SignedHeartbeat
				gs := gst.Get()
				if gs == nil {
					// No valid guardian set yet - dropping heartbeat
					logger.Debug("skipping heartbeat - no guardian set",
						zap.Any("value", s),
						zap.String("from", envelope.GetFrom().String()))
					break
				}
				if heartbeat, err := processSignedHeartbeat(envelope.GetFrom(), s, gs, gst, disableHeartbeatVerify); err != nil {
					p2pMessagesReceived.WithLabelValues("invalid_heartbeat").Inc()
					logger.Debug("invalid signed heartbeat received",
						zap.Error(err),
						zap.Any("payload", msg.Message),
						zap.Any("value", s),
						zap.Binary("raw", envelope.Data),
						zap.String("from", envelope.GetFrom().String()))
				} else {
					p2pMessagesReceived.WithLabelValues("valid_heartbeat").Inc()
					logger.Debug("valid signed heartbeat received",
						zap.Any("value", heartbeat),
						zap.String("from", envelope.GetFrom().String()))

					func() {
						if len(heartbeat.P2PNodeId) != 0 {
							components.ProtectedHostByGuardianKeyLock.Lock()
							defer components.ProtectedHostByGuardianKeyLock.Unlock()
							var peerId peer.ID
							if err = peerId.Unmarshal(heartbeat.P2PNodeId); err != nil {
								logger.Error("p2p_node_id_in_heartbeat_invalid",
									zap.Any("payload", msg.Message),
									zap.Any("value", s),
									zap.Binary("raw", envelope.Data),
									zap.String("from", envelope.GetFrom().String()))
							} else {
								guardianAddr := common.BytesToAddress(s.GuardianAddr)
								prevPeerId, ok := components.ProtectedHostByGuardianKey[guardianAddr]
								if ok {
									if prevPeerId != peerId {
										logger.Info("p2p_guardian_peer_changed",
											zap.String("guardian_addr", guardianAddr.String()),
											zap.String("prevPeerId", prevPeerId.String()),
											zap.String("newPeerId", peerId.String()),
										)
										components.ConnMgr.Unprotect(prevPeerId, "heartbeat")
										components.ConnMgr.Protect(peerId, "heartbeat")
										components.ProtectedHostByGuardianKey[guardianAddr] = peerId
									}
								} else {
									components.ConnMgr.Protect(peerId, "heartbeat")
									components.ProtectedHostByGuardianKey[guardianAddr] = peerId
								}
							}
						} else {
							logger.Debug("p2p_node_id_not_in_heartbeat",
								zap.Error(err),
								zap.Any("payload", heartbeat.NodeName))
						}
					}()
				}
			case *gossipv1.GossipMessage_SignedObservation:
				select {
				case obsvC <- m.SignedObservation:
					p2pMessagesReceived.WithLabelValues("observation").Inc()
				default:
					p2pReceiveChannelOverflow.WithLabelValues("observation").Inc()
				}
			case *gossipv1.GossipMessage_SignedVaaWithQuorum:
				select {
				case signedInC <- m.SignedVaaWithQuorum:
					p2pMessagesReceived.WithLabelValues("signed_vaa_with_quorum").Inc()
				default:
					p2pReceiveChannelOverflow.WithLabelValues("signed_vaa_with_quorum").Inc()
				}
			case *gossipv1.GossipMessage_SignedObservationRequest:
				s := m.SignedObservationRequest
				gs := gst.Get()
				if gs == nil {
					logger.Debug("dropping SignedObservationRequest - no guardian set",
						zap.Any("value", s),
						zap.String("from", envelope.GetFrom().String()))
					break
				}
				r, err := processSignedObservationRequest(s, gs)
				if err != nil {
					p2pMessagesReceived.WithLabelValues("invalid_signed_observation_request").Inc()
					logger.Debug("invalid signed observation request received",
						zap.Error(err),
						zap.Any("payload", msg.Message),
						zap.Any("value", s),
						zap.Binary("raw", envelope.Data),
						zap.String("from", envelope.GetFrom().String()))
				} else {
					logger.Info("valid signed observation request received",
						zap.Any("value", r),
						zap.String("from", envelope.GetFrom().String()))

					select {
					case obsvReqC <- r:
						p2pMessagesReceived.WithLabelValues("signed_observation_request").Inc()
					default:
						p2pReceiveChannelOverflow.WithLabelValues("signed_observation_request").Inc()
					}
				}
			case *gossipv1.GossipMessage_SignedChainGovernorConfig:
				if signedGovCfg != nil {
					signedGovCfg <- m.SignedChainGovernorConfig
				}
			case *gossipv1.GossipMessage_SignedChainGovernorStatus:
				if signedGovSt != nil {
					signedGovSt <- m.SignedChainGovernorStatus
				}
			case *gossipv1.GossipMessage_SignedQueryRequest:
				if signedQueryReqC != nil {
					signedQueryReqC <- m.SignedQueryRequest
				}
			default:
				p2pMessagesReceived.WithLabelValues("unknown").Inc()
				logger.Warn("received unknown message type (running outdated software?)",
					zap.Any("payload", msg.Message),
					zap.Binary("raw", envelope.Data),
					zap.String("from", envelope.GetFrom().String()))
			}
		}
	}
}

func createSignedHeartbeat(gk *ecdsa.PrivateKey, heartbeat *gossipv1.Heartbeat) *gossipv1.SignedHeartbeat {
	ourAddr := ethcrypto.PubkeyToAddress(gk.PublicKey)

	b, err := proto.Marshal(heartbeat)
	if err != nil {
		panic(err)
	}

	// Sign the heartbeat using our node's guardian key.
	digest := heartbeatDigest(b)
	sig, err := ethcrypto.Sign(digest.Bytes(), gk)
	if err != nil {
		panic(err)
	}

	return &gossipv1.SignedHeartbeat{
		Heartbeat:    b,
		Signature:    sig,
		GuardianAddr: ourAddr.Bytes(),
	}
}

func processSignedHeartbeat(from peer.ID, s *gossipv1.SignedHeartbeat, gs *node_common.GuardianSet, gst *node_common.GuardianSetState, disableVerify bool) (*gossipv1.Heartbeat, error) {
	envelopeAddr := common.BytesToAddress(s.GuardianAddr)
	idx, ok := gs.KeyIndex(envelopeAddr)
	var pk common.Address
	if !ok {
		if !disableVerify {
			return nil, fmt.Errorf("invalid message: %s not in guardian set", envelopeAddr)
		}
	} else {
		pk = gs.Keys[idx]
	}

	digest := heartbeatDigest(s.Heartbeat)

	// SECURITY: see whitepapers/0009_guardian_key.md
	if len(heartbeatMessagePrefix)+len(s.Heartbeat) < 34 {
		return nil, fmt.Errorf("invalid message: too short")
	}

	pubKey, err := ethcrypto.Ecrecover(digest.Bytes(), s.Signature)
	if err != nil {
		return nil, errors.New("failed to recover public key")
	}

	signerAddr := common.BytesToAddress(ethcrypto.Keccak256(pubKey[1:])[12:])
	if pk != signerAddr && !disableVerify {
		return nil, fmt.Errorf("invalid signer: %v", signerAddr)
	}

	var h gossipv1.Heartbeat
	err = proto.Unmarshal(s.Heartbeat, &h)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal heartbeat: %w", err)
	}

	if time.Until(time.Unix(0, h.Timestamp)).Abs() > heartbeatMaxTimeDifference {
		return nil, fmt.Errorf("heartbeat is too old or too far into the future")
	}

	if h.GuardianAddr != signerAddr.String() {
		return nil, fmt.Errorf("GuardianAddr in heartbeat does not match signerAddr")
	}

	// Store verified heartbeat in global guardian set state.
	if err := gst.SetHeartbeat(signerAddr, from, &h); err != nil {
		return nil, fmt.Errorf("failed to store in guardian set state: %w", err)
	}

	collectNodeMetrics(signerAddr, from, &h)

	return &h, nil
}

func processSignedObservationRequest(s *gossipv1.SignedObservationRequest, gs *node_common.GuardianSet) (*gossipv1.ObservationRequest, error) {
	envelopeAddr := common.BytesToAddress(s.GuardianAddr)
	idx, ok := gs.KeyIndex(envelopeAddr)
	var pk common.Address
	if !ok {
		return nil, fmt.Errorf("invalid message: %s not in guardian set", envelopeAddr)
	} else {
		pk = gs.Keys[idx]
	}

	// SECURITY: see whitepapers/0009_guardian_key.md
	if len(signedObservationRequestPrefix)+len(s.ObservationRequest) < 34 {
		return nil, fmt.Errorf("invalid observation request: too short")
	}

	digest := signedObservationRequestDigest(s.ObservationRequest)

	pubKey, err := ethcrypto.Ecrecover(digest.Bytes(), s.Signature)
	if err != nil {
		return nil, errors.New("failed to recover public key")
	}

	signerAddr := common.BytesToAddress(ethcrypto.Keccak256(pubKey[1:])[12:])
	if pk != signerAddr {
		return nil, fmt.Errorf("invalid signer: %v", signerAddr)
	}

	var h gossipv1.ObservationRequest
	err = proto.Unmarshal(s.ObservationRequest, &h)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal observation request: %w", err)
	}

	// TODO: implement per-guardian rate limiting

	return &h, nil
}
