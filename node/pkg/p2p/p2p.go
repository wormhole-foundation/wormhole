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

	"github.com/certusone/wormhole/node/pkg/common"
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
	libp2ppb "github.com/libp2p/go-libp2p-pubsub/pb"
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
	p2pDrop = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_drops",
			Help: "Total number of messages that were dropped by libp2p",
		})
	p2pReject = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_rejects",
			Help: "Total number of messages rejected by libp2p",
		})
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
	// GossipParams is used to configure the GossipSub instance used by the Guardian.
	GossipParams pubsub.GossipSubParams
	// GossipAdvertiseAddress is an override for the external IP advertised via p2p to other peers.
	GossipAdvertiseAddress string
}

func (f *Components) ListeningAddresses() []string {
	la := make([]string, 0, len(f.ListeningAddressesPatterns))
	for _, pattern := range f.ListeningAddressesPatterns {
		pattern = cutOverAddressPattern(pattern)
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
		GossipParams:               pubsub.DefaultGossipSubParams(),
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

// traceHandler is used to intercept libp2p trace events so we can peg metrics.
type traceHandler struct {
}

// Trace is the interface to the libp2p trace handler. It pegs metrics as appropriate.
func (*traceHandler) Trace(evt *libp2ppb.TraceEvent) {
	if evt.Type != nil {
		if *evt.Type == libp2ppb.TraceEvent_DROP_RPC {
			p2pDrop.Inc()
		} else if *evt.Type == libp2ppb.TraceEvent_REJECT_MESSAGE {
			p2pReject.Inc()
		}
	}
}

// BootstrapAddrs takes a comma-separated string of multi-address strings and returns an array of []peer.AddrInfo that does not include `self`.
// if `self` is part of `bootstrapPeers`, return isBootstrapNode=true
func BootstrapAddrs(logger *zap.Logger, bootstrapPeers string, self peer.ID) (bootstrappers []peer.AddrInfo, isBootstrapNode bool) {
	bootstrapPeers = cutOverBootstrapPeers(bootstrapPeers)
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

// ConnectToPeers connects `h` to `peers` and returns the number of successful connections.
func ConnectToPeers(ctx context.Context, logger *zap.Logger, h host.Host, peers []peer.AddrInfo) (successes int) {
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

	// if an override of the advertised gossip addresses is requested
	// check & render address once for use in the AddrsFactory below
	var gossipAdvertiseAddress multiaddr.Multiaddr
	if components.GossipAdvertiseAddress != "" {
		gossipAdvertiseAddress, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/%s/udp/%d", components.GossipAdvertiseAddress, components.Port))
		if err != nil {
			// If the multiaddr is specified incorrectly, blow up
			logger.Fatal("error with the specified gossip address",
				zap.String("GossipAdvertiseAddress", components.GossipAdvertiseAddress),
				zap.Error(err),
			)
		}
		logger.Info("Overriding the advertised p2p address",
			zap.String("GossipAdvertiseAddress", gossipAdvertiseAddress.String()),
		)
	}

	h, err := libp2p.New(
		// Use the keypair we generated
		libp2p.Identity(priv),

		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			components.ListeningAddresses()...,
		),

		// Takes the multiaddrs we are listening on and returns the multiaddrs to advertise to the network to
		// connect to. Allows overriding the announce address for nodes running behind a NAT or in kubernetes
		// This function gets called by the libp2p background() process regularly to check for address changes
		// that are then announced to the rest of the network.
		libp2p.AddrsFactory(func(addrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
			if gossipAdvertiseAddress != nil {
				return []multiaddr.Multiaddr{gossipAdvertiseAddress}
			}
			return addrs
		}),

		// Enable TLS security as the only security protocol.
		libp2p.Security(libp2ptls.ID, libp2ptls.New),

		// Enable QUIC transport as the only transport.
		libp2p.Transport(libp2pquic.NewTransport),

		// Let's prevent our peer from having too many
		// connections by attaching a connection manager.
		libp2p.ConnectionManager(components.ConnMgr),

		// Let this host use the DHT to find other hosts
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			// Update the bootstrap peers string so we will log the updated value.
			bootstrapPeers = cutOverBootstrapPeers(bootstrapPeers)
			logger.Info("Connecting to bootstrap peers", zap.String("bootstrap_peers", bootstrapPeers))

			bootstrappers, _ := BootstrapAddrs(logger, bootstrapPeers, h.ID())

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

func Run(params *RunParams) func(ctx context.Context) error {
	if params == nil {
		return func(ctx context.Context) error {
			return errors.New("params may not be nil")
		}
	}
	if params.components == nil {
		params.components = DefaultComponents()
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
			params.rootCtxCancel()
		}()

		h, err := NewHost(logger, ctx, params.networkID, params.bootstrapPeers, params.components, params.priv)
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

		topic := fmt.Sprintf("%s/%s", params.networkID, "broadcast")

		bootstrappers, bootstrapNode := BootstrapAddrs(logger, params.bootstrapPeers, h.ID())

		if bootstrapNode {
			logger.Info("We are a bootstrap node.")
			if params.networkID == "/wormhole/testnet/2/1" {
				params.components.GossipParams.Dhi = TESTNET_BOOTSTRAP_DHI
				logger.Info("We are a bootstrap node in Testnet. Setting gossipParams.Dhi.", zap.Int("gossipParams.Dhi", params.components.GossipParams.Dhi))
			}
		}

		logger.Info("Subscribing pubsub topic", zap.String("topic", topic))
		ourTracer := &traceHandler{}
		ps, err := pubsub.NewGossipSub(ctx, h,
			pubsub.WithValidateQueueSize(P2P_VALIDATE_QUEUE_SIZE),
			pubsub.WithGossipSubParams(params.components.GossipParams),
			pubsub.WithEventTracer(ourTracer),
			// TODO: Investigate making this change. May need to use LaxSign until everyone has upgraded to that.
			// pubsub.WithMessageSignaturePolicy(pubsub.StrictNoSign),
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

		successes := ConnectToPeers(ctx, logger, h, bootstrappers)

		if successes == 0 && !bootstrapNode { // If we're a bootstrap node it's okay to not have any peers.
			// If we fail to connect to any bootstrap peer, kill the service
			// returning from this function will lead to rootCtxCancel() being called in the defer() above. The service will then be restarted by Tilt/kubernetes.
			return fmt.Errorf("failed to connect to any bootstrap peer")
		}
		logger.Info("Connected to bootstrap peers", zap.Int("num", successes))

		logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
			zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))

		bootTime := time.Now()

		if params.ccqEnabled {
			ccqErrC := make(chan error)
			ccq := newCcqRunP2p(logger, params.ccqAllowedPeers, params.components)
			if err := ccq.run(ctx, params.priv, params.gk, params.networkID, params.ccqBootstrapPeers, params.ccqPort, params.signedQueryReqC, params.queryResponseReadC, ccqErrC); err != nil {
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
						params.rootCtxCancel()
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
					params.gst.Cleanup()
				case <-ctx.Done():
					return
				}
			}
		}()

		go func() {
			// Disable heartbeat when no node name is provided (spy mode)
			if params.nodeName == "" {
				return
			}
			ourAddr := ethcrypto.PubkeyToAddress(params.gk.PublicKey)

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
						if params.gov != nil {
							features = append(features, "governor")
						}
						if params.acct != nil {
							features = append(features, params.acct.FeatureString())
						}
						if params.ibcFeaturesFunc != nil {
							ibcFlags := params.ibcFeaturesFunc()
							if ibcFlags != "" {
								features = append(features, ibcFlags)
							}
						}
						if params.gatewayRelayerEnabled {
							features = append(features, "gwrelayer")
						}
						if params.ccqEnabled {
							features = append(features, "ccq")
						}

						heartbeat := &gossipv1.Heartbeat{
							NodeName:      params.nodeName,
							Counter:       ctr,
							Timestamp:     time.Now().UnixNano(),
							Networks:      networks,
							Version:       version.Version(),
							GuardianAddr:  ourAddr.String(),
							BootTimestamp: bootTime.UnixNano(),
							Features:      features,
						}

						if params.components.P2PIDInHeartbeat {
							heartbeat.P2PNodeId = nodeIdBytes
						}

						if err := params.gst.SetHeartbeat(ourAddr, h.ID(), heartbeat); err != nil {
							panic(err)
						}
						collectNodeMetrics(ourAddr, h.ID(), heartbeat)

						if params.gov != nil {
							params.gov.CollectMetrics(heartbeat, params.gossipSendC, params.gk, ourAddr)
						}

						msg := gossipv1.GossipMessage{
							Message: &gossipv1.GossipMessage_SignedHeartbeat{
								SignedHeartbeat: createSignedHeartbeat(params.gk, heartbeat),
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
				case msg := <-params.gossipSendC:
					err := th.Publish(ctx, msg)
					p2pMessagesSent.Inc()
					if err != nil {
						logger.Error("failed to publish message from queue", zap.Error(err))
					}
				case msg := <-params.obsvReqSendC:
					b, err := proto.Marshal(msg)
					if err != nil {
						panic(err)
					}

					// Sign the observation request using our node's guardian key.
					digest := signedObservationRequestDigest(b)
					sig, err := ethcrypto.Sign(digest.Bytes(), params.gk)
					if err != nil {
						panic(err)
					}

					sReq := &gossipv1.SignedObservationRequest{
						ObservationRequest: b,
						Signature:          sig,
						GuardianAddr:       ethcrypto.PubkeyToAddress(params.gk.PublicKey).Bytes(),
					}

					envelope := &gossipv1.GossipMessage{
						Message: &gossipv1.GossipMessage_SignedObservationRequest{
							SignedObservationRequest: sReq}}

					b, err = proto.Marshal(envelope)
					if err != nil {
						panic(err)
					}

					// Send to local observation request queue (the loopback message is ignored)
					if params.obsvReqC != nil {
						params.obsvReqC <- msg
					}

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
				if logger.Level().Enabled(zapcore.DebugLevel) {
					logger.Debug("received message from ourselves, ignoring", zap.Any("payload", msg.Message))
				}
				p2pMessagesReceived.WithLabelValues("loopback").Inc()
				continue
			}

			if logger.Level().Enabled(zapcore.DebugLevel) {
				logger.Debug("received message",
					zap.Any("payload", msg.Message),
					zap.Binary("raw", envelope.Data),
					zap.String("from", envelope.GetFrom().String()))
			}

			switch m := msg.Message.(type) {
			case *gossipv1.GossipMessage_SignedHeartbeat:
				s := m.SignedHeartbeat
				gs := params.gst.Get()
				if gs == nil {
					// No valid guardian set yet - dropping heartbeat
					logger.Log(params.components.SignedHeartbeatLogLevel, "skipping heartbeat - no guardian set",
						zap.Any("value", s),
						zap.String("from", envelope.GetFrom().String()))
					break
				}
				if heartbeat, err := processSignedHeartbeat(envelope.GetFrom(), s, gs, params.gst, params.disableHeartbeatVerify); err != nil {
					p2pMessagesReceived.WithLabelValues("invalid_heartbeat").Inc()
					logger.Log(params.components.SignedHeartbeatLogLevel, "invalid signed heartbeat received",
						zap.Error(err),
						zap.Any("payload", msg.Message),
						zap.Any("value", s),
						zap.Binary("raw", envelope.Data),
						zap.String("from", envelope.GetFrom().String()))
				} else {
					p2pMessagesReceived.WithLabelValues("valid_heartbeat").Inc()
					logger.Log(params.components.SignedHeartbeatLogLevel, "valid signed heartbeat received",
						zap.Any("value", heartbeat),
						zap.String("from", envelope.GetFrom().String()))

					func() {
						if len(heartbeat.P2PNodeId) != 0 {
							params.components.ProtectedHostByGuardianKeyLock.Lock()
							defer params.components.ProtectedHostByGuardianKeyLock.Unlock()
							var peerId peer.ID
							if err = peerId.Unmarshal(heartbeat.P2PNodeId); err != nil {
								logger.Error("p2p_node_id_in_heartbeat_invalid",
									zap.Any("payload", msg.Message),
									zap.Any("value", s),
									zap.Binary("raw", envelope.Data),
									zap.String("from", envelope.GetFrom().String()))
							} else {
								guardianAddr := eth_common.BytesToAddress(s.GuardianAddr)
								if params.gk == nil || guardianAddr != ethcrypto.PubkeyToAddress(params.gk.PublicKey) {
									prevPeerId, ok := params.components.ProtectedHostByGuardianKey[guardianAddr]
									if ok {
										if prevPeerId != peerId {
											logger.Info("p2p_guardian_peer_changed",
												zap.String("guardian_addr", guardianAddr.String()),
												zap.String("prevPeerId", prevPeerId.String()),
												zap.String("newPeerId", peerId.String()),
											)
											params.components.ConnMgr.Unprotect(prevPeerId, "heartbeat")
											params.components.ConnMgr.Protect(peerId, "heartbeat")
											params.components.ProtectedHostByGuardianKey[guardianAddr] = peerId
										}
									} else {
										params.components.ConnMgr.Protect(peerId, "heartbeat")
										params.components.ProtectedHostByGuardianKey[guardianAddr] = peerId
									}
								}
							}
						} else {
							if logger.Level().Enabled(zapcore.DebugLevel) {
								logger.Debug("p2p_node_id_not_in_heartbeat", zap.Error(err), zap.Any("payload", heartbeat.NodeName))
							}
						}
					}()
				}
			case *gossipv1.GossipMessage_SignedObservation:
				if params.obsvC != nil {
					if err := common.PostMsgWithTimestamp[gossipv1.SignedObservation](m.SignedObservation, params.obsvC); err == nil {
						p2pMessagesReceived.WithLabelValues("observation").Inc()
					} else {
						if params.components.WarnChannelOverflow {
							logger.Warn("Ignoring SignedObservation because obsvC full", zap.String("hash", hex.EncodeToString(m.SignedObservation.Hash)))
						}
						p2pReceiveChannelOverflow.WithLabelValues("observation").Inc()
					}
				}
			case *gossipv1.GossipMessage_SignedVaaWithQuorum:
				if params.signedInC != nil {
					select {
					case params.signedInC <- m.SignedVaaWithQuorum:
						p2pMessagesReceived.WithLabelValues("signed_vaa_with_quorum").Inc()
					default:
						if params.components.WarnChannelOverflow {
							// TODO do not log this in production
							var hexStr string
							if vaa, err := vaa.Unmarshal(m.SignedVaaWithQuorum.Vaa); err == nil {
								hexStr = vaa.HexDigest()
							}
							logger.Warn("Ignoring SignedVaaWithQuorum because signedInC full", zap.String("hash", hexStr))
						}
						p2pReceiveChannelOverflow.WithLabelValues("signed_vaa_with_quorum").Inc()
					}
				}
			case *gossipv1.GossipMessage_SignedObservationRequest:
				if params.obsvReqC != nil {
					s := m.SignedObservationRequest
					gs := params.gst.Get()
					if gs == nil {
						if logger.Level().Enabled(zapcore.DebugLevel) {
							logger.Debug("dropping SignedObservationRequest - no guardian set", zap.Any("value", s), zap.String("from", envelope.GetFrom().String()))
						}
						break
					}
					r, err := processSignedObservationRequest(s, gs)
					if err != nil {
						p2pMessagesReceived.WithLabelValues("invalid_signed_observation_request").Inc()
						if logger.Level().Enabled(zapcore.DebugLevel) {
							logger.Debug("invalid signed observation request received",
								zap.Error(err),
								zap.Any("payload", msg.Message),
								zap.Any("value", s),
								zap.Binary("raw", envelope.Data),
								zap.String("from", envelope.GetFrom().String()))
						}
					} else {
						if logger.Level().Enabled(zapcore.DebugLevel) {
							logger.Debug("valid signed observation request received", zap.Any("value", r), zap.String("from", envelope.GetFrom().String()))
						}

						select {
						case params.obsvReqC <- r:
							p2pMessagesReceived.WithLabelValues("signed_observation_request").Inc()
						default:
							p2pReceiveChannelOverflow.WithLabelValues("signed_observation_request").Inc()
						}
					}
				}
			case *gossipv1.GossipMessage_SignedChainGovernorConfig:
				if params.signedGovCfg != nil {
					params.signedGovCfg <- m.SignedChainGovernorConfig
				}
			case *gossipv1.GossipMessage_SignedChainGovernorStatus:
				if params.signedGovSt != nil {
					params.signedGovSt <- m.SignedChainGovernorStatus
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

	// Don't accept replayed heartbeats from other peers
	signedPeer, err := peer.IDFromBytes(h.P2PNodeId)
	if err != nil {
		return nil, fmt.Errorf("failed to decode peer ID from bytes: %w", err)
	}
	if signedPeer != from {
		return nil, fmt.Errorf("guardian signed peer does not match sending peer")
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
