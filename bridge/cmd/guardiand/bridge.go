package guardiand

import (
	"context"
	"fmt"
	"github.com/gagliardetto/solana-go/rpc"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"syscall"

	solana_types "github.com/gagliardetto/solana-go"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	eth_common "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"github.com/certusone/wormhole/bridge/pkg/common"
	"github.com/certusone/wormhole/bridge/pkg/devnet"
	"github.com/certusone/wormhole/bridge/pkg/ethereum"
	"github.com/certusone/wormhole/bridge/pkg/p2p"
	"github.com/certusone/wormhole/bridge/pkg/processor"
	gossipv1 "github.com/certusone/wormhole/bridge/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/bridge/pkg/publicrpc"
	"github.com/certusone/wormhole/bridge/pkg/readiness"
	solana "github.com/certusone/wormhole/bridge/pkg/solana"
	"github.com/certusone/wormhole/bridge/pkg/supervisor"
	"github.com/certusone/wormhole/bridge/pkg/vaa"

	"github.com/certusone/wormhole/bridge/pkg/terra"

	ipfslog "github.com/ipfs/go-log/v2"
)

var (
	p2pNetworkID *string
	p2pPort      *uint
	p2pBootstrap *string

	nodeKeyPath *string

	adminSocketPath *string

	dataDir *string

	statusAddr *string

	bridgeKeyPath       *string
	solanaBridgeAddress *string

	ethRPC      *string
	ethContract *string

	bscRPC      *string
	bscContract *string

	terraWS       *string
	terraLCD      *string
	terraChainID  *string
	terraContract *string

	solanaWsRPC *string
	solanaRPC   *string

	logLevel *string

	unsafeDevMode   *bool
	devNumGuardians *uint
	nodeName        *string

	publicRPC *string
)

func init() {
	p2pNetworkID = BridgeCmd.Flags().String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort = BridgeCmd.Flags().Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = BridgeCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")

	statusAddr = BridgeCmd.Flags().String("statusAddr", "Listen address for status server (disabled if blank)", "[::1]:6060")

	nodeKeyPath = BridgeCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	adminSocketPath = BridgeCmd.Flags().String("adminSocket", "", "Admin gRPC service UNIX domain socket path")

	dataDir = BridgeCmd.Flags().String("dataDir", "", "Data directory")

	bridgeKeyPath = BridgeCmd.Flags().String("bridgeKey", "", "Path to guardian key (required)")
	solanaBridgeAddress = BridgeCmd.Flags().String("solanaBridgeAddress", "", "Address of the Solana Bridge Program (required)")

	ethRPC = BridgeCmd.Flags().String("ethRPC", "", "Ethereum RPC URL")
	ethContract = BridgeCmd.Flags().String("ethContract", "", "Ethereum bridge contract address")

	bscRPC = BridgeCmd.Flags().String("bscRPC", "", "Binance Smart Chain RPC URL")
	bscContract = BridgeCmd.Flags().String("bscContract", "", "Binance Smart Chain bridge contract address")

	terraWS = BridgeCmd.Flags().String("terraWS", "", "Path to terrad root for websocket connection")
	terraLCD = BridgeCmd.Flags().String("terraLCD", "", "Path to LCD service root for http calls")
	terraChainID = BridgeCmd.Flags().String("terraChainID", "", "Terra chain ID, used in LCD client initialization")
	terraContract = BridgeCmd.Flags().String("terraContract", "", "Wormhole contract address on Terra blockchain")

	solanaWsRPC = BridgeCmd.Flags().String("solanaWS", "", "Solana Websocket URL (required")
	solanaRPC = BridgeCmd.Flags().String("solanaRPC", "", "Solana RPC URL (required")

	logLevel = BridgeCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")

	unsafeDevMode = BridgeCmd.Flags().Bool("unsafeDevMode", false, "Launch node in unsafe, deterministic devnet mode")
	devNumGuardians = BridgeCmd.Flags().Uint("devNumGuardians", 5, "Number of devnet guardians to include in guardian set")
	nodeName = BridgeCmd.Flags().String("nodeName", "", "Node name to announce in gossip heartbeats")

	publicRPC = BridgeCmd.Flags().String("publicRPC", "", "Listen address for public gRPC interface")
}

