package guardiand

import (
	"context"
	"encoding/base64"
	"fmt"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/notify/discord"
	"github.com/certusone/wormhole/node/pkg/telemetry"
	"github.com/certusone/wormhole/node/pkg/version"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap/zapcore"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path"
	"strings"

	solana_types "github.com/gagliardetto/solana-go"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/certusone/wormhole/node/pkg/ethereum"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/processor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/reporter"
	solana "github.com/certusone/wormhole/node/pkg/solana"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	"github.com/certusone/wormhole/node/pkg/vaa"
	eth_common "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/terra"

	"github.com/certusone/wormhole/node/pkg/algorand"

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

	guardianKeyPath *string
	solanaContract  *string

	ethRPC      *string
	ethContract *string

	bscRPC      *string
	bscContract *string

	polygonRPC      *string
	polygonContract *string

	ethRopstenRPC      *string
	ethRopstenContract *string

	fantomRPC	*string
	fantomContract	*string

	avalancheRPC      *string
	avalancheContract *string

	oasisRPC      *string
	oasisContract *string

	terraWS       *string
	terraLCD      *string
	terraContract *string

	algorandRPC      *string
	algorandToken    *string
	algorandContract *string

	solanaWsRPC *string
	solanaRPC   *string

	logLevel *string

	unsafeDevMode   *bool
	testnetMode     *bool
	devNumGuardians *uint
	nodeName        *string

	publicRPC *string
	publicWeb *string

	tlsHostname *string
	tlsProdEnv  *bool

	disableHeartbeatVerify *bool
	disableTelemetry       *bool

	telemetryKey *string

	discordToken   *string
	discordChannel *string

	bigTablePersistenceEnabled *bool
	bigTableGCPProject         *string
	bigTableInstanceName       *string
	bigTableTableName          *string
	bigTableTopicName          *string
	bigTableKeyPath            *string
)

