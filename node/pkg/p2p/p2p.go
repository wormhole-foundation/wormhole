package p2p

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/guardiansigner"
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

// MaxObservationBatchSize is the maximum number of observations that will fit in a single `SignedObservationBatch` message.
const MaxObservationBatchSize = 4000

// MaxObservationBatchDelay is the longest we will wait before publishing any queued up observations.
const MaxObservationBatchDelay = time.Second

var (
	p2pHeartbeatsSent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_heartbeats_sent_total",
			Help: "Total number of p2p heartbeats sent",
		})
	p2pMessagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_broadcast_messages_sent_total",
			Help: "Total number of p2p pubsub broadcast messages sent",
		}, []string{"type"})
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
	// WARNING: This should not be enabled in production. It is only used in node tests to watch for overflows.
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

	// The default libp2p options.
	opts := []libp2p.Option{
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
	}

	// If the external IP to advertise is known ahead of time, disable address discovery.
	if gossipAdvertiseAddress != nil {
		opts = append(opts, libp2p.DisableIdentifyAddressDiscovery())
	}

	return libp2p.New(opts...)
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
		p2pMessagesSent.WithLabelValues("control").Add(0)
		p2pMessagesSent.WithLabelValues("attestation").Add(0)
		p2pMessagesSent.WithLabelValues("vaa").Add(0)
		p2pReceiveChannelOverflow.WithLabelValues("observation").Add(0)
		p2pReceiveChannelOverflow.WithLabelValues("batch_observation").Add(0)
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

		if len(params.protectedPeers) != 0 {
			for _, peerId := range params.protectedPeers {
				logger.Info("protecting peer", zap.String("peerId", peerId))
				params.components.ConnMgr.Protect(peer.ID(peerId), "configured")
			}
		}

		nodeIdBytes, err := h.ID().Marshal()
		if err != nil {
			panic(err)
		}

		bootstrappers, bootstrapNode := BootstrapAddrs(logger, params.bootstrapPeers, h.ID())

		if bootstrapNode {
			logger.Info("We are a bootstrap node.")
			if params.networkID == "/wormhole/testnet/2/1" {
				params.components.GossipParams.Dhi = TESTNET_BOOTSTRAP_DHI
				logger.Info("We are a bootstrap node in Testnet. Setting gossipParams.Dhi.", zap.Int("gossipParams.Dhi", params.components.GossipParams.Dhi))
			}
		}

		logger.Info("connecting to pubsub")
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

		// These will only be non-nil if the application plans to listen for or publish to that topic.
		var controlPubsubTopic, attestationPubsubTopic, vaaPubsubTopic *pubsub.Topic
		var controlSubscription, attestationSubscription, vaaSubscription *pubsub.Subscription

		// Set up the control channel. ////////////////////////////////////////////////////////////////////
		if params.nodeName != "" || params.gossipControlSendC != nil || params.obsvReqSendC != nil || params.obsvReqRecvC != nil || params.signedGovCfgRecvC != nil || params.signedGovStatusRecvC != nil || params.gst.IsSubscribedToHeartbeats() {
			controlTopic := fmt.Sprintf("%s/%s", params.networkID, "control")
			logger.Info("joining the control topic", zap.String("topic", controlTopic))
			controlPubsubTopic, err = ps.Join(controlTopic)
			if err != nil {
				return fmt.Errorf("failed to join the control topic: %w", err)
			}

			defer func() {
				if err := controlPubsubTopic.Close(); err != nil && !errors.Is(err, context.Canceled) {
					logger.Error("Error closing the control topic", zap.Error(err))
				}
			}()

			if params.obsvReqRecvC != nil || params.signedGovCfgRecvC != nil || params.signedGovStatusRecvC != nil || params.gst.IsSubscribedToHeartbeats() {
				logger.Info("subscribing to the control topic", zap.String("topic", controlTopic))
				controlSubscription, err = controlPubsubTopic.Subscribe(pubsub.WithBufferSize(P2P_SUBSCRIPTION_BUFFER_SIZE))
				if err != nil {
					return fmt.Errorf("failed to subscribe to the control topic: %w", err)
				}
				defer controlSubscription.Cancel()
			}
		}

		// Set up the attestation channel. ////////////////////////////////////////////////////////////////////
		if params.gossipAttestationSendC != nil || params.batchObsvRecvC != nil {
			attestationTopic := fmt.Sprintf("%s/%s", params.networkID, "attestation")
			logger.Info("joining the attestation topic", zap.String("topic", attestationTopic))
			attestationPubsubTopic, err = ps.Join(attestationTopic)
			if err != nil {
				return fmt.Errorf("failed to join the attestation topic: %w", err)
			}

			defer func() {
				if err := attestationPubsubTopic.Close(); err != nil && !errors.Is(err, context.Canceled) {
					logger.Error("Error closing the attestation topic", zap.Error(err))
				}
			}()

			if params.batchObsvRecvC != nil {
				logger.Info("subscribing to the attestation topic", zap.String("topic", attestationTopic))
				attestationSubscription, err = attestationPubsubTopic.Subscribe(pubsub.WithBufferSize(P2P_SUBSCRIPTION_BUFFER_SIZE))
				if err != nil {
					return fmt.Errorf("failed to subscribe to the attestation topic: %w", err)
				}
				defer attestationSubscription.Cancel()
			}
		}

		// Set up the VAA channel. ////////////////////////////////////////////////////////////////////
		if params.gossipVaaSendC != nil || params.signedIncomingVaaRecvC != nil {
			vaaTopic := fmt.Sprintf("%s/%s", params.networkID, "broadcast")
			logger.Info("joining the vaa topic", zap.String("topic", vaaTopic))
			vaaPubsubTopic, err = ps.Join(vaaTopic)
			if err != nil {
				return fmt.Errorf("failed to join the vaa topic: %w", err)
			}

			defer func() {
				if err := vaaPubsubTopic.Close(); err != nil && !errors.Is(err, context.Canceled) {
					logger.Error("Error closing the vaa topic", zap.Error(err))
				}
			}()

			if params.signedIncomingVaaRecvC != nil {
				logger.Info("subscribing to the vaa topic", zap.String("topic", vaaTopic))
				vaaSubscription, err = vaaPubsubTopic.Subscribe(pubsub.WithBufferSize(P2P_SUBSCRIPTION_BUFFER_SIZE))
				if err != nil {
					return fmt.Errorf("failed to subscribe to the vaa topic: %w", err)
				}
				defer vaaSubscription.Cancel()
			}
		}

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
			if err := ccq.run(ctx, params.priv, params.guardianSigner, params.networkID, params.ccqBootstrapPeers, params.ccqPort, params.signedQueryReqC, params.queryResponseReadC, params.ccqProtectedPeers, ccqErrC); err != nil {
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

		// Start up heartbeating if it is enabled.
		if params.nodeName != "" {
			go func() {
				ourAddr := ethcrypto.PubkeyToAddress(params.guardianSigner.PublicKey(ctx))

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
								errCtr := DefaultRegistry.GetErrorCount(vaa.ChainID(v.Id)) // #nosec G115 -- This is safe as chain id is constrained in SetNetworkStats
								v.ErrorCount = errCtr
								networks = append(networks, v)
							}

							features := make([]string, 0)
							if params.processorFeaturesFunc != nil {
								flag := params.processorFeaturesFunc()
								if flag != "" {
									features = append(features, flag)
								}
							}
							if params.gov != nil {
								if params.gov.IsFlowCancelEnabled() {
									features = append(features, "governor:fc")
								} else {
									features = append(features, "governor")
								}
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
							if len(params.featureFlags) != 0 {
								features = append(features, params.featureFlags...)
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
								params.gov.CollectMetrics(ctx, heartbeat, params.gossipControlSendC, params.guardianSigner, ourAddr)
							}

							msg := gossipv1.GossipMessage{
								Message: &gossipv1.GossipMessage_SignedHeartbeat{
									SignedHeartbeat: createSignedHeartbeat(ctx, params.guardianSigner, heartbeat),
								},
							}

							b, err := proto.Marshal(&msg)
							if err != nil {
								panic(err)
							}
							return b
						}()

						if controlPubsubTopic == nil {
							panic("controlPubsubTopic should not be nil when nodeName is set")
						}
						err = controlPubsubTopic.Publish(ctx, b)
						p2pMessagesSent.WithLabelValues("control").Inc()
						if err != nil {
							logger.Warn("failed to publish heartbeat message", zap.Error(err))
						}

						p2pHeartbeatsSent.Inc()
						ctr += 1
					}
				}
			}()
		}

		// This routine processes messages received from the internal channels and publishes them to gossip. ///////////////////
		// NOTE: The go specification says that it is safe to receive on a nil channel, it just blocks forever.
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-params.gossipControlSendC:
					if controlPubsubTopic == nil {
						panic("controlPubsubTopic should not be nil when gossipControlSendC is set")
					}
					err := controlPubsubTopic.Publish(ctx, msg)
					p2pMessagesSent.WithLabelValues("control").Inc()
					if err != nil {
						logger.Error("failed to publish message from control queue", zap.Error(err))
					}
				case msg := <-params.gossipAttestationSendC:
					if attestationPubsubTopic == nil {
						panic("attestationPubsubTopic should not be nil when gossipAttestationSendC is set")
					}
					err := attestationPubsubTopic.Publish(ctx, msg)
					p2pMessagesSent.WithLabelValues("attestation").Inc()
					if err != nil {
						logger.Error("failed to publish message from attestation queue", zap.Error(err))
					}
				case msg := <-params.gossipVaaSendC:
					if vaaPubsubTopic == nil {
						panic("vaaPubsubTopic should not be nil when gossipVaaSendC is set")
					}
					err := vaaPubsubTopic.Publish(ctx, msg)
					p2pMessagesSent.WithLabelValues("vaa").Inc()
					if err != nil {
						logger.Error("failed to publish message from vaa queue", zap.Error(err))
					}
				case msg := <-params.obsvReqSendC:
					b, err := proto.Marshal(msg)
					if err != nil {
						panic(err)
					}

					// Sign the observation request using our node's guardian key.
					digest := signedObservationRequestDigest(b)
					sig, err := params.guardianSigner.Sign(ctx, digest.Bytes())
					if err != nil {
						panic(err)
					}

					sReq := &gossipv1.SignedObservationRequest{
						ObservationRequest: b,
						Signature:          sig,
						GuardianAddr:       ethcrypto.PubkeyToAddress(params.guardianSigner.PublicKey(ctx)).Bytes(),
					}

					envelope := &gossipv1.GossipMessage{
						Message: &gossipv1.GossipMessage_SignedObservationRequest{
							SignedObservationRequest: sReq}}

					b, err = proto.Marshal(envelope)
					if err != nil {
						panic(err)
					}

					// Send to local observation request queue (the loopback message is ignored)
					if params.obsvReqRecvC != nil {
						params.obsvReqRecvC <- msg
					}

					if controlPubsubTopic == nil {
						panic("controlPubsubTopic should not be nil when obsvReqSendC is set")
					}
					err = controlPubsubTopic.Publish(ctx, b)
					p2pMessagesSent.WithLabelValues("control").Inc()
					if err != nil {
						logger.Error("failed to publish observation request", zap.Error(err))
					} else {
						logger.Info("published signed observation request", zap.Any("signed_observation_request", sReq))
					}
				}
			}
		}()

		errC := make(chan error)

		// This routine processes control messages received from gossip. //////////////////////////////////////////////
		if controlSubscription != nil {
			go func() {
				for {
					envelope, err := controlSubscription.Next(ctx) // Note: sub.Next(ctx) will return an error once ctx is canceled
					if err != nil {
						errC <- fmt.Errorf("failed to receive pubsub message on control topic: %w", err)
						return
					}

					var msg gossipv1.GossipMessage
					err = proto.Unmarshal(envelope.Data, &msg)
					if err != nil {
						logger.Info("received invalid message on control topic",
							zap.Binary("data", envelope.Data),
							zap.String("from", envelope.GetFrom().String()))
						p2pMessagesReceived.WithLabelValues("invalid").Inc()
						continue
					}

					if envelope.GetFrom() == h.ID() {
						if logger.Level().Enabled(zapcore.DebugLevel) {
							logger.Debug("received message from ourselves on control topic, ignoring", zap.Any("payload", msg.Message))
						}
						p2pMessagesReceived.WithLabelValues("loopback").Inc()
						continue
					}

					if logger.Level().Enabled(zapcore.DebugLevel) {
						logger.Debug("received message on control topic",
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
							if logger.Level().Enabled(params.components.SignedHeartbeatLogLevel) {
								logger.Log(params.components.SignedHeartbeatLogLevel, "skipping heartbeat - no guardian set",
									zap.Any("value", s),
									zap.String("from", envelope.GetFrom().String()))
							}
							break
						}
						if heartbeat, err := processSignedHeartbeat(envelope.GetFrom(), s, gs, params.gst, params.disableHeartbeatVerify); err != nil {
							p2pMessagesReceived.WithLabelValues("invalid_heartbeat").Inc()
							if logger.Level().Enabled(params.components.SignedHeartbeatLogLevel) {
								logger.Log(params.components.SignedHeartbeatLogLevel, "invalid signed heartbeat received",
									zap.Error(err),
									zap.Any("payload", msg.Message),
									zap.Any("value", s),
									zap.Binary("raw", envelope.Data),
									zap.String("from", envelope.GetFrom().String()))
							}
						} else {
							p2pMessagesReceived.WithLabelValues("valid_heartbeat").Inc()
							if logger.Level().Enabled(params.components.SignedHeartbeatLogLevel) {
								logger.Log(params.components.SignedHeartbeatLogLevel, "valid signed heartbeat received",
									zap.Any("value", heartbeat),
									zap.String("from", envelope.GetFrom().String()))
							}

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
										if params.guardianSigner == nil || guardianAddr != ethcrypto.PubkeyToAddress(params.guardianSigner.PublicKey(ctx)) {
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
					case *gossipv1.GossipMessage_SignedObservationRequest:
						if params.obsvReqRecvC != nil {
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
								case params.obsvReqRecvC <- r:
									p2pMessagesReceived.WithLabelValues("signed_observation_request").Inc()
								default:
									p2pReceiveChannelOverflow.WithLabelValues("signed_observation_request").Inc()
								}
							}
						}
					case *gossipv1.GossipMessage_SignedChainGovernorConfig:
						if params.signedGovCfgRecvC != nil {
							params.signedGovCfgRecvC <- m.SignedChainGovernorConfig
						}
					case *gossipv1.GossipMessage_SignedChainGovernorStatus:
						if params.signedGovStatusRecvC != nil {
							params.signedGovStatusRecvC <- m.SignedChainGovernorStatus
						}
					default:
						p2pMessagesReceived.WithLabelValues("unknown").Inc()
						logger.Warn("received unknown message type on control topic (running outdated software?)",
							zap.Any("payload", msg.Message),
							zap.Binary("raw", envelope.Data),
							zap.String("from", envelope.GetFrom().String()))
					}
				}
			}()
		}

		// This routine processes attestation messages received from gossip. //////////////////////////////////////////////
		if attestationSubscription != nil {
			go func() {
				for {
					envelope, err := attestationSubscription.Next(ctx) // Note: sub.Next(ctx) will return an error once ctx is canceled
					if err != nil {
						errC <- fmt.Errorf("failed to receive pubsub message on attestation topic: %w", err)
						return
					}

					var msg gossipv1.GossipMessage
					err = proto.Unmarshal(envelope.Data, &msg)
					if err != nil {
						logger.Info("received invalid message on attestation topic",
							zap.Binary("data", envelope.Data),
							zap.String("from", envelope.GetFrom().String()))
						p2pMessagesReceived.WithLabelValues("invalid").Inc()
						continue
					}

					if envelope.GetFrom() == h.ID() {
						if logger.Level().Enabled(zapcore.DebugLevel) {
							logger.Debug("received message from ourselves on attestation topic, ignoring", zap.Any("payload", msg.Message))
						}
						p2pMessagesReceived.WithLabelValues("loopback").Inc()
						continue
					}

					if logger.Level().Enabled(zapcore.DebugLevel) {
						logger.Debug("received message on attestation topic",
							zap.Any("payload", msg.Message),
							zap.Binary("raw", envelope.Data),
							zap.String("from", envelope.GetFrom().String()))
					}

					switch m := msg.Message.(type) {
					case *gossipv1.GossipMessage_SignedObservationBatch:
						if params.batchObsvRecvC != nil {
							if err := common.PostMsgWithTimestamp(m.SignedObservationBatch, params.batchObsvRecvC); err == nil {
								p2pMessagesReceived.WithLabelValues("batch_observation").Inc()
							} else {
								if params.components.WarnChannelOverflow {
									logger.Warn("Ignoring SignedObservationBatch because batchObsvRecvC is full", zap.String("addr", hex.EncodeToString(m.SignedObservationBatch.Addr)))
								}
								p2pReceiveChannelOverflow.WithLabelValues("batch_observation").Inc()
							}
						}
					default:
						p2pMessagesReceived.WithLabelValues("unknown").Inc()
						logger.Warn("received unknown message type on attestation topic (running outdated software?)",
							zap.Any("payload", msg.Message),
							zap.Binary("raw", envelope.Data),
							zap.String("from", envelope.GetFrom().String()))
					}
				}
			}()
		}

		// This routine processes signed VAA messages received from gossip. //////////////////////////////////////////////
		if vaaSubscription != nil {
			go func() {
				for {
					envelope, err := vaaSubscription.Next(ctx) // Note: sub.Next(ctx) will return an error once ctx is canceled
					if err != nil {
						errC <- fmt.Errorf("failed to receive pubsub message on vaa topic: %w", err)
						return
					}

					var msg gossipv1.GossipMessage
					err = proto.Unmarshal(envelope.Data, &msg)
					if err != nil {
						logger.Info("received invalid message on vaa topic",
							zap.Binary("data", envelope.Data),
							zap.String("from", envelope.GetFrom().String()))
						p2pMessagesReceived.WithLabelValues("invalid").Inc()
						continue
					}

					if envelope.GetFrom() == h.ID() {
						if logger.Level().Enabled(zapcore.DebugLevel) {
							logger.Debug("received message from ourselves on vaa topic, ignoring", zap.Any("payload", msg.Message))
						}
						p2pMessagesReceived.WithLabelValues("loopback").Inc()
						continue
					}

					if logger.Level().Enabled(zapcore.DebugLevel) {
						logger.Debug("received message on vaa topic",
							zap.Any("payload", msg.Message),
							zap.Binary("raw", envelope.Data),
							zap.String("from", envelope.GetFrom().String()))
					}

					switch m := msg.Message.(type) {
					case *gossipv1.GossipMessage_SignedVaaWithQuorum:
						if params.signedIncomingVaaRecvC != nil {
							select {
							case params.signedIncomingVaaRecvC <- m.SignedVaaWithQuorum:
								p2pMessagesReceived.WithLabelValues("signed_vaa_with_quorum").Inc()
							default:
								if params.components.WarnChannelOverflow {
									var hexStr string
									if vaa, err := vaa.Unmarshal(m.SignedVaaWithQuorum.Vaa); err == nil {
										hexStr = vaa.HexDigest()
									}
									logger.Warn("Ignoring SignedVaaWithQuorum because signedIncomingVaaRecvC full", zap.String("hash", hexStr))
								}
								p2pReceiveChannelOverflow.WithLabelValues("signed_vaa_with_quorum").Inc()
							}
						}
					default:
						p2pMessagesReceived.WithLabelValues("unknown").Inc()
						logger.Warn("received unknown message type on vaa topic (running outdated software?)",
							zap.Any("payload", msg.Message),
							zap.Binary("raw", envelope.Data),
							zap.String("from", envelope.GetFrom().String()))
					}
				}
			}()
		}

		// Wait for either a shutdown or a fatal error from a pubsub subscription.
		select {
		case <-ctx.Done():
			return nil
		case err := <-errC:
			return err
		}
	}
}

func createSignedHeartbeat(ctx context.Context, guardianSigner guardiansigner.GuardianSigner, heartbeat *gossipv1.Heartbeat) *gossipv1.SignedHeartbeat {
	ourAddr := ethcrypto.PubkeyToAddress(guardianSigner.PublicKey(ctx))

	b, err := proto.Marshal(heartbeat)
	if err != nil {
		panic(err)
	}

	// Sign the heartbeat using our node's guardian signer.
	digest := heartbeatDigest(b)
	sig, err := guardianSigner.Sign(ctx, digest.Bytes())
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