var (
	rootCtx       context.Context
	rootCtxCancel context.CancelFunc
)

// "Why would anyone do this?" are famous last words.
//
// We already forcibly override RPC URLs and keys in dev mode to prevent security
// risks from operator error, but an extra warning won't hurt.
const devwarning = `
        +++++++++++++++++++++++++++++++++++++++++++++++++++
        |   NODE IS RUNNING IN INSECURE DEVELOPMENT MODE  |
        |                                                 |
        |      Do not use -unsafeDevMode in prod.         |
        +++++++++++++++++++++++++++++++++++++++++++++++++++

`

func rootLoggerName() string {
	if *unsafeDevMode {
		// FIXME: add hostname to root logger for cleaner console output in multi-node development.
		// The proper way is to change the output format to include the hostname.
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}

		return fmt.Sprintf("%s-%s", "wormhole", hostname)
	} else {
		return "wormhole"
	}
}

// lockMemory locks current and future pages in memory to protect secret keys from being swapped out to disk.
// It's possible (and strongly recommended) to deploy Wormhole such that keys are only ever
// stored in memory and never touch the disk. This is a privileged operation and requires CAP_IPC_LOCK.
func lockMemory() {
	err := unix.Mlockall(syscall.MCL_CURRENT | syscall.MCL_FUTURE)
	if err != nil {
		fmt.Printf("Failed to lock memory: %v (CAP_IPC_LOCK missing?)\n", err)
		os.Exit(1)
	}
}

// setRestrictiveUmask masks the group and world bits. This ensures that key material
// and sockets we create aren't accidentally group- or world-readable.
func setRestrictiveUmask() {
	syscall.Umask(0077) // cannot fail
}

// BridgeCmd represents the bridge command
var BridgeCmd = &cobra.Command{
	Use:   "bridge",
	Short: "Run the bridge server",
	Run:   runBridge,
}