func init() {
	p2pNetworkID = NodeCmd.Flags().String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort = NodeCmd.Flags().Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = NodeCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")

	statusAddr = NodeCmd.Flags().String("statusAddr", "[::]:6060", "Listen address for status server (disabled if blank)")

	nodeKeyPath = NodeCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	adminSocketPath = NodeCmd.Flags().String("adminSocket", "", "Admin gRPC service UNIX domain socket path")

	dataDir = NodeCmd.Flags().String("dataDir", "", "Data directory")

	guardianKeyPath = NodeCmd.Flags().String("guardianKey", "", "Path to guardian key (required)")
	solanaContract = NodeCmd.Flags().String("solanaContract", "", "Address of the Solana program (required)")

	ethRPC = NodeCmd.Flags().String("ethRPC", "", "Ethereum RPC URL")
	ethContract = NodeCmd.Flags().String("ethContract", "", "Ethereum contract address")

	bscRPC = NodeCmd.Flags().String("bscRPC", "", "Binance Smart Chain RPC URL")
	bscContract = NodeCmd.Flags().String("bscContract", "", "Binance Smart Chain contract address")

	polygonRPC = NodeCmd.Flags().String("polygonRPC", "", "Polygon RPC URL")
	polygonContract = NodeCmd.Flags().String("polygonContract", "", "Polygon contract address")

	ethRopstenRPC = NodeCmd.Flags().String("ethRopstenRPC", "", "Ethereum Ropsten RPC URL")
	ethRopstenContract = NodeCmd.Flags().String("ethRopstenContract", "", "Ethereum Ropsten contract address")

	avalancheRPC = NodeCmd.Flags().String("avalancheRPC", "", "Avalanche RPC URL")
	avalancheContract = NodeCmd.Flags().String("avalancheContract", "", "Avalanche contract address")

	oasisRPC = NodeCmd.Flags().String("oasisRPC", "", "Oasis RPC URL")
	oasisContract = NodeCmd.Flags().String("oasisContract", "", "Oasis contract address")

	fantomRPC = NodeCmd.Flags().String("fantomRPC", "", "Fantom Websocket RPC URL")
	fantomContract = NodeCmd.Flags().String("fantomContract", "", "Fantom contract address")

	terraWS = NodeCmd.Flags().String("terraWS", "", "Path to terrad root for websocket connection")
	terraLCD = NodeCmd.Flags().String("terraLCD", "", "Path to LCD service root for http calls")
	terraContract = NodeCmd.Flags().String("terraContract", "", "Wormhole contract address on Terra blockchain")

	algorandRPC = NodeCmd.Flags().String("algorandRPC", "", "Algorand RPC URL")
	algorandToken = NodeCmd.Flags().String("algorandToken", "", "Algorand access token")
	algorandContract = NodeCmd.Flags().String("algorandContract", "", "Algorand contract")

	solanaWsRPC = NodeCmd.Flags().String("solanaWS", "", "Solana Websocket URL (required")
	solanaRPC = NodeCmd.Flags().String("solanaRPC", "", "Solana RPC URL (required")

	logLevel = NodeCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")

	unsafeDevMode = NodeCmd.Flags().Bool("unsafeDevMode", false, "Launch node in unsafe, deterministic devnet mode")
	testnetMode = NodeCmd.Flags().Bool("testnetMode", false, "Launch node in testnet mode (enables testnet-only features like Ropsten)")
	devNumGuardians = NodeCmd.Flags().Uint("devNumGuardians", 5, "Number of devnet guardians to include in guardian set")
	nodeName = NodeCmd.Flags().String("nodeName", "", "Node name to announce in gossip heartbeats")

	publicRPC = NodeCmd.Flags().String("publicRPC", "", "Listen address for public gRPC interface")
	publicWeb = NodeCmd.Flags().String("publicWeb", "", "Listen address for public REST and gRPC Web interface")

	tlsHostname = NodeCmd.Flags().String("tlsHostname", "", "If set, serve publicWeb as TLS with this hostname using Let's Encrypt")
	tlsProdEnv = NodeCmd.Flags().Bool("tlsProdEnv", false,
		"Use the production Let's Encrypt environment instead of staging")

	disableHeartbeatVerify = NodeCmd.Flags().Bool("disableHeartbeatVerify", false,
		"Disable heartbeat signature verification (useful during network startup)")
	disableTelemetry = NodeCmd.Flags().Bool("disableTelemetry", false,
		"Disable telemetry")

	telemetryKey = NodeCmd.Flags().String("telemetryKey", "",
		"Telemetry write key")

	discordToken = NodeCmd.Flags().String("discordToken", "", "Discord bot token (optional)")
	discordChannel = NodeCmd.Flags().String("discordChannel", "", "Discord channel name (optional)")

	bigTablePersistenceEnabled = NodeCmd.Flags().Bool("bigTablePersistenceEnabled", false, "Turn on forwarding events to BigTable")
	bigTableGCPProject = NodeCmd.Flags().String("bigTableGCPProject", "", "Google Cloud project ID for storing events")
	bigTableInstanceName = NodeCmd.Flags().String("bigTableInstanceName", "", "BigTable instance name for storing events")
	bigTableTableName = NodeCmd.Flags().String("bigTableTableName", "", "BigTable table name to store events in")
	bigTableTopicName = NodeCmd.Flags().String("bigTableTopicName", "", "GCP topic name to publish to")
	bigTableKeyPath = NodeCmd.Flags().String("bigTableKeyPath", "", "Path to json Service Account key")
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

// NodeCmd represents the node command
var NodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Run the guardiand node",
	Run:   runNode,
}

