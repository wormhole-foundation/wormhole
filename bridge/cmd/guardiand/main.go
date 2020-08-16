package main

import (
	"context"
	"encoding/base64"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p-core/protocol"
	swarm "github.com/libp2p/go-libp2p-swarm"
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
	nodeKeyPath  = flag.String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")
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

	// FIXME: add hostname to root logger for cleaner console output in multi-node development.
	// The proper way is to change the output format to include the hostname.
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	// Our root logger.
	logger := ipfslog.Logger(fmt.Sprintf("%s-%s", "wormhole", hostname))

	// Override the default go-log config, which uses a magic environment variable.
	ipfslog.SetAllLoggers(lvl)

	// Mute chatty subsystems.
	ipfslog.SetLogLevel("swarm2", "error") // connection errors

	// Verify flags
	if *nodeKeyPath == "" {
		logger.Fatal("Please specify -nodeKey")
	}

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
	}, supervisor.WithPropagatePanic)
	// TODO(leo): only propagate panics in debug mode. We currently need this to properly reset p2p
	// (it leaks its socket and we need to restart the process to fix it)

	select {}
}

func getOrCreateNodeKey(logger *zap.Logger, path string) (crypto.PrivKey, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Info("No node key found, generating a new one...", zap.String("path", path))

			// TODO(leo): what does -1 mean?
			priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519, -1)
			if err != nil {
				panic(err)
			}

			s, err := crypto.MarshalPrivateKey(priv)
			if err != nil {
				panic(err)
			}

			err = ioutil.WriteFile(path, s, 0600)
			if err != nil {
				return nil, fmt.Errorf("failed to write node key: %w", err)
			}

			return priv, nil
		} else {
			return nil, fmt.Errorf("failed to read node key: %w", err)
		}
	}

	priv, err := crypto.UnmarshalPrivateKey(b)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal node key: %w", err)
	}

	logger.Info("Found existing node key", zap.String("path", path))

	return priv, nil
}

// FIXME: this hardcodes the private key if we're guardian-0.
// Proper fix is to add a debug mode and fetch the remote peer ID,
// or add a special bootstrap pod.
func bootstrapNodePrivateKeyHack() crypto.PrivKey {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	if hostname == "guardian-0" {
		// node ID: 12D3KooWQ1sV2kowPY1iJX1hJcVTysZjKv3sfULTGwhdpUGGZ1VF
		b, err := base64.StdEncoding.DecodeString("CAESQGlv6OJOMXrZZVTCC0cgCv7goXr6QaSVMZIndOIXKNh80vYnG+EutVlZK20Nx9cLkUG5ymKB\n88LXi/vPBwP8zfY=")
		if err != nil {
			panic(err)
		}

		priv, err := crypto.UnmarshalPrivateKey(b)
		if err != nil {
			panic(err)
		}

		return priv
	}

	return nil
}

func p2p(ctx context.Context) error {
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
	defer func() {
		fmt.Printf("h is %+v", h)
		// FIXME: why can this be nil? We need to close the host to free the socket because apparently,
		// closing the context is not enough, but sometimes h is nil when the function runs.
		if h != nil {
			h.Close()
		}
	}()

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
		return fmt.Errorf("Failed to connect to any bootstrap peer")
	} else {
		logger.Info("Connected to bootstrap peers", zap.Int("num", successes))
	}

	// TODO(leo): crash if we couldn't connect to any bootstrap peers?
	// (i.e. can we get stuck here if the other nodes have yet to come up?)

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
