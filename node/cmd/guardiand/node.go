package guardiand

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	_ "net/http/pprof" // #nosec G108 we are using a custom router (`router := mux.NewRouter()`) and thus not automatically expose pprof.
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"

	"github.com/certusone/wormhole/node/pkg/watchers/wormchain"

	"github.com/certusone/wormhole/node/pkg/watchers/cosmwasm"

	"github.com/certusone/wormhole/node/pkg/watchers/algorand"
	"github.com/certusone/wormhole/node/pkg/watchers/aptos"
	"github.com/certusone/wormhole/node/pkg/watchers/evm"
	"github.com/certusone/wormhole/node/pkg/watchers/near"
	"github.com/certusone/wormhole/node/pkg/watchers/solana"
	"github.com/certusone/wormhole/node/pkg/watchers/sui"
	"github.com/certusone/wormhole/node/pkg/wormconn"

	"github.com/benbjohnson/clock"
	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/notify/discord"
	"github.com/certusone/wormhole/node/pkg/telemetry"
	"github.com/certusone/wormhole/node/pkg/version"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap/zapcore"

	solana_types "github.com/gagliardetto/solana-go"
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/certusone/wormhole/node/pkg/accountant"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/certusone/wormhole/node/pkg/governor"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/processor"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/readiness"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	cosmoscrypto "github.com/cosmos/cosmos-sdk/crypto/types"
	eth_common "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	ipfslog "github.com/ipfs/go-log/v2"
)

var (
	p2pNetworkID *string
	p2pPort      *uint
	p2pBootstrap *string

	nodeKeyPath *string

	adminSocketPath      *string
	publicGRPCSocketPath *string

	dataDir *string

	statusAddr *string

	guardianKeyPath *string
	solanaContract  *string

	ethRPC      *string
	ethContract *string

	bscRPC      *string
	bscContract *string

	polygonRPC                      *string
	polygonContract                 *string
	polygonRootChainRpc             *string
	polygonRootChainContractAddress *string

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

	xplaWS       *string
	xplaLCD      *string
	xplaContract *string

	algorandIndexerRPC   *string
	algorandIndexerToken *string
	algorandAlgodRPC     *string
	algorandAlgodToken   *string
	algorandAppID        *uint64

	nearRPC      *string
	nearContract *string

	wormchainWS            *string
	wormchainLCD           *string
	wormchainURL           *string
	wormchainKeyPath       *string
	wormchainKeyPassPhrase *string

	accountantContract     *string
	accountantWS           *string
	accountantCheckEnabled *bool

	aptosRPC     *string
	aptosAccount *string
	aptosHandle  *string

	suiRPC     *string
	suiWS      *string
	suiAccount *string
	suiPackage *string

	solanaRPC *string

	pythnetContract *string
	pythnetRPC      *string
	pythnetWS       *string

	arbitrumRPC      *string
	arbitrumContract *string

	optimismRPC                *string
	optimismContract           *string
	optimismCtcRpc             *string
	optimismCtcContractAddress *string

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

	chainGovernorEnabled *bool
)

