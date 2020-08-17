package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	swarm "github.com/libp2p/go-libp2p-swarm"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
)

func p2p(ctx context.Context) (re error) {
	logger := supervisor.Logger(ctx)

	priv := bootstrapNodePrivateKeyHack()

	var err error
	if priv == nil {
		priv, err = getOrCreateNodeKey(logger, *nodeKeyPath)
		if err != nil {
			panic(err)
		}
	} else {
		logger.Info("HACK: loaded hardcoded guardian-0 node key")
	}

	var idht *dht.IpfsDHT

	h, err := libp2p.New(ctx,
		// Use the keypair we generated
		libp2p.Identity(priv),
		// Multiple listen addresses
		libp2p.ListenAddrStrings(
			// Listen on QUIC only.
			// TODO(leo): listen on ipv6
			// TODO(leo): is this more or less stable than using both TCP and QUIC transports?
			// https://github.com/libp2p/go-libp2p/issues/688
			fmt.Sprintf("/ip4/0.0.0.0/udp/%d/quic", *p2pPort),
		),

		// Enable TLS security only.
		libp2p.Security(libp2ptls.ID, libp2ptls.New),

		// Enable QUIC transports.
		libp2p.Transport(libp2pquic.NewTransport),

		// Enable TCP so we can connect to bootstrap nodes.
		// (can be disabled if we bootstrap our own network)
		libp2p.DefaultTransports,

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

	logger.Info("Connecting to bootstrap peers")

	// Add our own bootstrap nodes

	// Count number of successful connection attempts. If we fail to connect to every bootstrap peer, kill
	// the service and have supervisor retry it.
	successes := 0

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

		if err = h.Connect(ctx, *pi); err != nil {
			if err != swarm.ErrDialToSelf {
				logger.Error("Failed to connect to bootstrap peer", zap.String("peer", addr), zap.Error(err))
			} else {
				// Dialing self, carrying on... (we're a bootstrap peer)
				logger.Info("Tried to connect to ourselves - we're a bootstrap peer")
				successes += 1
			}
		} else {
			successes += 1
		}
	}

	if successes == 0 {
		h.Close()
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

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	go func() {
		ctr := int64(0)

		for {
			msg := gossipv1.Heartbeat{
				Hostname: hostname,
				Index:    ctr,
			}

			b, err := proto.Marshal(&msg)
			if err != nil {
				panic(err)
			}

			err = th.Publish(ctx, b)
			if err != nil {
				logger.Warn("failed to publish message", zap.Error(err))
			}

			time.Sleep(15 * time.Second)
		}
	}()

	supervisor.Signal(ctx, supervisor.SignalHealthy)

	for {
		msg, err := sub.Next(ctx)
		if err != nil {
			return fmt.Errorf("failed to receive pubsub message: %w", err)
		}

		logger.Info("received message", zap.String("data", string(msg.Data)), zap.String("from", msg.GetFrom().String()))
	}
}
