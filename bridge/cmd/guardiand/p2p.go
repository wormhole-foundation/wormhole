package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	"github.com/certusone/wormhole/bridge/pkg/devnet"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
)

func p2p(obsvC chan *gossipv1.LockupObservation, sendC chan []byte) func(ctx context.Context) error {
	return func(ctx context.Context) (re error) {
		logger := supervisor.Logger(ctx)

		var priv crypto.PrivKey
		var err error

		if *unsafeDevMode {
			idx, err2 := devnet.GetDevnetIndex()
			if err2 != nil {
				logger.Fatal("Failed to parse hostname - are we running in devnet?")
			}
			priv = devnet.DeterministicP2PPrivKeyByIndex(int64(idx))
		} else {
			priv, err = getOrCreateNodeKey(logger, *nodeKeyPath)
			if err != nil {
				return fmt.Errorf("failed to load node key: %w", err)
			}
		}

		var idht *dht.IpfsDHT

		h, err := libp2p.New(ctx,
			// Use the keypair we generated
			libp2p.Identity(priv),

			// Multiple listen addresses
			libp2p.ListenAddrStrings(
				// Listen on QUIC only.
				// TODO(leo): is this more or less stable than using both TCP and QUIC transports?
				// https://github.com/libp2p/go-libp2p/issues/688
				fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", *p2pPort),
				fmt.Sprintf("/ip6/::/udp/%d/quic", *p2pPort),
			),

			// Enable TLS security as the only security protocol.
			libp2p.Security(libp2ptls.ID, libp2ptls.New),

			// Enable QUIC transport as the only transport.
			libp2p.Transport(libp2pquic.NewTransport),

			// Let's prevent our peer from having too many
			// connections by attaching a connection manager.
			libp2p.ConnectionManager(connmgr.NewConnManager(
				100,         // Lowwater
				400,         // HighWater,
				time.Minute, // GracePeriod
			)),

			// Let this host use the DHT to find other hosts
			libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
				// TODO(leo): Persistent data store (i.e. address book)
				idht, err = dht.New(ctx, h, dht.Mode(dht.ModeServer),
					// TODO(leo): This intentionally makes us incompatible with the global IPFS DHT
					dht.ProtocolPrefix(protocol.ID("/"+*p2pNetworkID)),
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

		logger.Info("Connecting to bootstrap peers", zap.String("bootstrap_peers", *p2pBootstrap))

		// Add our own bootstrap nodes

		// Count number of successful connection attempts. If we fail to connect to every bootstrap peer, kill
		// the service and have supervisor retry it.
		successes := 0
		// Are we a bootstrap node? If so, it's okay to not have any peers.
		bootstrap_node := false

		for _, addr := range strings.Split(*p2pBootstrap, ",") {
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
				bootstrap_node = true
				continue
			}

			if err = h.Connect(ctx, *pi); err != nil {
				logger.Error("Failed to connect to bootstrap peer", zap.String("peer", addr), zap.Error(err))
			} else {
				successes += 1
			}
		}

		// TODO: continually reconnect to bootstrap nodes?
		if successes == 0 && !bootstrap_node {
			return fmt.Errorf("Failed to connect to any bootstrap peer")
		} else {
			logger.Info("Connected to bootstrap peers", zap.Int("num", successes))
		}

		topic := fmt.Sprintf("%s/%s", *p2pNetworkID, "broadcast")

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

		go func() {
			ctr := int64(0)
			tick := time.NewTicker(15 * time.Second)
			defer tick.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case <-tick.C:
					msg := gossipv1.GossipMessage{Message: &gossipv1.GossipMessage_Heartbeat{
						Heartbeat: &gossipv1.Heartbeat{
							NodeName:  *nodeName,
							Counter:   ctr,
							Timestamp: time.Now().UnixNano(),
						}}}

					b, err := proto.Marshal(&msg)
					if err != nil {
						panic(err)
					}

					err = th.Publish(ctx, b)
					if err != nil {
						logger.Warn("failed to publish heartbeat message", zap.Error(err))
					}
				}
			}
		}()

		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case msg := <-sendC:
					err := th.Publish(ctx, msg)
					if err != nil {
						logger.Error("failed to publish message from queue", zap.Error(err))
					}
				}
			}
		}()

		supervisor.Signal(ctx, supervisor.SignalHealthy)

		for {
			envl, err := sub.Next(ctx)
			if err != nil {
				return fmt.Errorf("failed to receive pubsub message: %w", err)
			}

			var msg gossipv1.GossipMessage
			err = proto.Unmarshal(envl.Data, &msg)
			if err != nil {
				logger.Info("received invalid message",
					zap.String("data", string(envl.Data)),
					zap.String("from", envl.GetFrom().String()))
				continue
			}

			// TODO: better way to handle our own sigs?
			//if envl.GetFrom() == h.ID() {
			//	logger.Debug("received message from ourselves, ignoring",
			//		zap.Any("payload", msg.Message))
			//	continue
			//}

			logger.Debug("received message",
				zap.Any("payload", msg.Message),
				zap.Binary("raw", envl.Data),
				zap.String("from", envl.GetFrom().String()))

			switch m := msg.Message.(type) {
			case *gossipv1.GossipMessage_Heartbeat:
				logger.Info("heartbeat received",
					zap.Any("value", m.Heartbeat),
					zap.String("from", envl.GetFrom().String()))
			case *gossipv1.GossipMessage_LockupObservation:
				obsvC <- m.LockupObservation
			default:
				logger.Warn("received unknown message type (running outdated software?)",
					zap.Any("payload", msg.Message),
					zap.Binary("raw", envl.Data),
					zap.String("from", envl.GetFrom().String()))
			}
		}
	}
}
