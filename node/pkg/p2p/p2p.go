package p2p

import (
	"context"
	"fmt"
	"strings"
	"sync"

	node_common "github.com/certusone/wormhole/node/pkg/common"
	"github.com/ethereum/go-ethereum/common"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

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
	p2pMessagesSent = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_broadcast_messages_sent_total",
			Help: "Total number of p2p pubsub broadcast messages sent",
		})
	p2pMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "wormhole_p2p_broadcast_messages_received_total",
			Help: "Total number of p2p pubsub broadcast messages received",
		}, []string{"topic"})
)

type Gossip struct {
	priv           crypto.PrivKey
	gst            *node_common.GuardianSetState
	networkID      string
	bootstrapPeers string
	nodeName       string

	p2pHost host.Host
	ps      *pubsub.PubSub

	topics    map[string]*pubsub.Topic
	topicLock sync.Mutex

	logger *zap.Logger
}

func Runnable(
	priv crypto.PrivKey,
	gst *node_common.GuardianSetState,
	networkID string,
	bootstrapPeers string,
	nodeName string,
	components *Components,
	postBoot func(context.Context, *Gossip) supervisor.Runnable,
) supervisor.Runnable {
	return func(ctx context.Context) error {
		g := &Gossip{
			priv:           priv,
			gst:            gst,
			networkID:      networkID,
			bootstrapPeers: bootstrapPeers,
			nodeName:       nodeName,
			topics:         map[string]*pubsub.Topic{},
		}
		g.logger = supervisor.Logger(ctx)

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
		defer h.Close()
		g.p2pHost = h

		ps, err := pubsub.NewGossipSub(ctx, h)
		if err != nil {
			panic(err)
		}
		g.ps = ps

		// Make sure we connect to at least 1 bootstrap node (this is particularly important in a local devnet and CI
		// as peer discovery can take a long time).

		// Count number of successful connection attempts. If we fail to connect to any bootstrap peer, kill
		// the service and have supervisor retry it.
		successes := 0
		// Are we a bootstrap node? If so, it's okay to not have any peers.
		bootstrapNode := false

		for _, addr := range strings.Split(g.bootstrapPeers, ",") {
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
				bootstrapNode = true
				continue
			}

			if err = h.Connect(ctx, *pi); err != nil {
				logger.Error("Failed to connect to bootstrap peer", zap.String("peer", addr), zap.Error(err))
			} else {
				successes += 1
			}
		}

		if successes == 0 && !bootstrapNode {
			return fmt.Errorf("failed to connect to any bootstrap peer")
		} else {
			logger.Info("Connected to bootstrap peers", zap.Int("num", successes))
		}

		logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
			zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))

		err = supervisor.Run(ctx, "children", postBoot(ctx, g))
		if err != nil {
			return err
		}
		supervisor.Signal(ctx, supervisor.SignalHealthy)

		<-ctx.Done()

		return nil
	}
}

func (g *Gossip) Topic(handle string) (GossipIO, error) {
	g.topicLock.Lock()
	defer g.topicLock.Unlock()
	topic := fmt.Sprintf("%s/%s", g.networkID, handle)

	g.logger.Info("Subscribing pubsub topic", zap.String("topic", topic))

	if th, exists := g.topics[topic]; exists {
		return &GossipTopicHandle{th: th, gossip: g, logger: g.logger.With(zap.String("topic", handle))}, nil
	}
	th, err := g.ps.Join(topic)
	if err != nil {
		return nil, fmt.Errorf("failed to join topic: %w", err)
	}

	g.topics[topic] = th
	return &GossipTopicHandle{th: th, gossip: g, logger: g.logger.With(zap.String("topic", topic))}, nil
}

func (g *Gossip) PeerID() peer.ID {
	return g.p2pHost.ID()
}

type GossipTopicHandle struct {
	th     *pubsub.Topic
	gossip *Gossip
	logger *zap.Logger
}

func (g *GossipTopicHandle) Subscribe(ctx context.Context, ch chan<- *GossipEnvelope) error {
	sub, err := g.th.Subscribe()
	if err != nil {
		return fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	go func() {
		for {
			envelope, err := sub.Next(ctx)
			if err != nil {
				sub.Cancel()
				return
			}
			p2pMessagesReceived.WithLabelValues(g.th.String()).Inc()

			var msg gossipv1.GossipMessage
			err = proto.Unmarshal(envelope.Data, &msg)
			if err != nil {
				g.logger.Info("received invalid message",
					zap.Binary("data", envelope.Data),
					zap.String("from", envelope.GetFrom().String()))
				p2pMessagesReceived.WithLabelValues("invalid").Inc()
				continue
			}

			if envelope.GetFrom() == g.gossip.p2pHost.ID() {
				g.logger.Debug("received message from ourselves, ignoring",
					zap.Any("payload", msg.Message))
				p2pMessagesReceived.WithLabelValues("loopback").Inc()
				continue
			}

			g.logger.Debug("received message",
				zap.Any("payload", msg.Message),
				zap.Binary("raw", envelope.Data),
				zap.String("from", envelope.GetFrom().String()))

			ch <- &GossipEnvelope{
				Message: &msg,
				From:    envelope.GetFrom(),
			}
		}
	}()

	return nil
}

func (g *GossipTopicHandle) Send(ctx context.Context, msg *gossipv1.GossipMessage) error {
	b, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	p2pMessagesSent.Inc()

	return g.th.Publish(ctx, b)
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
