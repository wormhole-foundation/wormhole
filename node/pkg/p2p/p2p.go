package p2p

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/query"
	"github.com/certusone/wormhole/node/pkg/version"
	eth_common "github.com/ethereum/go-ethereum/common"
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
	"go.uber.org/zap/zapcore"
	"google.golang.org/protobuf/proto"

	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/supervisor"
)

const DefaultPort = 8999

const P2P_VALIDATE_QUEUE_SIZE = 1024
const P2P_SUBSCRIPTION_BUFFER_SIZE = 1024

// TESTNET_BOOTSTRAP_DHI configures how many nodes may connect to the testnet bootstrap node. This number should not exceed HighWaterMark.
const TESTNET_BOOTSTRAP_DHI = 350

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

func heartbeatDigest(b []byte) eth_common.Hash {
	return ethcrypto.Keccak256Hash(append(heartbeatMessagePrefix, b...))
}

func signedObservationRequestDigest(b []byte) eth_common.Hash {
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
	ProtectedHostByGuardianKey map[eth_common.Address]peer.ID
	// ProtectedHostByGuardianKeyLock is only useful to prevent a race condition in test as ProtectedHostByGuardianKey
	// is only accessed by a single routine at any given time in a running Guardian.
	ProtectedHostByGuardianKeyLock sync.Mutex
	// WarnChannelOverflow: If true, errors due to overflowing channels will produce logger.Warn
	WarnChannelOverflow bool
	// SignedHeartbeatLogLevel is the log level at which SignedHeartbeatReceived events will be logged.
	SignedHeartbeatLogLevel zapcore.Level
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
		ProtectedHostByGuardianKey: make(map[eth_common.Address]peer.ID),
		SignedHeartbeatLogLevel:    zapcore.DebugLevel,
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

// bootstrapAddrs takes a comma-separated string of multi-address strings and returns an array of []peer.AddrInfo that does not include `self`.
// if `self` is part of `bootstrapPeers`, return isBootstrapNode=true
func bootstrapAddrs(logger *zap.Logger, bootstrapPeers string, self peer.ID) (bootstrappers []peer.AddrInfo, isBootstrapNode bool) {
	bootstrappers = make([]peer.AddrInfo, 0)
	for _, addr := range strings.Split(bootstrapPeers, ",") {
		if addr == "" {
			continue
		}
		ma, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			logger.Error("invalid bootstrap address", zap.String("peer", addr), zap.Error(err))
			continue
		}
		pi, err := peer.AddrInfoFromP2pAddr(ma)
		if err != nil {
			logger.Error("invalid bootstrap address", zap.String("peer", addr), zap.Error(err))
			continue
		}
		if pi.ID == self {
			logger.Info("We're a bootstrap node")
			isBootstrapNode = true
			continue
		}
		bootstrappers = append(bootstrappers, *pi)
	}
	return
}

// connectToPeers connects `h` to `peers` and returns the number of successful connections.
func connectToPeers(ctx context.Context, logger *zap.Logger, h host.Host, peers []peer.AddrInfo) (successes int) {
	successes = 0
	for _, p := range peers {
		if err := h.Connect(ctx, p); err != nil {
			logger.Error("failed to connect to bootstrap peer", zap.String("peer", p.String()), zap.Error(err))
		} else {
			successes += 1
		}
	}
	return successes
}

func NewHost(logger *zap.Logger, ctx context.Context, networkID string, bootstrapPeers string, components *Components, priv crypto.PrivKey) (host.Host, error) {
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

			bootstrappers, _ := bootstrapAddrs(logger, bootstrapPeers, h.ID())

			// TODO(leo): Persistent data store (i.e. address book)
			idht, err := dht.New(ctx, h, dht.Mode(dht.ModeServer),
				// This intentionally makes us incompatible with the global IPFS DHT
				dht.ProtocolPrefix(protocol.ID("/"+networkID)),
				dht.BootstrapPeers(bootstrappers...),
			)
			return idht, err
		}),
	)

	return h, err
}

