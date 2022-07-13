package guardiand

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // #nosec G108 we are using a custom router (`router := mux.NewRouter()`) and thus not automatically expose pprof.
	"os"
	"path"
	"strings"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/notify/discord"
	"github.com/certusone/wormhole/node/pkg/telemetry"
	"github.com/certusone/wormhole/node/pkg/version"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap/zapcore"

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

	cosmwasm "github.com/certusone/wormhole/node/pkg/terra"

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

	auroraRPC      *string
	auroraContract *string

	fantomRPC      *string
	fantomContract *string

	avalancheRPC      *string
	avalancheContract *string

	oasisRPC      *string
	oasisContract *string

	karuraRPC      *string
	karuraContract *string

	acalaRPC      *string
	acalaContract *string

	klaytnRPC      *string
	klaytnContract *string

	celoRPC      *string
	celoContract *string

	moonbeamRPC      *string
	moonbeamContract *string

	neonRPC      *string
	neonContract *string

	terraWS       *string
	terraLCD      *string
	terraContract *string

	terra2WS       *string
	terra2LCD      *string
	terra2Contract *string

	injectiveWS       *string
	injectiveLCD      *string
	injectiveContract *string

	algorandIndexerRPC   *string
	algorandIndexerToken *string
	algorandAlgodRPC     *string
	algorandAlgodToken   *string
	algorandAppID        *uint64

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

	auroraRPC = NodeCmd.Flags().String("auroraRPC", "", "Aurora Websocket RPC URL")
	auroraContract = NodeCmd.Flags().String("auroraContract", "", "Aurora contract address")

	fantomRPC = NodeCmd.Flags().String("fantomRPC", "", "Fantom Websocket RPC URL")
	fantomContract = NodeCmd.Flags().String("fantomContract", "", "Fantom contract address")

	karuraRPC = NodeCmd.Flags().String("karuraRPC", "", "Karura RPC URL")
	karuraContract = NodeCmd.Flags().String("karuraContract", "", "Karura contract address")

	acalaRPC = NodeCmd.Flags().String("acalaRPC", "", "Acala RPC URL")
	acalaContract = NodeCmd.Flags().String("acalaContract", "", "Acala contract address")

	klaytnRPC = NodeCmd.Flags().String("klaytnRPC", "", "Klaytn RPC URL")
	klaytnContract = NodeCmd.Flags().String("klaytnContract", "", "Klaytn contract address")

	celoRPC = NodeCmd.Flags().String("celoRPC", "", "Celo RPC URL")
	celoContract = NodeCmd.Flags().String("celoContract", "", "Celo contract address")

	moonbeamRPC = NodeCmd.Flags().String("moonbeamRPC", "", "Moonbeam RPC URL")
	moonbeamContract = NodeCmd.Flags().String("moonbeamContract", "", "Moonbeam contract address")

	neonRPC = NodeCmd.Flags().String("neonRPC", "", "Neon RPC URL")
	neonContract = NodeCmd.Flags().String("neonContract", "", "Neon contract address")

	terraWS = NodeCmd.Flags().String("terraWS", "", "Path to terrad root for websocket connection")
	terraLCD = NodeCmd.Flags().String("terraLCD", "", "Path to LCD service root for http calls")
	terraContract = NodeCmd.Flags().String("terraContract", "", "Wormhole contract address on Terra blockchain")

	terra2WS = NodeCmd.Flags().String("terra2WS", "", "Path to terrad root for websocket connection")
	terra2LCD = NodeCmd.Flags().String("terra2LCD", "", "Path to LCD service root for http calls")
	terra2Contract = NodeCmd.Flags().String("terra2Contract", "", "Wormhole contract address on Terra 2 blockchain")

	injectiveWS = NodeCmd.Flags().String("injectiveWS", "", "Path to root for Injective websocket connection")
	injectiveLCD = NodeCmd.Flags().String("injectiveLCD", "", "Path to LCD service root for Injective http calls")
	injectiveContract = NodeCmd.Flags().String("injectiveContract", "", "Wormhole contract address on Injective blockchain")

	algorandIndexerRPC = NodeCmd.Flags().String("algorandIndexerRPC", "", "Algorand Indexer RPC URL")
	algorandIndexerToken = NodeCmd.Flags().String("algorandIndexerToken", "", "Algorand Indexer access token")
	algorandAlgodRPC = NodeCmd.Flags().String("algorandAlgodRPC", "", "Algorand Algod RPC URL")
	algorandAlgodToken = NodeCmd.Flags().String("algorandAlgodToken", "", "Algorand Algod access token")
	algorandAppID = NodeCmd.Flags().Uint64("algorandAppID", 0, "Algorand app id")

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
        |      Do not use --unsafeDevMode in prod.        |
        +++++++++++++++++++++++++++++++++++++++++++++++++++

