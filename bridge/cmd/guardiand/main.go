package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/bridge/pkg/supervisor"

	ipfslog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pquic "github.com/libp2p/go-libp2p-quic-transport"
	libp2ptls "github.com/libp2p/go-libp2p-tls"
)

var (
	p2pNetworkID = flag.String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort      = flag.Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = flag.String("bootstrap", "", "P2P bootstrap peers (comma-separated)")
	logLevel     = flag.String("loglevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")
)

func main() {
	flag.Parse()

	// Set up logging. The go-log zap wrapper that libp2p uses is compatible with our
	// usage of zap in supervisor, which is nice.
	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	// Our root logger.
	logger := ipfslog.Logger("wormhole")

	// Override the default go-log config, which uses a magic environment variable.
	ipfslog.SetAllLoggers(lvl)

	// Mute chatty subsystems.
	ipfslog.SetLogLevel("swarm2", "error") // connection errors

	// Node's main lifecycle context.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Run supervisor.
	supervisor.New(ctx, logger.Desugar(), func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "p2p", p2p); err != nil {
			return err
		}

		supervisor.Signal(ctx, supervisor.SignalHealthy)
		logger.Info("Created services")

		select {}
	}, supervisor.WithPropagatePanic) // TODO(leo): only propagate panics in debug mode

	select {}
}

func p2p(ctx context.Context) error {
	logger := supervisor.Logger(ctx)

	// TODO(leo): persist the key
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)

	if err != nil {
		panic(err)
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
	defer h.Close()

	if err != nil {
		panic(err)
	}

	logger.Info("Connecting to bootstrap peers")
	// TODO(leo): use our own bootstrap peers rather than the IPFS ones so we have a dedicated network
	//for _, addr := range dht.DefaultBootstrapPeers {
	//	pi, _ := peer.AddrInfoFromP2pAddr(addr)
	//	// We ignore errors as some bootstrap peers may be down and that is fine.
	//	_ = h.Connect(ctx, *pi)
	//}

	// Add our own bootstrap nodes
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
			logger.Error("Failed to connect to bootstrap peer", zap.String("peer", addr), zap.Error(err))
		}
	}

	topic := fmt.Sprintf("%s/%s", *p2pNetworkID, "broadcast")

	logger.Info("Subscribing pubsub topic", zap.String("topic", topic))
	ps, err := pubsub.NewGossipSub(ctx, h)
	if err != nil {
		panic(err)
	}

	logger.Info("Node has been started", zap.String("peer_id", h.ID().String()),
		zap.String("addrs", fmt.Sprintf("%v", h.Addrs())))

	for {
		time.Sleep(1 * time.Second)
		for _, p := range ps.ListPeers(topic) {
			logger.Debug("Found pubsub peer", zap.String("peer_id", p.Pretty()))
		}
	}

	select {}
}