func runBridge(cmd *cobra.Command, args []string) {
	if *unsafeDevMode {
		fmt.Print(devwarning)
	}

	lockMemory()
	setRestrictiveUmask()

	// Set up logging. The go-log zap wrapper that libp2p uses is compatible with our
	// usage of zap in supervisor, which is nice.
	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	// Our root logger. Convert directly to a regular Zap logger.
	logger := ipfslog.Logger(rootLoggerName()).Desugar()

	// Override the default go-log config, which uses a magic environment variable.
	ipfslog.SetAllLoggers(lvl)

	// Register components for readiness checks.
	readiness.RegisterComponent(common.ReadinessEthSyncing)
	readiness.RegisterComponent(common.ReadinessSolanaSyncing)
	readiness.RegisterComponent(common.ReadinessTerraSyncing)

	if *statusAddr != "" {
		// Use a custom routing instead of using http.DefaultServeMux directly to avoid accidentally exposing packages
		// that register themselves with it by default (like pprof).
		router := mux.NewRouter()

		// pprof server. NOT necessarily safe to expose publicly - only enable it in dev mode to avoid exposing it by
		// accident. There's benefit to having pprof enabled on production nodes, but we would likely want to expose it
		// via a dedicated port listening on localhost, or via the admin UNIX socket.
		if *unsafeDevMode {
			// Pass requests to http.DefaultServeMux, which pprof automatically registers with as an import side-effect.
			router.PathPrefix("/debug/pprof/").Handler(http.DefaultServeMux)
		}

		// Simple endpoint exposing node readiness (safe to expose to untrusted clients)
		router.HandleFunc("/readyz", readiness.Handler)

		// Prometheus metrics (safe to expose to untrusted clients)
		router.Handle("/metrics", promhttp.Handler())

		go func() {
			logger.Info("status server listening on [::]:6060")
			logger.Error("status server crashed", zap.Error(http.ListenAndServe("[::]:6060", router)))
		}()
	}

	// In devnet mode, we automatically set a number of flags that rely on deterministic keys.
	if *unsafeDevMode {
		g0key, err := peer.IDFromPrivateKey(devnet.DeterministicP2PPrivKeyByIndex(0))
		if err != nil {
			panic(err)
		}

		// Use the first guardian node as bootstrap
		*p2pBootstrap = fmt.Sprintf("/dns4/guardian-0.guardian/udp/%d/quic/p2p/%s", *p2pPort, g0key.String())

		// Deterministic ganache ETH devnet address.
		*ethContract = devnet.GanacheBridgeContractAddress.Hex()
		*bscContract = devnet.GanacheBridgeContractAddress.Hex()

		// Use the hostname as nodeName. For production, we don't want to do this to
		// prevent accidentally leaking sensitive hostnames.
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		*nodeName = hostname
	}

	// Verify flags

	if *nodeKeyPath == "" && !*unsafeDevMode { // In devnet mode, keys are deterministically generated.
		logger.Fatal("Please specify --nodeKey")
	}
	if *bridgeKeyPath == "" {
		logger.Fatal("Please specify --bridgeKey")
	}
	if *adminSocketPath == "" {
		logger.Fatal("Please specify --adminSocket")
	}
	if *dataDir == "" {
		logger.Fatal("Please specify --dataDir")
	}
	if *ethRPC == "" {
		logger.Fatal("Please specify --ethRPC")
	}
	if *ethContract == "" {
		logger.Fatal("Please specify --ethContract")
	}
	if *bscRPC == "" {
		logger.Fatal("Please specify --bscRPC")
	}
	if *bscContract == "" {
		logger.Fatal("Please specify --bscContract")
	}
	if *nodeName == "" {
		logger.Fatal("Please specify --nodeName")
	}

	if *solanaBridgeAddress == "" {
		logger.Fatal("Please specify --solanaBridgeAddress")
	}
	if *solanaWsRPC == "" {
		logger.Fatal("Please specify --solanaWsUrl")
	}
	if *solanaRPC == "" {
		logger.Fatal("Please specify --solanaUrl")
	}

	if *terraWS == "" {
		logger.Fatal("Please specify --terraWS")
	}
	if *terraLCD == "" {
		logger.Fatal("Please specify --terraLCD")
	}
	if *terraChainID == "" {
		logger.Fatal("Please specify --terraChainID")
	}
	if *terraContract == "" {
		logger.Fatal("Please specify --terraContract")
	}

	ethContractAddr := eth_common.HexToAddress(*ethContract)
	bscContractAddr := eth_common.HexToAddress(*bscContract)
	solBridgeAddress, err := solana_types.PublicKeyFromBase58(*solanaBridgeAddress)
	if err != nil {
		logger.Fatal("invalid Solana bridge address", zap.Error(err))
	}

	// In devnet mode, we generate a deterministic guardian key and write it to disk.
	if *unsafeDevMode {
		gk, err := generateDevnetGuardianKey()
		if err != nil {
			logger.Fatal("failed to generate devnet guardian key", zap.Error(err))
		}

		err = writeGuardianKey(gk, "auto-generated deterministic devnet key", *bridgeKeyPath, true)
		if err != nil {
			logger.Fatal("failed to write devnet guardian key", zap.Error(err))
		}
	}

	// Guardian key
	gk, err := loadGuardianKey(*bridgeKeyPath)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}

	guardianAddr := ethcrypto.PubkeyToAddress(gk.PublicKey).String()
	logger.Info("Loaded guardian key", zap.String(
		"address", guardianAddr))

	p2p.DefaultRegistry.SetGuardianAddress(guardianAddr)

	// Node's main lifecycle context.
	rootCtx, rootCtxCancel = context.WithCancel(context.Background())
	defer rootCtxCancel()

	// Ethereum lock event channel
	lockC := make(chan *common.MessagePublication)

	// Ethereum incoming guardian set updates
	setC := make(chan *common.GuardianSet)

	// Outbound gossip message queue
	sendC := make(chan []byte)

	// Inbound observations
	obsvC := make(chan *gossipv1.SignedObservation, 50)

	// Injected VAAs (manually generated rather than created via observation)
	injectC := make(chan *vaa.VAA)

	// Load p2p private key
	var priv crypto.PrivKey
	if *unsafeDevMode {
		idx, err := devnet.GetDevnetIndex()
		if err != nil {
			logger.Fatal("Failed to parse hostname - are we running in devnet?")
		}
		priv = devnet.DeterministicP2PPrivKeyByIndex(int64(idx))
	} else {
		priv, err = getOrCreateNodeKey(logger, *nodeKeyPath)
		if err != nil {
			logger.Fatal("Failed to load node key", zap.Error(err))
		}
	}

	// subscriber channel multiplexing for public gPRC streams
	rawHeartbeatListeners := publicrpc.HeartbeatStreamMultiplexer(logger)
	publicrpcService, err := publicrpcServiceRunnable(logger, *publicRPC, rawHeartbeatListeners)
	if err != nil {
		log.Fatal("failed to create publicrpc service socket", zap.Error(err))
	}

	// local admin service socket
	adminService, err := adminServiceRunnable(logger, *adminSocketPath, injectC, rawHeartbeatListeners)
	if err != nil {
		logger.Fatal("failed to create admin service socket", zap.Error(err))
	}

	// Run supervisor.
	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "p2p", p2p.Run(
			obsvC, sendC, rawHeartbeatListeners, priv, *p2pPort, *p2pNetworkID, *p2pBootstrap, *nodeName, rootCtxCancel)); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "ethwatch",
			ethereum.NewEthBridgeWatcher(*ethRPC, ethContractAddr, "eth", vaa.ChainIDEthereum, lockC, setC).Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "bscwatch",
			ethereum.NewEthBridgeWatcher(*bscRPC, bscContractAddr, "bsc", vaa.ChainIDBSC, lockC, nil).Run); err != nil {
			return err
		}

		// Start Terra watcher only if configured
		logger.Info("Starting Terra watcher")
		if err := supervisor.Run(ctx, "terrawatch",
			terra.NewTerraBridgeWatcher(*terraWS, *terraLCD, *terraContract, lockC, setC).Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "solwatch-confirmed",
			solana.NewSolanaWatcher(*solanaWsRPC, *solanaRPC, solBridgeAddress, lockC, rpc.CommitmentConfirmed).Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "solwatch-finalized",
			solana.NewSolanaWatcher(*solanaWsRPC, *solanaRPC, solBridgeAddress, lockC, rpc.CommitmentFinalized).Run); err != nil {
			return err
		}

		// TODO: this thing has way too many arguments at this point - make it an options struct
		p := processor.NewProcessor(ctx,
			lockC,
			setC,
			sendC,
			obsvC,
			injectC,
			gk,
			*unsafeDevMode,
			*devNumGuardians,
			*ethRPC,
			*terraLCD,
			*terraChainID,
			*terraContract,
		)
		if err := supervisor.Run(ctx, "processor", p.Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "admin", adminService); err != nil {
			return err
		}
		if *publicRPC != "" {
			if err := supervisor.Run(ctx, "publicrpc", publicrpcService); err != nil {
				return err
			}
		}

		logger.Info("Started internal services")

		select {
		case <-ctx.Done():
			return nil
		}
	},
		// It's safer to crash and restart the process in case we encounter a panic,
		// rather than attempting to reschedule the runnable.
		supervisor.WithPropagatePanic)

	select {
	case <-rootCtx.Done():
		logger.Info("root context cancelled, exiting...")
		// TODO: wait for things to shut down gracefully
	}
}