`

// NodeCmd represents the node command
var NodeCmd = &cobra.Command{
	Use:   "node",
	Short: "Run the guardiand node",
	Run:   runNode,
}

// This variable may be overridden by the -X linker flag to "dev" in which case
// we enforce the --unsafeDevMode flag. Only development binaries/docker images
// are distributed. Production binaries are required to be built from source by
// guardians to reduce risk from a compromised builder.
var Build = "prod"

func runNode(cmd *cobra.Command, args []string) {
	if Build == "dev" && !*unsafeDevMode {
		fmt.Println("This is a development build. --unsafeDevMode must be enabled.")
		os.Exit(1)
	}

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
	if *solanaWsRPC != "" {
		readiness.RegisterComponent(common.ReadinessSolanaSyncing)
	}
	if *terraWS != "" {
		readiness.RegisterComponent(common.ReadinessTerraSyncing)
	}
	if *terra2WS != "" {
		readiness.RegisterComponent(common.ReadinessTerra2Syncing)
	}
	if *algorandIndexerRPC != "" {
		readiness.RegisterComponent(common.ReadinessAlgorandSyncing)
	}
	readiness.RegisterComponent(common.ReadinessBSCSyncing)
	readiness.RegisterComponent(common.ReadinessPolygonSyncing)
	readiness.RegisterComponent(common.ReadinessAvalancheSyncing)
	readiness.RegisterComponent(common.ReadinessOasisSyncing)
	readiness.RegisterComponent(common.ReadinessAuroraSyncing)
	readiness.RegisterComponent(common.ReadinessFantomSyncing)
	readiness.RegisterComponent(common.ReadinessKaruraSyncing)
	readiness.RegisterComponent(common.ReadinessAcalaSyncing)
	readiness.RegisterComponent(common.ReadinessKlaytnSyncing)
	readiness.RegisterComponent(common.ReadinessCeloSyncing)

	if *testnetMode {
		readiness.RegisterComponent(common.ReadinessEthRopstenSyncing)
		readiness.RegisterComponent(common.ReadinessMoonbeamSyncing)
		readiness.RegisterComponent(common.ReadinessNeonSyncing)
		readiness.RegisterComponent(common.ReadinessInjectiveSyncing)
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
			// SECURITY: If making changes, ensure that we always do `router := mux.NewRouter()` before this to avoid accidentally exposing pprof
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
		*auroraContract = devnet.GanacheWormholeContractAddress.Hex()
		*fantomContract = devnet.GanacheWormholeContractAddress.Hex()
		*karuraContract = devnet.GanacheWormholeContractAddress.Hex()
		*acalaContract = devnet.GanacheWormholeContractAddress.Hex()
		*klaytnContract = devnet.GanacheWormholeContractAddress.Hex()
		*celoContract = devnet.GanacheWormholeContractAddress.Hex()
		*moonbeamContract = devnet.GanacheWormholeContractAddress.Hex()
		*neonContract = devnet.GanacheWormholeContractAddress.Hex()
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
	if *fantomContract == "" && !*unsafeDevMode {
		logger.Fatal("Please specify --fantomContract")
	}
	if *auroraRPC == "" {
		logger.Fatal("Please specify --auroraRPC")
	}
	if *auroraContract == "" && !*unsafeDevMode {
		logger.Fatal("Please specify --auroraContract")
	}
	if *karuraRPC == "" {
		logger.Fatal("Please specify --karuraRPC")
	}
	if *karuraContract == "" && !*unsafeDevMode {
		logger.Fatal("Please specify --karuraContract")
	}
	if *acalaRPC == "" {
		logger.Fatal("Please specify --acalaRPC")
	}
	if *acalaContract == "" && !*unsafeDevMode {
		logger.Fatal("Please specify --acalaContract")
	}
	if *klaytnRPC == "" {
		logger.Fatal("Please specify --klaytnRPC")
	}
	if *klaytnContract == "" && !*unsafeDevMode {
		logger.Fatal("Please specify --klaytnContract")
	}
	if *celoRPC == "" {
		logger.Fatal("Please specify --celoRPC")
	}
	if *celoContract == "" && !*unsafeDevMode {
		logger.Fatal("Please specify --celoContract")
	}
	if *testnetMode {
		if *ethRopstenRPC == "" {
			logger.Fatal("Please specify --ethRopstenRPC")
		}
		if *ethRopstenContract == "" {
			logger.Fatal("Please specify --ethRopstenContract")
		}
		if *moonbeamRPC == "" {
			logger.Fatal("Please specify --moonbeamRPC")
		}
		if *moonbeamContract == "" {
			logger.Fatal("Please specify --moonbeamContract")
		}
		if *neonRPC == "" {
			logger.Fatal("Please specify --neonRPC")
		}
		if *neonContract == "" {
			logger.Fatal("Please specify --neonContract")
		}
		if *injectiveWS == "" {
			logger.Fatal("Please specify --injectiveWS")
		}
		if *injectiveLCD == "" {
			logger.Fatal("Please specify --injectiveLCD")
		}
		if *injectiveContract == "" {
			logger.Fatal("Please specify --injectiveContract")
		}
	} else {
		if *ethRopstenRPC != "" {
			logger.Fatal("Please do not specify --ethRopstenRPC in non-testnet mode")
		}
		if *ethRopstenContract != "" {
			logger.Fatal("Please do not specify --ethRopstenContract in non-testnet mode")
		}
		if *moonbeamRPC != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --moonbeamRPC")
		}
		if *moonbeamContract != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --moonbeamContract")
		}
		if *neonRPC != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --neonRPC")
		}
		if *neonContract != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --neonContract")
		}
		if *injectiveWS != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --injectiveWS")
		}
		if *injectiveLCD != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --injectiveLCD")
		}
		if *injectiveContract != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --injectiveContract")
		}
	}
	if *nodeName == "" {
		logger.Fatal("Please specify --nodeName")
	}

	// Solana, Terra Classic, Terra 2, and Algorand are optional in devnet
	if !*unsafeDevMode {

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

		if *terra2WS == "" {
			logger.Fatal("Please specify --terra2WS")
		}
		if *terra2LCD == "" {
			logger.Fatal("Please specify --terra2LCD")
		}
		if *terra2Contract == "" {
			logger.Fatal("Please specify --terra2Contract")
		}

		if *testnetMode {
			if *algorandIndexerRPC == "" {
				logger.Fatal("Please specify --algorandIndexerRPC")
			}
			if *algorandIndexerToken == "" {
				logger.Fatal("Please specify --algorandIndexerToken")
			}
			if *algorandAlgodRPC == "" {
				logger.Fatal("Please specify --algorandAlgodRPC")
			}
			if *algorandAlgodToken == "" {
				logger.Fatal("Please specify --algorandAlgodToken")
			}
			if *algorandAppID == 0 {
				logger.Fatal("Please specify --algorandAppID")
			}
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
	auroraContractAddr := eth_common.HexToAddress(*auroraContract)
	fantomContractAddr := eth_common.HexToAddress(*fantomContract)
	karuraContractAddr := eth_common.HexToAddress(*karuraContract)
	acalaContractAddr := eth_common.HexToAddress(*acalaContract)
	klaytnContractAddr := eth_common.HexToAddress(*klaytnContract)
	celoContractAddr := eth_common.HexToAddress(*celoContract)
	moonbeamContractAddr := eth_common.HexToAddress(*moonbeamContract)
	neonContractAddr := eth_common.HexToAddress(*neonContract)
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

	// Inbound observation requests from the p2p service (for all chains)
	obsvReqC := make(chan *gossipv1.ObservationRequest, 50)

	// Outbound observation requests
	obsvReqSendC := make(chan *gossipv1.ObservationRequest)

	// Injected VAAs (manually generated rather than created via observation)
	injectC := make(chan *vaa.VAA)

	// Guardian set state managed by processor
	gst := common.NewGuardianSetState()

	// Per-chain observation requests
	chainObsvReqC := make(map[vaa.ChainID]chan *gossipv1.ObservationRequest)

	// Observation request channel for each chain supporting observation requests.
	chainObsvReqC[vaa.ChainIDSolana] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDEthereum] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDTerra] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDTerra2] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDBSC] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDPolygon] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDAvalanche] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDOasis] = make(chan *gossipv1.ObservationRequest)
	if *testnetMode || *unsafeDevMode {
		chainObsvReqC[vaa.ChainIDAlgorand] = make(chan *gossipv1.ObservationRequest)
	}
	chainObsvReqC[vaa.ChainIDAurora] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDFantom] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDKarura] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDAcala] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDKlaytn] = make(chan *gossipv1.ObservationRequest)
	chainObsvReqC[vaa.ChainIDCelo] = make(chan *gossipv1.ObservationRequest)
	if *testnetMode {
		chainObsvReqC[vaa.ChainIDMoonbeam] = make(chan *gossipv1.ObservationRequest)
		chainObsvReqC[vaa.ChainIDNeon] = make(chan *gossipv1.ObservationRequest)
		chainObsvReqC[vaa.ChainIDEthereumRopsten] = make(chan *gossipv1.ObservationRequest)
		chainObsvReqC[vaa.ChainIDInjective] = make(chan *gossipv1.ObservationRequest)
	}

	// Multiplex observation requests to the appropriate chain
	go func() {
		for {
			select {
			case <-rootCtx.Done():
				return
			case req := <-obsvReqC:
				if channel, ok := chainObsvReqC[vaa.ChainID(req.ChainId)]; ok {
					channel <- req
				} else {
					logger.Error("unknown chain ID for reobservation request",
						zap.Uint32("chain_id", req.ChainId),
						zap.String("tx_hash", hex.EncodeToString(req.TxHash)))
				}
			}
		}
	}()

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
			ethereum.NewEthWatcher(*ethRPC, ethContractAddr, "eth", common.ReadinessEthSyncing, vaa.ChainIDEthereum, lockC, setC, 1, chainObsvReqC[vaa.ChainIDEthereum], *unsafeDevMode).Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "bscwatch",
			ethereum.NewEthWatcher(*bscRPC, bscContractAddr, "bsc", common.ReadinessBSCSyncing, vaa.ChainIDBSC, lockC, nil, 1, chainObsvReqC[vaa.ChainIDBSC], *unsafeDevMode).Run); err != nil {
			return err
		}

		polygonMinConfirmations := uint64(512)
		if *testnetMode {
			polygonMinConfirmations = 64
		}

		if err := supervisor.Run(ctx, "polygonwatch",
			ethereum.NewEthWatcher(*polygonRPC, polygonContractAddr, "polygon", common.ReadinessPolygonSyncing, vaa.ChainIDPolygon, lockC, nil, polygonMinConfirmations, chainObsvReqC[vaa.ChainIDPolygon], *unsafeDevMode).Run); err != nil {
			// Special case: Polygon can fork like PoW Ethereum, and it's not clear what the safe number of blocks is
			//
			// Hardcode the minimum number of confirmations to 512 regardless of what the smart contract specifies to protect
			// developers from accidentally specifying an unsafe number of confirmations. We can remove this restriction as soon
			// as specific public guidance exists for Polygon developers.
			return err
		}
		if err := supervisor.Run(ctx, "avalanchewatch",
			ethereum.NewEthWatcher(*avalancheRPC, avalancheContractAddr, "avalanche", common.ReadinessAvalancheSyncing, vaa.ChainIDAvalanche, lockC, nil, 1, chainObsvReqC[vaa.ChainIDAvalanche], *unsafeDevMode).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "oasiswatch",
			ethereum.NewEthWatcher(*oasisRPC, oasisContractAddr, "oasis", common.ReadinessOasisSyncing, vaa.ChainIDOasis, lockC, nil, 1, chainObsvReqC[vaa.ChainIDOasis], *unsafeDevMode).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "aurorawatch",
			ethereum.NewEthWatcher(*auroraRPC, auroraContractAddr, "aurora", common.ReadinessAuroraSyncing, vaa.ChainIDAurora, lockC, nil, 1, chainObsvReqC[vaa.ChainIDAurora], *unsafeDevMode).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "fantomwatch",
			ethereum.NewEthWatcher(*fantomRPC, fantomContractAddr, "fantom", common.ReadinessFantomSyncing, vaa.ChainIDFantom, lockC, nil, 1, chainObsvReqC[vaa.ChainIDFantom], *unsafeDevMode).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "karurawatch",
			ethereum.NewEthWatcher(*karuraRPC, karuraContractAddr, "karura", common.ReadinessKaruraSyncing, vaa.ChainIDKarura, lockC, nil, 1, chainObsvReqC[vaa.ChainIDKarura], *unsafeDevMode).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "acalawatch",
			ethereum.NewEthWatcher(*acalaRPC, acalaContractAddr, "acala", common.ReadinessAcalaSyncing, vaa.ChainIDAcala, lockC, nil, 1, chainObsvReqC[vaa.ChainIDAcala], *unsafeDevMode).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "klaytnwatch",
			ethereum.NewEthWatcher(*klaytnRPC, klaytnContractAddr, "klaytn", common.ReadinessKlaytnSyncing, vaa.ChainIDKlaytn, lockC, nil, 1, chainObsvReqC[vaa.ChainIDKlaytn], *unsafeDevMode).Run); err != nil {
			return err
		}
		if err := supervisor.Run(ctx, "celowatch",
			ethereum.NewEthWatcher(*celoRPC, celoContractAddr, "celo", common.ReadinessCeloSyncing, vaa.ChainIDCelo, lockC, nil, 1, chainObsvReqC[vaa.ChainIDCelo], *unsafeDevMode).Run); err != nil {
			return err
		}

		if *testnetMode {
			if err := supervisor.Run(ctx, "ethropstenwatch",
				ethereum.NewEthWatcher(*ethRopstenRPC, ethRopstenContractAddr, "ethropsten", common.ReadinessEthRopstenSyncing, vaa.ChainIDEthereumRopsten, lockC, nil, 1, chainObsvReqC[vaa.ChainIDEthereumRopsten], *unsafeDevMode).Run); err != nil {
				return err
			}
			if err := supervisor.Run(ctx, "moonbeamwatch",
				ethereum.NewEthWatcher(*moonbeamRPC, moonbeamContractAddr, "moonbeam", common.ReadinessMoonbeamSyncing, vaa.ChainIDMoonbeam, lockC, nil, 1, chainObsvReqC[vaa.ChainIDMoonbeam], *unsafeDevMode).Run); err != nil {
				return err
			}
			if err := supervisor.Run(ctx, "neonwatch",
				ethereum.NewEthWatcher(*neonRPC, neonContractAddr, "neon", common.ReadinessNeonSyncing, vaa.ChainIDNeon, lockC, nil, 32, chainObsvReqC[vaa.ChainIDNeon], *unsafeDevMode).Run); err != nil {
				return err
			}
		}

		if *terraWS != "" {
			logger.Info("Starting Terra watcher")
			if err := supervisor.Run(ctx, "terrawatch",
				cosmwasm.NewWatcher(*terraWS, *terraLCD, *terraContract, lockC, chainObsvReqC[vaa.ChainIDTerra], common.ReadinessTerraSyncing, vaa.ChainIDTerra).Run); err != nil {
				return err
			}
		}

		if *terra2WS != "" {
			logger.Info("Starting Terra 2 watcher")
			if err := supervisor.Run(ctx, "terra2watch",
				cosmwasm.NewWatcher(*terra2WS, *terra2LCD, *terra2Contract, lockC, chainObsvReqC[vaa.ChainIDTerra2], common.ReadinessTerra2Syncing, vaa.ChainIDTerra2).Run); err != nil {
				return err
			}
		}

		if *testnetMode {
			logger.Info("Starting Injective watcher")
			if err := supervisor.Run(ctx, "injectivewatch",
				cosmwasm.NewWatcher(*injectiveWS, *injectiveLCD, *injectiveContract, lockC, chainObsvReqC[vaa.ChainIDInjective], common.ReadinessInjectiveSyncing, vaa.ChainIDInjective).Run); err != nil {
				return err
			}
		}

		if *algorandIndexerRPC != "" {
			if err := supervisor.Run(ctx, "algorandwatch",
				algorand.NewWatcher(*algorandIndexerRPC, *algorandIndexerToken, *algorandAlgodRPC, *algorandAlgodToken, *algorandAppID, lockC, setC, chainObsvReqC[vaa.ChainIDAlgorand]).Run); err != nil {
				return err
			}
		}

		if *solanaWsRPC != "" {
			if err := supervisor.Run(ctx, "solwatch-confirmed",
				solana.NewSolanaWatcher(*solanaWsRPC, *solanaRPC, solAddress, lockC, nil, rpc.CommitmentConfirmed).Run); err != nil {
				return err
			}

			if err := supervisor.Run(ctx, "solwatch-finalized",
				solana.NewSolanaWatcher(*solanaWsRPC, *solanaRPC, solAddress, lockC, chainObsvReqC[vaa.ChainIDSolana], rpc.CommitmentFinalized).Run); err != nil {
				return err
			}
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