func Run(
	obsvC chan<- *common.MsgWithTimeStamp[gossipv1.SignedObservation],
	obsvReqC chan<- *gossipv1.ObservationRequest,
	obsvReqSendC <-chan *gossipv1.ObservationRequest,
	gossipSendC chan []byte,
	signedInC chan<- *gossipv1.SignedVAAWithQuorum,
	priv crypto.PrivKey,
	gk *ecdsa.PrivateKey,
	gst *common.GuardianSetState,
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
	ibcFeaturesFunc func() string,
	gatewayRelayerEnabled bool,
	ccqEnabled bool,
	signedQueryReqC chan<- *gossipv1.SignedQueryRequest,
	queryResponseReadC <-chan *query.QueryResponsePublication,
	ccqBootstrapPeers string,
	ccqPort uint,
	ccqAllowedPeers string,
) func(ctx context.Context) error {
	if components == nil {
		components = DefaultComponents()
	}

	return func(ctx context.Context) error {
		p2pReceiveChannelOverflow.WithLabelValues("observation").Add(0)
		p2pReceiveChannelOverflow.WithLabelValues("signed_vaa_with_quorum").Add(0)
		p2pReceiveChannelOverflow.WithLabelValues("signed_observation_request").Add(0)

		logger := supervisor.Logger(ctx)

		defer func() {
			// TODO: Right now we're canceling the root context because it used to be the case that libp2p cannot be cleanly restarted.
			// But that seems to no longer be the case. We may want to revisit this. See (https://github.com/libp2p/go-libp2p/issues/992) for background.
			logger.Warn("p2p routine has exited, cancelling root context...")
			rootCtxCancel()
		}()

		h, err := NewHost(logger, ctx, networkID, bootstrapPeers, components, priv)
		if err != nil {
			panic(err)
		}

		defer func() {
			if err := h.Close(); err != nil {
				logger.Error("error closing the host", zap.Error(err))
			}
		}()

		nodeIdBytes, err := h.ID().Marshal()
		if err != nil {
			panic(err)
		}

		topic := fmt.Sprintf("%s/%s", networkID, "broadcast")

		bootstrappers, bootstrapNode := bootstrapAddrs(logger, bootstrapPeers, h.ID())
		gossipParams := pubsub.DefaultGossipSubParams()

		if bootstrapNode {
			logger.Info("We are a bootstrap node.")
			if networkID == "/wormhole/testnet/2/1" {
				gossipParams.Dhi = TESTNET_BOOTSTRAP_DHI
				logger.Info("We are a bootstrap node in Testnet. Setting gossipParams.Dhi.", zap.Int("gossipParams.Dhi", gossipParams.Dhi))
			}
		}

		logger.Info("Subscribing pubsub topic", zap.String("topic", topic))
		ps, err := pubsub.NewGossipSub(ctx, h,
			pubsub.WithValidateQueueSize(P2P_VALIDATE_QUEUE_SIZE),
			pubsub.WithGossipSubParams(gossipParams),
		)
		if err != nil {
			panic(err)
		}

		th, err := ps.Join(topic)
		if err != nil {
			return fmt.Errorf("failed to join topic: %w", err)
		}

		defer func() {
			if err := th.Close(); err != nil && !errors.Is(err, context.Canceled) {
				logger.Error("Error closing the topic", zap.Error(err))
			}
		}()

		// Increase the buffer size to prevent failed delivery
		// to slower subscribers
		sub, err := th.Subscribe(pubsub.WithBufferSize(P2P_SUBSCRIPTION_BUFFER_SIZE))
		if err != nil {
			return fmt.Errorf("failed to subscribe topic: %w", err)
		}
		defer sub.Cancel()

		// Make sure we connect to at least 1 bootstrap node (this is particularly important in a local devnet and CI
		// as peer discovery can take a long time).

		successes := connectToPeers(ctx, logger, h, bootstrappers)

		if successes == 0 && !bootstrapNode { // If we're a bootstrap node it's okay to not have any peers.
			// If we fail to connect to any bootstrap peer, kill the service
			// returning from this function will lead to rootCtxCancel() being called in the defer() above. The service will then be restarted by Tilt/kubernetes.
			return fmt.Errorf("failed to connect to any bootstrap peer")
		}
		logger.Info("Connected to bootstrap peers", zap.Int("num", successes))

		logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
			zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))

		bootTime := time.Now()

		if ccqEnabled {
			ccqErrC := make(chan error)
			ccq := newCcqRunP2p(logger, ccqAllowedPeers)
			if err := ccq.run(ctx, priv, gk, networkID, ccqBootstrapPeers, ccqPort, signedQueryReqC, queryResponseReadC, ccqErrC); err != nil {
				return fmt.Errorf("failed to start p2p for CCQ: %w", err)
			}
			defer ccq.close()
			go func() {
				for {
					select {
					case <-ctx.Done():
						return
					case ccqErr := <-ccqErrC:
						logger.Error("ccqp2p returned an error", zap.Error(ccqErr), zap.String("component", "ccqp2p"))
						rootCtxCancel()
						return
					}
				}
			}()
		}

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
			// Guardians should send out their first heartbeat immediately to speed up test runs.
			// But we also want to wait a little bit such that network connections can be established by then.
			timer := time.NewTimer(time.Second * 2)
			defer timer.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
					timer.Reset(15 * time.Second)

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
						if ibcFeaturesFunc != nil {
							ibcFlags := ibcFeaturesFunc()
							if ibcFlags != "" {
								features = append(features, ibcFlags)
							}
						}
						if gatewayRelayerEnabled {
							features = append(features, "gwrelayer")
						}
						if ccqEnabled {
							features = append(features, "ccq")
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
			envelope, err := sub.Next(ctx) // Note: sub.Next(ctx) will return an error once ctx is canceled
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
					logger.Log(components.SignedHeartbeatLogLevel, "skipping heartbeat - no guardian set",
						zap.Any("value", s),
						zap.String("from", envelope.GetFrom().String()))
					break
				}
				if heartbeat, err := processSignedHeartbeat(envelope.GetFrom(), s, gs, gst, disableHeartbeatVerify); err != nil {
					p2pMessagesReceived.WithLabelValues("invalid_heartbeat").Inc()
					logger.Log(components.SignedHeartbeatLogLevel, "invalid signed heartbeat received",
						zap.Error(err),
						zap.Any("payload", msg.Message),
						zap.Any("value", s),
						zap.Binary("raw", envelope.Data),
						zap.String("from", envelope.GetFrom().String()))
				} else {
					p2pMessagesReceived.WithLabelValues("valid_heartbeat").Inc()
					logger.Log(components.SignedHeartbeatLogLevel, "valid signed heartbeat received",
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
								guardianAddr := eth_common.BytesToAddress(s.GuardianAddr)
								if guardianAddr != ethcrypto.PubkeyToAddress(gk.PublicKey) {
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
							}
						} else {
							logger.Debug("p2p_node_id_not_in_heartbeat",
								zap.Error(err),
								zap.Any("payload", heartbeat.NodeName))
						}
					}()
				}
			case *gossipv1.GossipMessage_SignedObservation:
				if err := common.PostMsgWithTimestamp[gossipv1.SignedObservation](m.SignedObservation, obsvC); err == nil {
					p2pMessagesReceived.WithLabelValues("observation").Inc()
				} else {
					if components.WarnChannelOverflow {
						logger.Warn("Ignoring SignedObservation because obsvC full", zap.String("hash", hex.EncodeToString(m.SignedObservation.Hash)))
					}
					p2pReceiveChannelOverflow.WithLabelValues("observation").Inc()
				}
			case *gossipv1.GossipMessage_SignedVaaWithQuorum:
				select {
				case signedInC <- m.SignedVaaWithQuorum:
					p2pMessagesReceived.WithLabelValues("signed_vaa_with_quorum").Inc()
				default:
					if components.WarnChannelOverflow {
						// TODO do not log this in production
						var hexStr string
						if vaa, err := vaa.Unmarshal(m.SignedVaaWithQuorum.Vaa); err == nil {
							hexStr = vaa.HexDigest()
						}
						logger.Warn("Ignoring SignedVaaWithQuorum because signedInC full", zap.String("hash", hexStr))
					}
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
					logger.Debug("valid signed observation request received",
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

func processSignedHeartbeat(from peer.ID, s *gossipv1.SignedHeartbeat, gs *common.GuardianSet, gst *common.GuardianSetState, disableVerify bool) (*gossipv1.Heartbeat, error) {
	envelopeAddr := eth_common.BytesToAddress(s.GuardianAddr)
	idx, ok := gs.KeyIndex(envelopeAddr)
	var pk eth_common.Address
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

	signerAddr := eth_common.BytesToAddress(ethcrypto.Keccak256(pubKey[1:])[12:])
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

func processSignedObservationRequest(s *gossipv1.SignedObservationRequest, gs *common.GuardianSet) (*gossipv1.ObservationRequest, error) {
	envelopeAddr := eth_common.BytesToAddress(s.GuardianAddr)
	idx, ok := gs.KeyIndex(envelopeAddr)
	var pk eth_common.Address
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

	signerAddr := eth_common.BytesToAddress(ethcrypto.Keccak256(pubKey[1:])[12:])
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