func runNode(cmd *cobra.Command, args []string) {
	if *unsafeDevMode {
		fmt.Print(devwarning)
	}

	common.LockMemory()
	common.SetRestrictiveUmask()

	// Refuse to run as root in production mode.
	if !*unsafeDevMode && os.Geteuid() == 0 {
		fmt.Println("can't run as uid 0")
		os.Exit(1)
	}

	// Set up logging. The go-log zap wrapper that libp2p uses is compatible with our
	// usage of zap in supervisor, which is nice.
	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	logger := zap.New(zapcore.NewCore(
		consoleEncoder{zapcore.NewConsoleEncoder(
			zap.NewDevelopmentEncoderConfig())},
		zapcore.AddSync(zapcore.Lock(os.Stderr)),
		zap.NewAtomicLevelAt(zapcore.Level(lvl))))

	if *unsafeDevMode {
		// Use the hostname as nodeName. For production, we don't want to do this to
		// prevent accidentally leaking sensitive hostnames.
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		*nodeName = hostname

		// Put node name into the log for development.
		logger = logger.Named(*nodeName)
	}

	// Override the default go-log config, which uses a magic environment variable.
	ipfslog.SetAllLoggers(lvl)

	// Register components for readiness checks.
	readiness.RegisterComponent(common.ReadinessEthSyncing)
	readiness.RegisterComponent(common.ReadinessSolanaSyncing)
	readiness.RegisterComponent(common.ReadinessTerraSyncing)
	if *unsafeDevMode {
		readiness.RegisterComponent(common.ReadinessAlgorandSyncing)
	}
	readiness.RegisterComponent(common.ReadinessBSCSyncing)
	readiness.RegisterComponent(common.ReadinessPolygonSyncing)
	readiness.RegisterComponent(common.ReadinessAvalancheSyncing)
	readiness.RegisterComponent(common.ReadinessOasisSyncing)
	readiness.RegisterComponent(common.ReadinessFantomSyncing)
	if *testnetMode {
		readiness.RegisterComponent(common.ReadinessEthRopstenSyncing)
	}

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
			logger.Error("status server crashed", zap.Error(http.ListenAndServe(*statusAddr, router)))
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
		*ethContract = devnet.GanacheWormholeContractAddress.Hex()
		*bscContract = devnet.GanacheWormholeContractAddress.Hex()
		*polygonContract = devnet.GanacheWormholeContractAddress.Hex()
		*avalancheContract = devnet.GanacheWormholeContractAddress.Hex()
		*oasisContract = devnet.GanacheWormholeContractAddress.Hex()
		*fantomContract = devnet.GanacheWormholeContractAddress.Hex()
	}

	// Verify flags

	if *nodeKeyPath == "" && !*unsafeDevMode { // In devnet mode, keys are deterministically generated.
		logger.Fatal("Please specify --nodeKey")
	}
	if *guardianKeyPath == "" {
		logger.Fatal("Please specify --guardianKey")
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
	if *polygonRPC == "" {
		logger.Fatal("Please specify --polygonRPC")
	}
	if *polygonContract == "" {
		logger.Fatal("Please specify --polygonContract")
	}
	if *avalancheRPC == "" {
		logger.Fatal("Please specify --avalancheRPC")
	}
	if *oasisRPC == "" {
		logger.Fatal("Please specify --oasisRPC")
	}
	if *fantomRPC == "" {
		logger.Fatal("Please specify --fantomRPC")
	}
	if *testnetMode {
		if *ethRopstenRPC == "" {
			logger.Fatal("Please specify --ethRopstenRPC")
		}
		if *ethRopstenContract == "" {
			logger.Fatal("Please specify --ethRopstenContract")
		}
	} else {
		if *ethRopstenRPC != "" {
			logger.Fatal("Please do not specify --ethRopstenRPC in non-testnet mode")
		}
		if *ethRopstenContract != "" {
			logger.Fatal("Please do not specify --ethRopstenContract in non-testnet mode")
		}
	}
	if *nodeName == "" {
		logger.Fatal("Please specify --nodeName")
	}

	if *solanaContract == "" {
		logger.Fatal("Please specify --solanaContract")
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
	if *terraContract == "" {
		logger.Fatal("Please specify --terraContract")
	}

	if *unsafeDevMode {
		if *algorandRPC == "" {
			logger.Fatal("Please specify --algorandRPC")
		}
		if *algorandToken == "" {
			logger.Fatal("Please specify --algorandToken")
		}
		if *algorandContract == "" {
			logger.Fatal("Please specify --algorandContract")
		}
	}

	if *bigTablePersistenceEnabled {
		if *bigTableGCPProject == "" {
			logger.Fatal("Please specify --bigTableGCPProject")
		}
		if *bigTableInstanceName == "" {
			logger.Fatal("Please specify --bigTableInstanceName")
		}
		if *bigTableTableName == "" {
			logger.Fatal("Please specify --bigTableTableName")
		}
		if *bigTableTopicName == "" {
			logger.Fatal("Please specify --bigTableTopicName")
		}
		if *bigTableKeyPath == "" {
			logger.Fatal("Please specify --bigTableKeyPath")
		}
	}

	// Complain about Infura on mainnet.
	//
	// As it turns out, Infura has a bug where it would sometimes incorrectly round
	// block timestamps, which causes consensus issues - the timestamp is part of
	// the VAA and nodes using Infura would sometimes derive an incorrect VAA,
	// accidentally attacking the network by signing a conflicting VAA.
	//
	// Node operators do not usually rely on Infura in the first place - doing
	// so is insecure, since nodes blindly trust the connected nodes to verify
	// on-chain message proofs. However, node operators sometimes used
	// Infura during migrations where their primary node was offline, causing
	// the aforementioned consensus oddities which were eventually found to
	// be Infura-related. This is generally to the detriment of network security
	// and a judgement call made by individual operators. In the case of Infura,
	// we know it's actively dangerous so let's make an opinionated argument.
	//
	// Insert "I'm a sign, not a cop" meme.
	//
	if strings.Contains(*ethRPC, "mainnet.infura.io") ||
		strings.Contains(*polygonRPC, "polygon-mainnet.infura.io") {
		logger.Fatal("Infura is known to send incorrect blocks - please use your own nodes")
	}

	ethContractAddr := eth_common.HexToAddress(*ethContract)
	bscContractAddr := eth_common.HexToAddress(*bscContract)
	polygonContractAddr := eth_common.HexToAddress(*polygonContract)
	ethRopstenContractAddr := eth_common.HexToAddress(*ethRopstenContract)
	avalancheContractAddr := eth_common.HexToAddress(*avalancheContract)
	oasisContractAddr := eth_common.HexToAddress(*oasisContract)
	fantomContractAddr := eth_common.HexToAddress(*fantomContract)
	solAddress, err := solana_types.PublicKeyFromBase58(*solanaContract)
	if err != nil {
		logger.Fatal("invalid Solana contract address", zap.Error(err))
	}

	// In devnet mode, we generate a deterministic guardian key and write it to disk.
	if *unsafeDevMode {
		gk, err := generateDevnetGuardianKey()
		if err != nil {
			logger.Fatal("failed to generate devnet guardian key", zap.Error(err))
		}

		err = writeGuardianKey(gk, "auto-generated deterministic devnet key", *guardianKeyPath, true)
		if err != nil {
			logger.Fatal("failed to write devnet guardian key", zap.Error(err))
		}
	}

	// Database
	dbPath := path.Join(*dataDir, "db")
	if err := os.MkdirAll(dbPath, 0700); err != nil {
		logger.Fatal("failed to create database directory", zap.Error(err))
	}
	db, err := db.Open(dbPath)
	if err != nil {
		logger.Fatal("failed to open database", zap.Error(err))
	}
	defer db.Close()

	// Guardian key
	gk, err := loadGuardianKey(*guardianKeyPath)
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

	// Inbound signed VAAs
	signedInC := make(chan *gossipv1.SignedVAAWithQuorum, 50)

	// Inbound observation requests
	obsvReqC := make(chan *gossipv1.ObservationRequest, 50)

	// Outbound observation requests
	obsvReqSendC := make(chan *gossipv1.ObservationRequest)

	// Injected VAAs (manually generated rather than created via observation)
	injectC := make(chan *vaa.VAA)

	// Guardian set state managed by processor
	gst := common.NewGuardianSetState()

	var notifier *discord.DiscordNotifier
	if *discordToken != "" {
		notifier, err = discord.NewDiscordNotifier(*discordToken, *discordChannel, logger)
		if err != nil {
			logger.Error("failed to initialize Discord bot", zap.Error(err))
		}
	}

	// Load p2p private key
	var priv crypto.PrivKey
	if *unsafeDevMode {
		idx, err := devnet.GetDevnetIndex()
		if err != nil {
			logger.Fatal("Failed to parse hostname - are we running in devnet?")
		}
		priv = devnet.DeterministicP2PPrivKeyByIndex(int64(idx))
	} else {
		priv, err = common.GetOrCreateNodeKey(logger, *nodeKeyPath)
		if err != nil {
			logger.Fatal("Failed to load node key", zap.Error(err))
		}
	}

	// Enable unless it is disabled. For devnet, only when --telemetryKey is set.
	if !*disableTelemetry && (!*unsafeDevMode || *unsafeDevMode && *telemetryKey != "") {
		logger.Info("Telemetry enabled")

		if *telemetryKey == "" {
			logger.Fatal("Please specify --telemetryKey")
		}

		creds, err := decryptTelemetryServiceAccount()
		if err != nil {
			logger.Fatal("Failed to decrypt telemetry service account", zap.Error(err))
		}

		// Get libp2p peer ID from private key
		pk := priv.GetPublic()
		peerID, err := peer.IDFromPublicKey(pk)
		if err != nil {
			logger.Fatal("Failed to get peer ID from private key", zap.Error(err))
		}

		tm, err := telemetry.New(context.Background(), telemetryProject, creds, map[string]string{
			"node_name":     *nodeName,
			"node_key":      peerID.Pretty(),
			"guardian_addr": guardianAddr,
			"network":       *p2pNetworkID,
			"version":       version.Version(),
		})
		if err != nil {
			logger.Fatal("Failed to initialize telemetry", zap.Error(err))
		}
		defer tm.Close()
		logger = tm.WrapLogger(logger)
	} else {
		logger.Info("Telemetry disabled")
	}

	// Redirect ipfs logs to plain zap
	ipfslog.SetPrimaryCore(logger.Core())

	// provides methods for reporting progress toward message attestation, and channels for receiving attestation lifecyclye events.
	attestationEvents := reporter.EventListener(logger)

	publicrpcService, publicrpcServer, err := publicrpcServiceRunnable(logger, *publicRPC, db, gst)

	if err != nil {
		log.Fatal("failed to create publicrpc service socket", zap.Error(err))
	}

	// local admin service socket
	adminService, err := adminServiceRunnable(logger, *adminSocketPath, injectC, signedInC, obsvReqSendC, db, gst)
	if err != nil {
		logger.Fatal("failed to create admin service socket", zap.Error(err))
	}

	publicwebService, err := publicwebServiceRunnable(logger, *publicWeb, *adminSocketPath, publicrpcServer,
		*tlsHostname, *tlsProdEnv, path.Join(*dataDir, "autocert"))
	if err != nil {
		log.Fatal("failed to create publicrpc service socket", zap.Error(err))
	}

	// Run supervisor.
	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "p2p", p2p.Run(
			obsvC, obsvReqC, obsvReqSendC, sendC, signedInC, priv, gk, gst, *p2pPort, *p2pNetworkID, *p2pBootstrap, *nodeName, *disableHeartbeatVerify, rootCtxCancel)); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "ethwatch",
			ethereum.NewEthWatcher(*ethRPC, ethContractAddr, "eth", common.ReadinessEthSyncing, vaa.ChainIDEthereum, lockC, setC, 1).Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "bscwatch",
			ethereum.NewEthWatcher(*bscRPC, bscContractAddr, "bsc", common.ReadinessBSCSyncing, vaa.ChainIDBSC, lockC, nil, 1).Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "polygonwatch",
			ethereum.NewEthWatcher(
				*polygonRPC, polygonContractAddr, "polygon", common.ReadinessPolygonSyncing, vaa.ChainIDPolygon, lockC, nil,
				// Special case: Polygon can fork like PoW Ethereum, and it's not clear what the safe number of blocks is
				//
				// Hardcode the minimum number of confirmations to 256 regardless of what the smart contract specifies to protect
				// developers from accidentally specifying an unsafe number of confirmations. We can remove this restriction as soon
				// as specific public guidance exists for Polygon developers.
				256).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "avalanchewatch",
			ethereum.NewEthWatcher(*avalancheRPC, avalancheContractAddr, "avalanche", common.ReadinessAvalancheSyncing, vaa.ChainIDAvalanche, lockC, nil, 1).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "oasiswatch",
			ethereum.NewEthWatcher(*oasisRPC, oasisContractAddr, "oasis", common.ReadinessOasisSyncing, vaa.ChainIDOasis, lockC, nil, 1).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "fantomwatch",
			ethereum.NewEthWatcher(*fantomRPC, fantomContractAddr, "fantom", common.ReadinessFantomSyncing, vaa.ChainIDFantom, lockC, nil, 1).Run); err != nil {
			return err
		}
		if *testnetMode {
			if err := supervisor.Run(ctx, "ethropstenwatch",
				ethereum.NewEthWatcher(*ethRopstenRPC, ethRopstenContractAddr, "ethropsten", common.ReadinessEthRopstenSyncing, vaa.ChainIDEthereumRopsten, lockC, setC, 1).Run); err != nil {
				return err
			}
		}

		// Start Terra watcher only if configured
		logger.Info("Starting Terra watcher")
		if err := supervisor.Run(ctx, "terrawatch",
			terra.NewWatcher(*terraWS, *terraLCD, *terraContract, lockC, setC).Run); err != nil {
			return err
		}

		if *unsafeDevMode {
			if err := supervisor.Run(ctx, "algorandwatch",
				algorand.NewWatcher(*algorandRPC, *algorandToken, *algorandContract, lockC, setC).Run); err != nil {
				return err
			}
		}

		if err := supervisor.Run(ctx, "solwatch-confirmed",
			solana.NewSolanaWatcher(*solanaWsRPC, *solanaRPC, solAddress, lockC, nil, rpc.CommitmentConfirmed).Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "solwatch-finalized",
			solana.NewSolanaWatcher(*solanaWsRPC, *solanaRPC, solAddress, lockC, obsvReqC, rpc.CommitmentFinalized).Run); err != nil {
			return err
		}

		p := processor.NewProcessor(ctx,
			db,
			lockC,
			setC,
			sendC,
			obsvC,
			injectC,
			signedInC,
			gk,
			gst,
			*unsafeDevMode,
			*devNumGuardians,
			*ethRPC,
			*terraLCD,
			*terraContract,
			attestationEvents,
			notifier,
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
		if *publicWeb != "" {
			if err := supervisor.Run(ctx, "publicweb", publicwebService); err != nil {
				return err
			}
		}

		if *bigTablePersistenceEnabled {
			bigTableConnection := &reporter.BigTableConnectionConfig{
				GcpProjectID:    *bigTableGCPProject,
				GcpInstanceName: *bigTableInstanceName,
				TableName:       *bigTableTableName,
				TopicName:       *bigTableTopicName,
				GcpKeyFilePath:  *bigTableKeyPath,
			}
			if err := supervisor.Run(ctx, "bigtable", reporter.BigTableWriter(attestationEvents, bigTableConnection)); err != nil {
				return err
			}
		}

		logger.Info("Started internal services")

		<-ctx.Done()
		return nil
	},
		// It's safer to crash and restart the process in case we encounter a panic,
		// rather than attempting to reschedule the runnable.
		supervisor.WithPropagatePanic)

	<-rootCtx.Done()
	logger.Info("root context cancelled, exiting...")
	// TODO: wait for things to shut down gracefully
}

func decryptTelemetryServiceAccount() ([]byte, error) {
	// Decrypt service account credentials
	key, err := base64.StdEncoding.DecodeString(*telemetryKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(telemetryServiceAccount)
	if err != nil {
		panic(err)
	}

	creds, err := common.DecryptAESGCM(ciphertext, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return creds, err
}