func init() {
	p2pNetworkID = NodeCmd.Flags().String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort = NodeCmd.Flags().Uint("port", 8999, "P2P UDP listener port")
	p2pBootstrap = NodeCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")

	statusAddr = NodeCmd.Flags().String("statusAddr", "[::]:6060", "Listen address for status server (disabled if blank)")

	nodeKeyPath = NodeCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	adminSocketPath = NodeCmd.Flags().String("adminSocket", "", "Admin gRPC service UNIX domain socket path")
	publicGRPCSocketPath = NodeCmd.Flags().String("publicGRPCSocket", "", "Public gRPC service UNIX domain socket path")

	dataDir = NodeCmd.Flags().String("dataDir", "", "Data directory")

	guardianKeyPath = NodeCmd.Flags().String("guardianKey", "", "Path to guardian key (required)")
	solanaContract = NodeCmd.Flags().String("solanaContract", "", "Address of the Solana program (required)")

	ethRPC = NodeCmd.Flags().String("ethRPC", "", "Ethereum RPC URL")
	ethContract = NodeCmd.Flags().String("ethContract", "", "Ethereum contract address")

	bscRPC = NodeCmd.Flags().String("bscRPC", "", "Binance Smart Chain RPC URL")
	bscContract = NodeCmd.Flags().String("bscContract", "", "Binance Smart Chain contract address")

	polygonRPC = NodeCmd.Flags().String("polygonRPC", "", "Polygon RPC URL")
	polygonContract = NodeCmd.Flags().String("polygonContract", "", "Polygon contract address")
	polygonRootChainRpc = NodeCmd.Flags().String("polygonRootChainRpc", "", "Polygon root chain RPC")
	polygonRootChainContractAddress = NodeCmd.Flags().String("polygonRootChainContractAddress", "", "Polygon root chain contract address")

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

	xplaWS = NodeCmd.Flags().String("xplaWS", "", "Path to root for XPLA websocket connection")
	xplaLCD = NodeCmd.Flags().String("xplaLCD", "", "Path to LCD service root for XPLA http calls")
	xplaContract = NodeCmd.Flags().String("xplaContract", "", "Wormhole contract address on XPLA blockchain")

	algorandIndexerRPC = NodeCmd.Flags().String("algorandIndexerRPC", "", "Algorand Indexer RPC URL")
	algorandIndexerToken = NodeCmd.Flags().String("algorandIndexerToken", "", "Algorand Indexer access token")
	algorandAlgodRPC = NodeCmd.Flags().String("algorandAlgodRPC", "", "Algorand Algod RPC URL")
	algorandAlgodToken = NodeCmd.Flags().String("algorandAlgodToken", "", "Algorand Algod access token")
	algorandAppID = NodeCmd.Flags().Uint64("algorandAppID", 0, "Algorand app id")

	nearRPC = NodeCmd.Flags().String("nearRPC", "", "near RPC URL")
	nearContract = NodeCmd.Flags().String("nearContract", "", "near contract")

	wormchainWS = NodeCmd.Flags().String("wormchainWS", "", "Path to wormchaind root for websocket connection")
	wormchainLCD = NodeCmd.Flags().String("wormchainLCD", "", "Path to LCD service root for http calls")
	wormchainURL = NodeCmd.Flags().String("wormchainURL", "", "wormhole-chain gRPC URL")
	wormchainKeyPath = NodeCmd.Flags().String("wormchainKeyPath", "", "path to wormhole-chain private key for signing transactions")
	wormchainKeyPassPhrase = NodeCmd.Flags().String("wormchainKeyPassPhrase", "", "pass phrase used to unarmor the wormchain key file")

	accountantWS = NodeCmd.Flags().String("accountantWS", "", "Websocket used to listen to the accountant smart contract on wormchain")
	accountantContract = NodeCmd.Flags().String("accountantContract", "", "Address of the accountant smart contract on wormchain")
	accountantCheckEnabled = NodeCmd.Flags().Bool("accountantCheckEnabled", false, "Should accountant be enforced on transfers")

	aptosRPC = NodeCmd.Flags().String("aptosRPC", "", "aptos RPC URL")
	aptosAccount = NodeCmd.Flags().String("aptosAccount", "", "aptos account")
	aptosHandle = NodeCmd.Flags().String("aptosHandle", "", "aptos handle")

	suiRPC = NodeCmd.Flags().String("suiRPC", "", "sui RPC URL")
	suiWS = NodeCmd.Flags().String("suiWS", "", "sui WS URL")
	suiAccount = NodeCmd.Flags().String("suiAccount", "", "sui account")
	suiPackage = NodeCmd.Flags().String("suiPackage", "", "sui package")

	solanaRPC = NodeCmd.Flags().String("solanaRPC", "", "Solana RPC URL (required)")

	pythnetContract = NodeCmd.Flags().String("pythnetContract", "", "Address of the PythNet program (required)")
	pythnetRPC = NodeCmd.Flags().String("pythnetRPC", "", "PythNet RPC URL (required)")
	pythnetWS = NodeCmd.Flags().String("pythnetWS", "", "PythNet WS URL")

	arbitrumRPC = NodeCmd.Flags().String("arbitrumRPC", "", "Arbitrum RPC URL")
	arbitrumContract = NodeCmd.Flags().String("arbitrumContract", "", "Arbitrum contract address")

	optimismRPC = NodeCmd.Flags().String("optimismRPC", "", "Optimism RPC URL")
	optimismContract = NodeCmd.Flags().String("optimismContract", "", "Optimism contract address")
	optimismCtcRpc = NodeCmd.Flags().String("optimismCtcRpc", "", "Optimism CTC RPC")
	optimismCtcContractAddress = NodeCmd.Flags().String("optimismCtcContractAddress", "", "Optimism CTC contract address")

	logLevel = NodeCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")

	unsafeDevMode = NodeCmd.Flags().Bool("unsafeDevMode", false, "Launch node in unsafe, deterministic devnet mode")
	testnetMode = NodeCmd.Flags().Bool("testnetMode", false, "Launch node in testnet mode (enables testnet-only features)")
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

	chainGovernorEnabled = NodeCmd.Flags().Bool("chainGovernorEnabled", false, "Run the chain governor")
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

// observationRequestBufferSize is the buffer size of the per-network reobservation channel
const observationRequestBufferSize = 25

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
			logger.Error("status server crashed", zap.Error(http.ListenAndServe(*statusAddr, router))) // #nosec G114 local status server not vulnerable to DoS attack
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
		*ethContract = unsafeDevModeEvmContractAddress(*ethContract)
		*bscContract = unsafeDevModeEvmContractAddress(*bscContract)
		*polygonContract = unsafeDevModeEvmContractAddress(*polygonContract)
		*avalancheContract = unsafeDevModeEvmContractAddress(*avalancheContract)
		*oasisContract = unsafeDevModeEvmContractAddress(*oasisContract)
		*auroraContract = unsafeDevModeEvmContractAddress(*auroraContract)
		*fantomContract = unsafeDevModeEvmContractAddress(*fantomContract)
		*karuraContract = unsafeDevModeEvmContractAddress(*karuraContract)
		*acalaContract = unsafeDevModeEvmContractAddress(*acalaContract)
		*klaytnContract = unsafeDevModeEvmContractAddress(*klaytnContract)
		*celoContract = unsafeDevModeEvmContractAddress(*celoContract)
		*moonbeamContract = unsafeDevModeEvmContractAddress(*moonbeamContract)
		*neonContract = unsafeDevModeEvmContractAddress(*neonContract)
		*arbitrumContract = unsafeDevModeEvmContractAddress(*arbitrumContract)
		*optimismContract = unsafeDevModeEvmContractAddress(*optimismContract)
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
	if *adminSocketPath == *publicGRPCSocketPath {
		logger.Fatal("--adminSocket must not equal --publicGRPCSocket")
	}
	if (*publicRPC != "" || *publicWeb != "") && *publicGRPCSocketPath == "" {
		logger.Fatal("If either --publicRPC or --publicWeb is specified, --publicGRPCSocket must also be specified")
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
	if *nearRPC != "" {
		if *nearContract == "" {
			logger.Fatal("If --nearRPC is specified, then --nearContract must be specified")
		}
	} else if *nearContract != "" {
		logger.Fatal("If --nearRPC is not specified, then --nearContract must not be specified")
	}
	if *moonbeamRPC == "" {
		logger.Fatal("Please specify --moonbeamRPC")
	}
	if *moonbeamContract == "" {
		logger.Fatal("Please specify --moonbeamContract")
	}
	if *arbitrumRPC == "" {
		logger.Fatal("Please specify --arbitrumRPC")
	}
	if *arbitrumContract == "" {
		logger.Fatal("Please specify --arbitrumContract")
	}
	if *xplaWS != "" {
		if *xplaLCD == "" || *xplaContract == "" {
			logger.Fatal("If --xplaWS is specified, then --xplaLCD and --xplaContract must be specified")
		}
	} else if *xplaLCD != "" || *xplaContract != "" {
		logger.Fatal("If --xplaWS is not specified, then --xplaLCD and --xplaContract must not be specified")
	}
	if *wormchainWS != "" {
		if *wormchainLCD == "" {
			logger.Fatal("If --wormchainWS is specified, then --wormchainLCD must be specified")
		}
	} else if *wormchainLCD != "" {
		logger.Fatal("If --wormchainWS is not specified, then --wormchainLCD must not be specified")
	}

	if *aptosRPC != "" {
		if *aptosAccount == "" {
			logger.Fatal("If --aptosRPC is specified, then --aptosAccount must be specified")
		}
		if *aptosHandle == "" {
			logger.Fatal("If --aptosRPC is specified, then --aptosHandle must be specified")
		}
	}
	if *suiRPC != "" {
		if *suiWS == "" {
			logger.Fatal("If --suiRPC is specified, then --suiWS must be specified")
		}
		if *suiAccount == "" {
			logger.Fatal("If --suiRPC is specified, then --suiAccount must be specified")
		}
		if *suiPackage == "" && !*unsafeDevMode {
			logger.Fatal("If --suiRPC is specified, then --suiPackage must be specified")
		}
	}

	if (*optimismRPC == "") != (*optimismContract == "") {
		logger.Fatal("Both --optimismContract and --optimismRPC must be set together or both unset")
	}

	if *testnetMode {
		if *neonRPC == "" {
			logger.Fatal("Please specify --neonRPC")
		}
		if *neonContract == "" {
			logger.Fatal("Please specify --neonContract")
		}
	} else {
		if *neonRPC != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --neonRPC")
		}
		if *neonContract != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --neonContract")
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
		if *solanaRPC == "" {
			logger.Fatal("Please specify --solanaRPC")
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

		if *algorandIndexerRPC == "" {
			logger.Fatal("Please specify --algorandIndexerRPC")
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

		if *pythnetContract == "" {
			logger.Fatal("Please specify --pythnetContract")
		}
		if *pythnetRPC == "" {
			logger.Fatal("Please specify --pythnetRPC")
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
	arbitrumContractAddr := eth_common.HexToAddress(*arbitrumContract)
	optimismContractAddr := eth_common.HexToAddress(*optimismContract)
	solAddress, err := solana_types.PublicKeyFromBase58(*solanaContract)
	if err != nil {
		logger.Fatal("invalid Solana contract address", zap.Error(err))
	}
	var pythnetAddress solana_types.PublicKey
	pythnetAddress, err = solana_types.PublicKeyFromBase58(*pythnetContract)
	if err != nil {
		logger.Fatal("invalid PythNet contract address", zap.Error(err))
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

	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	go func() {
		<-sigterm
		logger.Info("Received sigterm. exiting.")
		rootCtxCancel()
	}()

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
	obsvReqC := make(chan *gossipv1.ObservationRequest, common.ObsvReqChannelSize)

	// Outbound observation requests
	obsvReqSendC := make(chan *gossipv1.ObservationRequest, common.ObsvReqChannelSize)

	// Injected VAAs (manually generated rather than created via observation)
	injectC := make(chan *vaa.VAA)

	// Guardian set state managed by processor
	gst := common.NewGuardianSetState(nil)

	// Per-chain observation requests
	chainObsvReqC := make(map[vaa.ChainID]chan *gossipv1.ObservationRequest)

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

	// If the wormchain sending info is configured, connect to it.
	var wormchainKey cosmoscrypto.PrivKey
	var wormchainConn *wormconn.ClientConn
	if *wormchainURL != "" {
		if *wormchainKeyPath == "" {
			logger.Fatal("if wormchainURL is specified, wormchainKeyPath is required")
		}

		if *wormchainKeyPassPhrase == "" {
			logger.Fatal("if wormchainURL is specified, wormchainKeyPassPhrase is required")
		}

		// Load the wormchain key.
		wormchainKeyPathName := *wormchainKeyPath
		if *unsafeDevMode {
			idx, err := devnet.GetDevnetIndex()
			if err != nil {
				logger.Fatal("failed to get devnet index", zap.Error(err))
			}
			wormchainKeyPathName = fmt.Sprint(*wormchainKeyPath, idx)
		}

		logger.Debug("acct: loading key file", zap.String("key path", wormchainKeyPathName))
		wormchainKey, err = wormconn.LoadWormchainPrivKey(wormchainKeyPathName, *wormchainKeyPassPhrase)
		if err != nil {
			logger.Fatal("failed to load devnet wormchain private key", zap.Error(err))
		}

		// Connect to wormchain.
		logger.Info("Connecting to wormchain", zap.String("wormchainURL", *wormchainURL), zap.String("wormchainKeyPath", wormchainKeyPathName))
		wormchainConn, err = wormconn.NewConn(rootCtx, *wormchainURL, wormchainKey)
		if err != nil {
			logger.Fatal("failed to connect to wormchain", zap.Error(err))
		}
	}

	// Set up the accountant. If the accountant smart contract is configured, we will instantiate the accountant and VAAs
	// will be passed to it for processing. It will forward all token bridge transfers to the accountant contract.
	// If accountantCheckEnabled is set to true, token bridge transfers will not be signed and published until they
	// are approved by the accountant smart contract.

	// TODO: Use this once PR #1931 is merged.
	//acctReadC, acctWriteC := makeChannelPair[*common.MessagePublication](0)
	acctChan := make(chan *common.MessagePublication)
	var acctReadC <-chan *common.MessagePublication = acctChan
	var acctWriteC chan<- *common.MessagePublication = acctChan

	var acct *accountant.Accountant
	if *accountantContract != "" {
		if *accountantWS == "" {
			logger.Fatal("acct: if accountantContract is specified, accountantWS is required")
		}
		if *wormchainLCD == "" {
			logger.Fatal("acct: if accountantContract is specified, wormchainLCD is required")
		}
		if wormchainConn == nil {
			logger.Fatal("acct: if accountantContract is specified, the wormchain sending connection must be enabled")
		}
		if *accountantCheckEnabled {
			logger.Info("acct: accountant is enabled and will be enforced")
		} else {
			logger.Info("acct: accountant is enabled but will not be enforced")
		}
		env := accountant.MainNetMode
		if *testnetMode {
			env = accountant.TestNetMode
		} else if *unsafeDevMode {
			env = accountant.DevNetMode
		}
		acct = accountant.NewAccountant(
			rootCtx,
			logger,
			db,
			*accountantContract,
			*accountantWS,
			wormchainConn,
			*accountantCheckEnabled,
			gk,
			gst,
			acctWriteC,
			env,
		)
	} else {
		logger.Info("acct: accountant is disabled")
	}

	var gov *governor.ChainGovernor
	if *chainGovernorEnabled {
		logger.Info("chain governor is enabled")
		env := governor.MainNetMode
		if *testnetMode {
			env = governor.TestNetMode
		} else if *unsafeDevMode {
			env = governor.DevNetMode
		}
		gov = governor.NewChainGovernor(logger, db, env)
	} else {
		logger.Info("chain governor is disabled")
	}

	// local admin service socket
	adminService, err := adminServiceRunnable(logger, *adminSocketPath, injectC, signedInC, obsvReqSendC, db, gst, gov, gk, ethRPC, ethContract)
	if err != nil {
		logger.Fatal("failed to create admin service socket", zap.Error(err))
	}

	// Run supervisor.
	supervisor.New(rootCtx, logger, func(ctx context.Context) error {
		if err := supervisor.Run(ctx, "p2p", p2p.Run(
			obsvC, obsvReqC, obsvReqSendC, sendC, signedInC, priv, gk, gst, *p2pPort, *p2pNetworkID, *p2pBootstrap, *nodeName, *disableHeartbeatVerify, rootCtxCancel, acct, gov, nil, nil)); err != nil {
			return err
		}

		// For each chain that wants a watcher, we:
		// - create and register a component for readiness checks.
		// - create an observation request channel.
		// - create the watcher.
		//
		// NOTE:  The "none" is a special indicator to disable a watcher until it is desirable to turn it back on.

		var ethWatcher *evm.Watcher
		if shouldStart(ethRPC) {
			logger.Info("Starting Ethereum watcher")
			readiness.RegisterComponent(common.ReadinessEthSyncing)
			chainObsvReqC[vaa.ChainIDEthereum] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			ethWatcher = evm.NewEthWatcher(*ethRPC, ethContractAddr, "eth", common.ReadinessEthSyncing, vaa.ChainIDEthereum, lockC, setC, chainObsvReqC[vaa.ChainIDEthereum], *unsafeDevMode)
			if err := supervisor.Run(ctx, "ethwatch",
				common.WrapWithScissors(ethWatcher.Run, "ethwatch")); err != nil {
				return err
			}
		}

		if shouldStart(bscRPC) {
			logger.Info("Starting BSC watcher")
			readiness.RegisterComponent(common.ReadinessBSCSyncing)
			chainObsvReqC[vaa.ChainIDBSC] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			bscWatcher := evm.NewEthWatcher(*bscRPC, bscContractAddr, "bsc", common.ReadinessBSCSyncing, vaa.ChainIDBSC, lockC, nil, chainObsvReqC[vaa.ChainIDBSC], *unsafeDevMode)
			bscWatcher.SetWaitForConfirmations(true)
			if err := supervisor.Run(ctx, "bscwatch", common.WrapWithScissors(bscWatcher.Run, "bscwatch")); err != nil {
				return err
			}
		}

		if shouldStart(polygonRPC) {
			// Checkpointing is required in mainnet, so we don't need to wait for confirmations.
			waitForConfirmations := *unsafeDevMode || *testnetMode
			if !waitForConfirmations && *polygonRootChainRpc == "" {
				log.Fatal("Polygon checkpointing is required in mainnet")
			}
			logger.Info("Starting Polygon watcher")
			readiness.RegisterComponent(common.ReadinessPolygonSyncing)
			chainObsvReqC[vaa.ChainIDPolygon] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			polygonWatcher := evm.NewEthWatcher(*polygonRPC, polygonContractAddr, "polygon", common.ReadinessPolygonSyncing, vaa.ChainIDPolygon, lockC, nil, chainObsvReqC[vaa.ChainIDPolygon], *unsafeDevMode)
			polygonWatcher.SetWaitForConfirmations(waitForConfirmations)
			if err := polygonWatcher.SetRootChainParams(*polygonRootChainRpc, *polygonRootChainContractAddress); err != nil {
				return err
			}
			if err := supervisor.Run(ctx, "polygonwatch", common.WrapWithScissors(polygonWatcher.Run, "polygonwatch")); err != nil {
				return err
			}
		}
		if shouldStart(avalancheRPC) {
			logger.Info("Starting Avalanche watcher")
			readiness.RegisterComponent(common.ReadinessAvalancheSyncing)
			chainObsvReqC[vaa.ChainIDAvalanche] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "avalanchewatch",
				common.WrapWithScissors(evm.NewEthWatcher(*avalancheRPC, avalancheContractAddr, "avalanche", common.ReadinessAvalancheSyncing, vaa.ChainIDAvalanche, lockC, nil, chainObsvReqC[vaa.ChainIDAvalanche], *unsafeDevMode).Run, "avalanchewatch")); err != nil {
				return err
			}
		}
		if shouldStart(oasisRPC) {
			logger.Info("Starting Oasis watcher")
			chainObsvReqC[vaa.ChainIDOasis] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "oasiswatch",
				common.WrapWithScissors(evm.NewEthWatcher(*oasisRPC, oasisContractAddr, "oasis", common.ReadinessOasisSyncing, vaa.ChainIDOasis, lockC, nil, chainObsvReqC[vaa.ChainIDOasis], *unsafeDevMode).Run, "oasiswatch")); err != nil {
				return err
			}
		}
		if shouldStart(auroraRPC) {
			logger.Info("Starting Aurora watcher")
			readiness.RegisterComponent(common.ReadinessAuroraSyncing)
			chainObsvReqC[vaa.ChainIDAurora] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "aurorawatch",
				common.WrapWithScissors(evm.NewEthWatcher(*auroraRPC, auroraContractAddr, "aurora", common.ReadinessAuroraSyncing, vaa.ChainIDAurora, lockC, nil, chainObsvReqC[vaa.ChainIDAurora], *unsafeDevMode).Run, "aurorawatch")); err != nil {
				return err
			}
		}
		if shouldStart(fantomRPC) {
			logger.Info("Starting Fantom watcher")
			readiness.RegisterComponent(common.ReadinessFantomSyncing)
			chainObsvReqC[vaa.ChainIDFantom] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "fantomwatch",
				common.WrapWithScissors(evm.NewEthWatcher(*fantomRPC, fantomContractAddr, "fantom", common.ReadinessFantomSyncing, vaa.ChainIDFantom, lockC, nil, chainObsvReqC[vaa.ChainIDFantom], *unsafeDevMode).Run, "fantomwatch")); err != nil {
				return err
			}
		}
		if shouldStart(karuraRPC) {
			logger.Info("Starting Karura watcher")
			readiness.RegisterComponent(common.ReadinessKaruraSyncing)
			chainObsvReqC[vaa.ChainIDKarura] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "karurawatch",
				common.WrapWithScissors(evm.NewEthWatcher(*karuraRPC, karuraContractAddr, "karura", common.ReadinessKaruraSyncing, vaa.ChainIDKarura, lockC, nil, chainObsvReqC[vaa.ChainIDKarura], *unsafeDevMode).Run, "karurawatch")); err != nil {
				return err
			}
		}
		if shouldStart(acalaRPC) {
			logger.Info("Starting Acala watcher")
			readiness.RegisterComponent(common.ReadinessAcalaSyncing)
			chainObsvReqC[vaa.ChainIDAcala] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "acalawatch",
				common.WrapWithScissors(evm.NewEthWatcher(*acalaRPC, acalaContractAddr, "acala", common.ReadinessAcalaSyncing, vaa.ChainIDAcala, lockC, nil, chainObsvReqC[vaa.ChainIDAcala], *unsafeDevMode).Run, "acalawatch")); err != nil {
				return err
			}
		}
		if shouldStart(klaytnRPC) {
			logger.Info("Starting Klaytn watcher")
			readiness.RegisterComponent(common.ReadinessKlaytnSyncing)
			chainObsvReqC[vaa.ChainIDKlaytn] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "klaytnwatch",
				common.WrapWithScissors(evm.NewEthWatcher(*klaytnRPC, klaytnContractAddr, "klaytn", common.ReadinessKlaytnSyncing, vaa.ChainIDKlaytn, lockC, nil, chainObsvReqC[vaa.ChainIDKlaytn], *unsafeDevMode).Run, "klaytnwatch")); err != nil {
				return err
			}
		}
		if shouldStart(celoRPC) {
			logger.Info("Starting Celo watcher")
			readiness.RegisterComponent(common.ReadinessCeloSyncing)
			chainObsvReqC[vaa.ChainIDCelo] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "celowatch",
				common.WrapWithScissors(evm.NewEthWatcher(*celoRPC, celoContractAddr, "celo", common.ReadinessCeloSyncing, vaa.ChainIDCelo, lockC, nil, chainObsvReqC[vaa.ChainIDCelo], *unsafeDevMode).Run, "celowatch")); err != nil {
				return err
			}
		}
		if shouldStart(moonbeamRPC) {
			logger.Info("Starting Moonbeam watcher")
			readiness.RegisterComponent(common.ReadinessMoonbeamSyncing)
			chainObsvReqC[vaa.ChainIDMoonbeam] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "moonbeamwatch",
				common.WrapWithScissors(evm.NewEthWatcher(*moonbeamRPC, moonbeamContractAddr, "moonbeam", common.ReadinessMoonbeamSyncing, vaa.ChainIDMoonbeam, lockC, nil, chainObsvReqC[vaa.ChainIDMoonbeam], *unsafeDevMode).Run, "moonbeamwatch")); err != nil {
				return err
			}
		}
		if shouldStart(arbitrumRPC) {
			if ethWatcher == nil {
				log.Fatalf("if arbitrum is enabled then ethereum must also be enabled.")
			}
			logger.Info("Starting Arbitrum watcher")
			readiness.RegisterComponent(common.ReadinessArbitrumSyncing)
			chainObsvReqC[vaa.ChainIDArbitrum] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			arbitrumWatcher := evm.NewEthWatcher(*arbitrumRPC, arbitrumContractAddr, "arbitrum", common.ReadinessArbitrumSyncing, vaa.ChainIDArbitrum, lockC, nil, chainObsvReqC[vaa.ChainIDArbitrum], *unsafeDevMode)
			arbitrumWatcher.SetL1Finalizer(ethWatcher)
			if err := supervisor.Run(ctx, "arbitrumwatch", common.WrapWithScissors(arbitrumWatcher.Run, "arbitrumwatch")); err != nil {
				return err
			}
		}
		if shouldStart(optimismRPC) {
			if ethWatcher == nil {
				log.Fatalf("if optimism is enabled then ethereum must also be enabled.")
			}
			if !*unsafeDevMode {
				if *optimismCtcRpc == "" || *optimismCtcContractAddress == "" {
					log.Fatalf("--optimismCtcRpc and --optimismCtcContractAddress both need to be set.")
				}
			}
			logger.Info("Starting Optimism watcher")
			readiness.RegisterComponent(common.ReadinessOptimismSyncing)
			chainObsvReqC[vaa.ChainIDOptimism] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			optimismWatcher := evm.NewEthWatcher(*optimismRPC, optimismContractAddr, "optimism", common.ReadinessOptimismSyncing, vaa.ChainIDOptimism, lockC, nil, chainObsvReqC[vaa.ChainIDOptimism], *unsafeDevMode)
			optimismWatcher.SetL1Finalizer(ethWatcher)
			if err := optimismWatcher.SetRootChainParams(*optimismCtcRpc, *optimismCtcContractAddress); err != nil {
				return err
			}
			if err := supervisor.Run(ctx, "optimismwatch", common.WrapWithScissors(optimismWatcher.Run, "optimismwatch")); err != nil {
				return err
			}
		}

		if shouldStart(terraWS) {
			logger.Info("Starting Terra watcher")
			readiness.RegisterComponent(common.ReadinessTerraSyncing)
			chainObsvReqC[vaa.ChainIDTerra] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "terrawatch",
				common.WrapWithScissors(cosmwasm.NewWatcher(*terraWS, *terraLCD, *terraContract, lockC, chainObsvReqC[vaa.ChainIDTerra], common.ReadinessTerraSyncing, vaa.ChainIDTerra).Run, "terrawatch")); err != nil {
				return err
			}
		}

		if shouldStart(terra2WS) {
			logger.Info("Starting Terra 2 watcher")
			readiness.RegisterComponent(common.ReadinessTerra2Syncing)
			chainObsvReqC[vaa.ChainIDTerra2] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "terra2watch",
				common.WrapWithScissors(cosmwasm.NewWatcher(*terra2WS, *terra2LCD, *terra2Contract, lockC, chainObsvReqC[vaa.ChainIDTerra2], common.ReadinessTerra2Syncing, vaa.ChainIDTerra2).Run, "terra2watch")); err != nil {
				return err
			}
		}

		if shouldStart(xplaWS) {
			logger.Info("Starting XPLA watcher")
			readiness.RegisterComponent(common.ReadinessXplaSyncing)
			chainObsvReqC[vaa.ChainIDXpla] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "xplawatch",
				common.WrapWithScissors(cosmwasm.NewWatcher(*xplaWS, *xplaLCD, *xplaContract, lockC, chainObsvReqC[vaa.ChainIDXpla], common.ReadinessXplaSyncing, vaa.ChainIDXpla).Run, "xplawatch")); err != nil {
				return err
			}
		}

		if shouldStart(algorandIndexerRPC) {
			logger.Info("Starting Algorand watcher")
			readiness.RegisterComponent(common.ReadinessAlgorandSyncing)
			chainObsvReqC[vaa.ChainIDAlgorand] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "algorandwatch",
				common.WrapWithScissors(algorand.NewWatcher(*algorandIndexerRPC, *algorandIndexerToken, *algorandAlgodRPC, *algorandAlgodToken, *algorandAppID, lockC, chainObsvReqC[vaa.ChainIDAlgorand]).Run, "algorandwatch")); err != nil {
				return err
			}
		}
		if shouldStart(nearRPC) {
			logger.Info("Starting Near watcher")
			readiness.RegisterComponent(common.ReadinessNearSyncing)
			chainObsvReqC[vaa.ChainIDNear] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "nearwatch",
				common.WrapWithScissors(near.NewWatcher(*nearRPC, *nearContract, lockC, chainObsvReqC[vaa.ChainIDNear], !(*unsafeDevMode || *testnetMode)).Run, "nearwatch")); err != nil {
				return err
			}
		}

		// Start Wormchain watcher only if configured
		if shouldStart(wormchainWS) {
			logger.Info("Starting Wormchain watcher")
			readiness.RegisterComponent(common.ReadinessWormchainSyncing)
			chainObsvReqC[vaa.ChainIDWormchain] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "wormchainwatch",
				wormchain.NewWatcher(*wormchainWS, *wormchainLCD, lockC, setC, chainObsvReqC[vaa.ChainIDWormchain]).Run); err != nil {
				return err
			}
		}
		if shouldStart(aptosRPC) {
			logger.Info("Starting Aptos watcher")
			readiness.RegisterComponent(common.ReadinessAptosSyncing)
			chainObsvReqC[vaa.ChainIDAptos] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "aptoswatch",
				aptos.NewWatcher(*aptosRPC, *aptosAccount, *aptosHandle, lockC, chainObsvReqC[vaa.ChainIDAptos]).Run); err != nil {
				return err
			}
		}

		if shouldStart(suiRPC) {
			logger.Info("Starting Sui watcher")
			readiness.RegisterComponent(common.ReadinessSuiSyncing)
			chainObsvReqC[vaa.ChainIDSui] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "suiwatch",
				sui.NewWatcher(*suiRPC, *suiWS, *suiAccount, *suiPackage, *unsafeDevMode, lockC, chainObsvReqC[vaa.ChainIDSui]).Run); err != nil {
				return err
			}
		}

		var solanaFinalizedWatcher *solana.SolanaWatcher
		if shouldStart(solanaRPC) {
			logger.Info("Starting Solana watcher")
			readiness.RegisterComponent(common.ReadinessSolanaSyncing)
			chainObsvReqC[vaa.ChainIDSolana] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "solwatch-confirmed",
				common.WrapWithScissors(solana.NewSolanaWatcher(*solanaRPC, nil, solAddress, *solanaContract, lockC, nil, rpc.CommitmentConfirmed, common.ReadinessSolanaSyncing, vaa.ChainIDSolana).Run, "solwatch-confirmed")); err != nil {
				return err
			}
			solanaFinalizedWatcher = solana.NewSolanaWatcher(*solanaRPC, nil, solAddress, *solanaContract, lockC, chainObsvReqC[vaa.ChainIDSolana], rpc.CommitmentFinalized, common.ReadinessSolanaSyncing, vaa.ChainIDSolana)
			if err := supervisor.Run(ctx, "solwatch-finalized", common.WrapWithScissors(solanaFinalizedWatcher.Run, "solwatch-finalized")); err != nil {
				return err
			}
		}

		if shouldStart(pythnetRPC) {
			logger.Info("Starting Pythnet watcher")
			readiness.RegisterComponent(common.ReadinessPythNetSyncing)
			chainObsvReqC[vaa.ChainIDPythNet] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "pythwatch-confirmed",
				common.WrapWithScissors(solana.NewSolanaWatcher(*pythnetRPC, pythnetWS, pythnetAddress, *pythnetContract, lockC, nil, rpc.CommitmentConfirmed, common.ReadinessPythNetSyncing, vaa.ChainIDPythNet).Run, "pythwatch-confirmed")); err != nil {
				return err
			}
		}

		if shouldStart(injectiveWS) {
			logger.Info("Starting Injective watcher")
			readiness.RegisterComponent(common.ReadinessInjectiveSyncing)
			chainObsvReqC[vaa.ChainIDInjective] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
			if err := supervisor.Run(ctx, "injectivewatch",
				common.WrapWithScissors(cosmwasm.NewWatcher(*injectiveWS, *injectiveLCD, *injectiveContract, lockC, chainObsvReqC[vaa.ChainIDInjective], common.ReadinessInjectiveSyncing, vaa.ChainIDInjective).Run, "injectivewatch")); err != nil {
				return err
			}
		}

		if *testnetMode {
			if shouldStart(neonRPC) {
				if solanaFinalizedWatcher == nil {
					log.Fatalf("if neon is enabled then solana must also be enabled.")
				}
				logger.Info("Starting Neon watcher")
				readiness.RegisterComponent(common.ReadinessNeonSyncing)
				chainObsvReqC[vaa.ChainIDNeon] = make(chan *gossipv1.ObservationRequest, observationRequestBufferSize)
				neonWatcher := evm.NewEthWatcher(*neonRPC, neonContractAddr, "neon", common.ReadinessNeonSyncing, vaa.ChainIDNeon, lockC, nil, chainObsvReqC[vaa.ChainIDNeon], *unsafeDevMode)
				neonWatcher.SetL1Finalizer(solanaFinalizedWatcher)
				if err := supervisor.Run(ctx, "neonwatch", common.WrapWithScissors(neonWatcher.Run, "neonwatch")); err != nil {
					return err
				}
			}
		}
		go handleReobservationRequests(rootCtx, clock.New(), logger, obsvReqC, chainObsvReqC)

		if acct != nil {
			if err := acct.Start(ctx); err != nil {
				logger.Fatal("acct: failed to start accountant", zap.Error(err))
			}
		}

		if gov != nil {
			err := gov.Run(ctx)
			if err != nil {
				log.Fatal("failed to create chain governor", zap.Error(err))
			}
		}

		p := processor.NewProcessor(ctx,
			db,
			lockC,
			setC,
			sendC,
			obsvC,
			obsvReqSendC,
			injectC,
			signedInC,
			gk,
			gst,
			*unsafeDevMode,
			*devNumGuardians,
			*ethRPC,
			*wormchainLCD,
			attestationEvents,
			notifier,
			gov,
			acct,
			acctReadC,
		)
		if err := supervisor.Run(ctx, "processor", p.Run); err != nil {
			return err
		}

		if err := supervisor.Run(ctx, "admin", adminService); err != nil {
			return err
		}

		if shouldStart(publicGRPCSocketPath) {

			// local public grpc service socket
			publicrpcUnixService, publicrpcServer, err := publicrpcUnixServiceRunnable(logger, *publicGRPCSocketPath, db, gst, gov)
			if err != nil {
				logger.Fatal("failed to create publicrpc service socket", zap.Error(err))
			}

			if err := supervisor.Run(ctx, "publicrpcsocket", publicrpcUnixService); err != nil {
				return err
			}

			if shouldStart(publicRPC) {
				publicrpcService, err := publicrpcTcpServiceRunnable(logger, *publicRPC, db, gst, gov)
				if err != nil {
					log.Fatal("failed to create publicrpc tcp service", zap.Error(err))
				}
				if err := supervisor.Run(ctx, "publicrpc", publicrpcService); err != nil {
					return err
				}
			}

			if shouldStart(publicWeb) {
				publicwebService, err := publicwebServiceRunnable(logger, *publicWeb, *publicGRPCSocketPath, publicrpcServer,
					*tlsHostname, *tlsProdEnv, path.Join(*dataDir, "autocert"))
				if err != nil {
					log.Fatal("failed to create publicrpc web service", zap.Error(err))
				}

				if err := supervisor.Run(ctx, "publicweb", publicwebService); err != nil {
					return err
				}
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

func shouldStart(rpc *string) bool {
	return *rpc != "" && *rpc != "none"
}

func unsafeDevModeEvmContractAddress(contractAddr string) string {
	if contractAddr != "" {
		return contractAddr
	}

	return devnet.GanacheWormholeContractAddress.Hex()
}
