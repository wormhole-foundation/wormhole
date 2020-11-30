package guardiand

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"syscall"

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

	bridgeKeyPath *string

	ethRPC           *string
	ethContract      *string
	ethConfirmations *uint64

	terraSupport  *bool
	terraWS       *string
	terraLCD      *string
	terraChaidID  *string
	terraContract *string
	terraKeyPath  *string

	agentRPC *string

	logLevel *string

	unsafeDevMode   *bool
	devNumGuardians *uint
	nodeName        *string
)

func init() {
	p2pNetworkID = BridgeCmd.Flags().String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort = BridgeCmd.Flags().Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = BridgeCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")

	nodeKeyPath = BridgeCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	adminSocketPath = BridgeCmd.Flags().String("adminSocket", "", "Admin gRPC service UNIX domain socket path")

	bridgeKeyPath = BridgeCmd.Flags().String("bridgeKey", "", "Path to guardian key (required)")

	ethRPC = BridgeCmd.Flags().String("ethRPC", "", "Ethereum RPC URL")
	ethContract = BridgeCmd.Flags().String("ethContract", "", "Ethereum bridge contract address")
	ethConfirmations = BridgeCmd.Flags().Uint64("ethConfirmations", 15, "Ethereum confirmation count requirement")

	terraSupport = BridgeCmd.Flags().Bool("terra", false, "Turn on support for Terra")
	terraWS = BridgeCmd.Flags().String("terraWS", "", "Path to terrad root for websocket connection")
	terraLCD = BridgeCmd.Flags().String("terraLCD", "", "Path to LCD service root for http calls")
	terraChaidID = BridgeCmd.Flags().String("terraChainID", "", "Terra chain ID, used in LCD client initialization")
	terraContract = BridgeCmd.Flags().String("terraContract", "", "Wormhole contract address on Terra blockhain")
	terraKeyPath = BridgeCmd.Flags().String("terraKey", "", "Path to mnemonic for account paying gas for submitting transactions to Terra")

	agentRPC = BridgeCmd.Flags().String("agentRPC", "", "Solana agent sidecar gRPC socket path")

	logLevel = BridgeCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")

	unsafeDevMode = BridgeCmd.Flags().Bool("unsafeDevMode", false, "Launch node in unsafe, deterministic devnet mode")
	devNumGuardians = BridgeCmd.Flags().Uint("devNumGuardians", 5, "Number of devnet guardians to include in guardian set")
	nodeName = BridgeCmd.Flags().String("nodeName", "", "Node name to announce in gossip heartbeats")
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
	if *terraSupport {
		readiness.RegisterComponent(common.ReadinessTerraSyncing)
	}

	// In devnet mode, we automatically set a number of flags that rely on deterministic keys.
	if *unsafeDevMode {
		go func() {
			// TODO: once monitoring server is implemented, move this to that http server instance
			http.HandleFunc("/readyz", readiness.Handler)

			logger.Info("debug server listening on [::]:6060")
			logger.Error("debug server crashed", zap.Error(http.ListenAndServe("[::]:6060", nil)))
		}()

		g0key, err := peer.IDFromPrivateKey(devnet.DeterministicP2PPrivKeyByIndex(0))
		if err != nil {
			panic(err)
		}

		// Use the first guardian node as bootstrap
		*p2pBootstrap = fmt.Sprintf("/dns4/guardian-0.guardian/udp/%d/quic/p2p/%s", *p2pPort, g0key.String())

		// Deterministic ganache ETH devnet address.
		*ethContract = devnet.GanacheBridgeContractAddress.Hex()

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
	if *agentRPC == "" {
		logger.Fatal("Please specify --agentRPC")
	}
	if *ethRPC == "" {
		logger.Fatal("Please specify --ethRPC")
	}
	if *ethContract == "" {
		logger.Fatal("Please specify --ethContract")
	}
	if *nodeName == "" {
		logger.Fatal("Please specify --nodeName")
	}
	if *terraSupport {
		if !*unsafeDevMode {
			logger.Fatal("cannot enable terra support in production mode")
		}

		if *terraWS == "" {
			logger.Fatal("Please specify --terraWS")
		}
		if *terraLCD == "" {
			logger.Fatal("Please specify --terraLCD")
		}
		if *terraChaidID == "" {
			logger.Fatal("Please specify --terraChaidID")
		}
		if *terraContract == "" {
			logger.Fatal("Please specify --terraContract")
		}
		if *terraKeyPath == "" {
			logger.Fatal("Please specify --terraKey")
		}
	}

	ethContractAddr := eth_common.HexToAddress(*ethContract)

	// In devnet mode, we generate a deterministic guardian key and write it to disk.
	if *unsafeDevMode {
		gk, err := generateDevnetGuardianKey()
		if err != nil {
			logger.Fatal("failed to generate devnet guardian key", zap.Error(err))
		}

		err = writeGuardianKey(gk, "auto-generated deterministic devnet key", *bridgeKeyPath)
		if err != nil {
			logger.Fatal("failed to write devnet guardian key", zap.Error(err))
		}
	}

	// Guardian key
	gk, err := loadGuardianKey(*bridgeKeyPath)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}

	logger.Info("Loaded guardian key", zap.String(
		"address", ethcrypto.PubkeyToAddress(gk.PublicKey).String()))

	// Node's main lifecycle context.
	rootCtx, rootCtxCancel = context.WithCancel(context.Background())
	defer rootCtxCancel()

	// Ethereum lock event channel
	lockC := make(chan *common.ChainLock)

	// Ethereum incoming guardian set updates
	setC := make(chan *common.GuardianSet)

	// Outbound gossip message queue
	sendC := make(chan []byte)

	// Inbound observations
	obsvC := make(chan *gossipv1.SignedObservation, 50)

	// VAAs to submit to Solana
	solanaVaaC := make(chan *vaa.VAA)

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

	// Load Terra fee payer key
	var terraFeePayer string
	if *terraSupport {
		if *unsafeDevMode {
			terra.WriteDevnetKey(*terraKeyPath)
		}
		terraFeePayer, err = terra.ReadKey(*terraKeyPath)
		if err != nil {
			logger.Fatal("Failed to load Terra fee payer key", zap.Error(err))
		}
	}

	adminService, err := adminServiceRunnable(logger, *adminSocketPath, injectC)
	if err != nil {
		logger.Fatal("failed to create admin service socket", zap.Error(err))
	}

	// Run supervisor.
	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "p2p", p2p.Run(
			obsvC, sendC, priv, *p2pPort, *p2pNetworkID, *p2pBootstrap, *nodeName, rootCtxCancel)); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "ethwatch",
			ethereum.NewEthBridgeWatcher(*ethRPC, ethContractAddr, *ethConfirmations, lockC, setC).Run); err != nil {
			return err
		}

		// Start Terra watcher only if configured
		if *terraSupport {
			logger.Info("Starting Terra watcher")
			if err := supervisor.Run(ctx, "terrawatch",
				terra.NewTerraBridgeWatcher(*terraWS, *terraLCD, *terraContract, lockC, setC).Run); err != nil {
				return err
			}
		}

		if err := supervisor.Run(ctx, "solwatch",
			solana.NewSolanaBridgeWatcher(*agentRPC, lockC, solanaVaaC).Run); err != nil {
			return err
		}

		// TODO: this thing has way too many arguments at this point - make it an options struct
		p := processor.NewProcessor(ctx,
			lockC,
			setC,
			sendC,
			obsvC,
			solanaVaaC,
			injectC,
			gk,
			*unsafeDevMode,
			*devNumGuardians,
			*ethRPC,
			*terraSupport,
			*terraLCD,
			*terraChaidID,
			*terraContract,
			terraFeePayer,
		)
		if err := supervisor.Run(ctx, "processor", p.Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "admin", adminService); err != nil {
			return err
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
