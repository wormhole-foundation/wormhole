package guardiand

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	_ "net/http/pprof" // #nosec G108 we are using a custom router (`router := mux.NewRouter()`) and thus not automatically expose pprof.
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/certusone/wormhole/node/pkg/watchers/ibc"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/certusone/wormhole/node/pkg/watchers/cosmwasm"

	"github.com/certusone/wormhole/node/pkg/watchers/algorand"
	"github.com/certusone/wormhole/node/pkg/watchers/aptos"
	"github.com/certusone/wormhole/node/pkg/watchers/evm"
	"github.com/certusone/wormhole/node/pkg/watchers/near"
	"github.com/certusone/wormhole/node/pkg/watchers/solana"
	"github.com/certusone/wormhole/node/pkg/watchers/sui"
	"github.com/certusone/wormhole/node/pkg/wormconn"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/telemetry"
	"github.com/certusone/wormhole/node/pkg/version"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap/zapcore"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/certusone/wormhole/node/pkg/node"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/reporter"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	cosmoscrypto "github.com/cosmos/cosmos-sdk/crypto/types"
	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	ipfslog "github.com/ipfs/go-log/v2"
	googleapi_option "google.golang.org/api/option"
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

	wormchainURL           *string
	wormchainKeyPath       *string
	wormchainKeyPassPhrase *string

	ibcWS       *string
	ibcLCD      *string
	ibcContract *string

	accountantContract     *string
	accountantWS           *string
	accountantCheckEnabled *bool

	aptosRPC     *string
	aptosAccount *string
	aptosHandle  *string

	suiRPC           *string
	suiWS            *string
	suiMoveEventType *string

	solanaRPC *string

	pythnetContract *string
	pythnetRPC      *string
	pythnetWS       *string

	arbitrumRPC      *string
	arbitrumContract *string

	optimismRPC      *string
	optimismContract *string

	baseRPC      *string
	baseContract *string

	sepoliaRPC      *string
	sepoliaContract *string

	logLevel                *string
	publicRpcLogDetailStr   *string
	publicRpcLogToTelemetry *bool

	unsafeDevMode *bool
	testnetMode   *bool
	nodeName      *string

	publicRPC *string
	publicWeb *string

	tlsHostname *string
	tlsProdEnv  *bool

	disableHeartbeatVerify *bool

	disableTelemetry *bool

	// Google cloud logging parameters
	telemetryKey                *string
	telemetryServiceAccountFile *string
	telemetryProject            *string

	// Loki cloud logging parameters
	telemetryLokiURL *string

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
	p2pPort = NodeCmd.Flags().Uint("port", p2p.DefaultPort, "P2P UDP listener port")
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

	wormchainURL = NodeCmd.Flags().String("wormchainURL", "", "wormhole-chain gRPC URL")
	wormchainKeyPath = NodeCmd.Flags().String("wormchainKeyPath", "", "path to wormhole-chain private key for signing transactions")
	wormchainKeyPassPhrase = NodeCmd.Flags().String("wormchainKeyPassPhrase", "", "pass phrase used to unarmor the wormchain key file")

	ibcWS = NodeCmd.Flags().String("ibcWS", "", "Websocket used to listen to the IBC receiver smart contract on wormchain")
	ibcLCD = NodeCmd.Flags().String("ibcLCD", "", "Path to LCD service root for http calls")
	ibcContract = NodeCmd.Flags().String("ibcContract", "", "Address of the IBC smart contract on wormchain")

	accountantWS = NodeCmd.Flags().String("accountantWS", "", "Websocket used to listen to the accountant smart contract on wormchain")
	accountantContract = NodeCmd.Flags().String("accountantContract", "", "Address of the accountant smart contract on wormchain")
	accountantCheckEnabled = NodeCmd.Flags().Bool("accountantCheckEnabled", false, "Should accountant be enforced on transfers")

	aptosRPC = NodeCmd.Flags().String("aptosRPC", "", "aptos RPC URL")
	aptosAccount = NodeCmd.Flags().String("aptosAccount", "", "aptos account")
	aptosHandle = NodeCmd.Flags().String("aptosHandle", "", "aptos handle")

	suiRPC = NodeCmd.Flags().String("suiRPC", "", "sui RPC URL")
	suiWS = NodeCmd.Flags().String("suiWS", "", "sui WS URL")
	suiMoveEventType = NodeCmd.Flags().String("suiMoveEventType", "", "sui move event type for publish_message")

	solanaRPC = NodeCmd.Flags().String("solanaRPC", "", "Solana RPC URL (required)")

	pythnetContract = NodeCmd.Flags().String("pythnetContract", "", "Address of the PythNet program (required)")
	pythnetRPC = NodeCmd.Flags().String("pythnetRPC", "", "PythNet RPC URL (required)")
	pythnetWS = NodeCmd.Flags().String("pythnetWS", "", "PythNet WS URL")

	arbitrumRPC = NodeCmd.Flags().String("arbitrumRPC", "", "Arbitrum RPC URL")
	arbitrumContract = NodeCmd.Flags().String("arbitrumContract", "", "Arbitrum contract address")

	sepoliaRPC = NodeCmd.Flags().String("sepoliaRPC", "", "Sepolia RPC URL")
	sepoliaContract = NodeCmd.Flags().String("sepoliaContract", "", "Sepolia contract address")

	optimismRPC = NodeCmd.Flags().String("optimismRPC", "", "Optimism RPC URL")
	optimismContract = NodeCmd.Flags().String("optimismContract", "", "Optimism contract address")

	baseRPC = NodeCmd.Flags().String("baseRPC", "", "Base RPC URL")
	baseContract = NodeCmd.Flags().String("baseContract", "", "Base contract address")

	logLevel = NodeCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")
	publicRpcLogDetailStr = NodeCmd.Flags().String("publicRpcLogDetail", "full", "The detail with which public RPC requests shall be logged (none=no logging, minimal=only log gRPC methods, full=log gRPC method, payload (up to 200 bytes) and user agent (up to 200 bytes))")
	publicRpcLogToTelemetry = NodeCmd.Flags().Bool("logPublicRpcToTelemetry", true, "whether or not to include publicRpc request logs in telemetry")

	unsafeDevMode = NodeCmd.Flags().Bool("unsafeDevMode", false, "Launch node in unsafe, deterministic devnet mode")
	testnetMode = NodeCmd.Flags().Bool("testnetMode", false, "Launch node in testnet mode (enables testnet-only features)")
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
	telemetryServiceAccountFile = NodeCmd.Flags().String("telemetryServiceAccountFile", "",
		"Google Cloud credentials json for accessing Cloud Logging")
	telemetryProject = NodeCmd.Flags().String("telemetryProject", defaultTelemetryProject,
		"Google Cloud Project to use for Telemetry logging")

	telemetryLokiURL = NodeCmd.Flags().String("telemetryLokiURL", "", "Loki cloud logging URL")

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

func runNode(cmd *cobra.Command, args []string) {
	if Build == "dev" && !*unsafeDevMode {
		fmt.Println("This is a development build. --unsafeDevMode must be enabled.")
		os.Exit(1)
	}

	if *unsafeDevMode {
		fmt.Print(devwarning)
	}

	if *testnetMode || *unsafeDevMode {
		fmt.Println("Not locking in memory.")
	} else {
		common.LockMemory()
	}

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
		*baseContract = unsafeDevModeEvmContractAddress(*baseContract)
		*sepoliaContract = unsafeDevModeEvmContractAddress(*sepoliaContract)
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
		if *suiMoveEventType == "" {
			logger.Fatal("If --suiRPC is specified, then --suiMoveEventType must be specified")
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
		if *baseRPC == "" {
			logger.Fatal("Please specify --baseRPC")
		}
		if *baseContract == "" {
			logger.Fatal("Please specify --baseContract")
		}
		if *sepoliaRPC == "" {
			logger.Fatal("Please specify --sepoliaRPC")
		}
		if *sepoliaContract == "" {
			logger.Fatal("Please specify --sepoliaContract")
		}
	} else {
		if *neonRPC != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --neonRPC")
		}
		if *neonContract != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --neonContract")
		}
		if *baseRPC != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --baseRPC")
		}
		if *baseContract != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --baseContract")
		}
		if *sepoliaRPC != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --sepoliaRPC")
		}
		if *sepoliaContract != "" && !*unsafeDevMode {
			logger.Fatal("Please do not specify --sepoliaContract")
		}
	}

	var publicRpcLogDetail common.GrpcLogDetail
	switch *publicRpcLogDetailStr {
	case "none":
		publicRpcLogDetail = common.GrpcLogDetailNone
	case "minimal":
		publicRpcLogDetail = common.GrpcLogDetailMinimal
	case "full":
		publicRpcLogDetail = common.GrpcLogDetailFull
	default:
		logger.Fatal("--publicRpcLogDetail should be one of (none, minimal, full)")
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

	if *telemetryKey != "" && *telemetryServiceAccountFile != "" {
		logger.Fatal("Please do not specify both --telemetryKey and --telemetryServiceAccountFile")
	}

	// Determine execution mode
	// TODO: refactor usage of these variables elsewhere. *unsafeDevMode and *testnetMode should not be accessed directly.
	var env common.Environment
	if *unsafeDevMode {
		env = common.UnsafeDevNet
	} else if *testnetMode {
		env = common.TestNet
	} else {
		env = common.MainNet
	}

	if *unsafeDevMode && *testnetMode {
		logger.Fatal("Cannot be in unsafeDevMode and testnetMode at the same time.")
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
	db := db.OpenDb(logger, dataDir)
	defer db.Close()

	// Guardian key
	gk, err := loadGuardianKey(*guardianKeyPath)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}

	logger.Info("Loaded guardian key", zap.String(
		"address", ethcrypto.PubkeyToAddress(gk.PublicKey).String()))

	// Load p2p private key
	var p2pKey libp2p_crypto.PrivKey
	if *unsafeDevMode {
		idx, err := devnet.GetDevnetIndex()
		if err != nil {
			logger.Fatal("Failed to parse hostname - are we running in devnet?")
		}
		p2pKey = devnet.DeterministicP2PPrivKeyByIndex(int64(idx))

		if idx != 0 {
			// try to connect to guardian-0
			for {
				_, err := net.LookupIP("guardian-0.guardian")
				if err == nil {
					break
				}
				logger.Info("Error resolving guardian-0.guardian. Trying again...")
				time.Sleep(time.Second)
			}
			// TODO this is a hack. If this is not the bootstrap Guardian, we wait 5s such that the bootstrap Guardian has enough time to start.
			logger.Info("This is not a bootstrap Guardian. Waiting another 10 seconds so the bootstrap guardian to come online.")
			time.Sleep(time.Second * 10)
		}
	} else {
		p2pKey, err = common.GetOrCreateNodeKey(logger, *nodeKeyPath)
		if err != nil {
			logger.Fatal("Failed to load node key", zap.Error(err))
		}
	}

	rpcMap := make(map[string]string)
	rpcMap["acalaRPC"] = *acalaRPC
	rpcMap["algorandIndexerRPC"] = *algorandIndexerRPC
	rpcMap["algorandAlgodRPC"] = *algorandAlgodRPC
	rpcMap["aptosRPC"] = *aptosRPC
	rpcMap["arbitrumRPC"] = *arbitrumRPC
	rpcMap["auroraRPC"] = *auroraRPC
	rpcMap["avalancheRPC"] = *avalancheRPC
	rpcMap["baseRPC"] = *baseRPC
	rpcMap["bscRPC"] = *bscRPC
	rpcMap["celoRPC"] = *celoRPC
	rpcMap["ethRPC"] = *ethRPC
	rpcMap["fantomRPC"] = *fantomRPC
	rpcMap["ibcLCD"] = *ibcLCD
	rpcMap["ibcWS"] = *ibcWS
	rpcMap["karuraRPC"] = *karuraRPC
	rpcMap["klaytnRPC"] = *klaytnRPC
	rpcMap["moonbeamRPC"] = *moonbeamRPC
	rpcMap["nearRPC"] = *nearRPC
	rpcMap["neonRPC"] = *neonRPC
	rpcMap["oasisRPC"] = *oasisRPC
	rpcMap["optimismRPC"] = *optimismRPC
	rpcMap["polygonRPC"] = *polygonRPC
	rpcMap["pythnetRPC"] = *pythnetRPC
	rpcMap["pythnetWS"] = *pythnetWS
	rpcMap["sei"] = "IBC"
	if env == common.TestNet {
		rpcMap["sepoliaRPC"] = *sepoliaRPC
	}
	rpcMap["solanaRPC"] = *solanaRPC
	rpcMap["suiRPC"] = *suiRPC
	rpcMap["terraWS"] = *terraWS
	rpcMap["terraLCD"] = *terraLCD
	rpcMap["terra2WS"] = *terra2WS
	rpcMap["terra2LCD"] = *terra2LCD
	rpcMap["xplaWS"] = *xplaWS
	rpcMap["xplaLCD"] = *xplaLCD

	// Node's main lifecycle context.
	rootCtx, rootCtxCancel = context.WithCancel(context.Background())
	defer rootCtxCancel()

	// Handle SIGTERM
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	go func() {
		<-sigterm
		logger.Info("Received sigterm. exiting.")
		rootCtxCancel()
	}()

	usingLoki := *telemetryLokiURL != ""
	usingGCP := *telemetryKey != "" || *telemetryServiceAccountFile != ""

	var hasTelemetryCredential bool = usingGCP || usingLoki

	// Telemetry is enabled by default in mainnet/testnet. In devnet it is disabled by default
	if !*disableTelemetry && (!*unsafeDevMode || *unsafeDevMode && hasTelemetryCredential) {
		if !hasTelemetryCredential {
			logger.Fatal("Please either specify --telemetryKey, --telemetryServiceAccountFile or --telemetryLokiURL or set --disableTelemetry=false")
		}

		if usingLoki && usingGCP {
			logger.Fatal("May only enable one telemetry logger at a time, either specify --telemetryLokiURL or --telemetryKey/--telemetryServiceAccountFile")
		}

		// Get libp2p peer ID from private key
		pk := p2pKey.GetPublic()
		peerID, err := peer.IDFromPublicKey(pk)
		if err != nil {
			logger.Fatal("Failed to get peer ID from private key", zap.Error(err))
		}

		labels := map[string]string{
			"node_name":     *nodeName,
			"node_key":      peerID.Pretty(),
			"guardian_addr": ethcrypto.PubkeyToAddress(gk.PublicKey).String(),
			"network":       *p2pNetworkID,
			"version":       version.Version(),
		}

		skipPrivateLogs := !*publicRpcLogToTelemetry

		var tm *telemetry.Telemetry
		if usingLoki {
			logger.Info("Using Loki telemetry logger",
				zap.String("publicRpcLogDetail", *publicRpcLogDetailStr),
				zap.Bool("logPublicRpcToTelemetry", *publicRpcLogToTelemetry))

			tm, err = telemetry.NewLokiCloudLogger(context.Background(), logger, *telemetryLokiURL, "wormhole", true, labels)
			if err != nil {
				logger.Fatal("Failed to initialize telemetry", zap.Error(err))
			}
		} else {
			logger.Info("Using Google Cloud telemetry logger",
				zap.String("publicRpcLogDetail", *publicRpcLogDetailStr),
				zap.Bool("logPublicRpcToTelemetry", *publicRpcLogToTelemetry))

			var options []googleapi_option.ClientOption

			if *telemetryKey != "" {
				creds, err := decryptTelemetryServiceAccount()
				if err != nil {
					logger.Fatal("Failed to decrypt telemetry service account", zap.Error(err))
				}

				options = append(options, googleapi_option.WithCredentialsJSON(creds))
			}

			if *telemetryServiceAccountFile != "" {
				options = append(options, googleapi_option.WithCredentialsFile(*telemetryServiceAccountFile))
			}

			tm, err = telemetry.NewGoogleCloudLogger(context.Background(), *telemetryProject, skipPrivateLogs, labels, options...)
			if err != nil {
				logger.Fatal("Failed to initialize telemetry", zap.Error(err))
			}
		}

		defer tm.Close()
		logger = tm.WrapLogger(logger) // Wrap logger with telemetry logger
	}

	// log golang version
	logger.Info("golang version", zap.String("golang_version", runtime.Version()))

	// Redirect ipfs logs to plain zap
	ipfslog.SetPrimaryCore(logger.Core())

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

		logger.Debug("loading key file", zap.String("key path", wormchainKeyPathName))
		wormchainKey, err = wormconn.LoadWormchainPrivKey(wormchainKeyPathName, *wormchainKeyPassPhrase)
		if err != nil {
			logger.Fatal("failed to load wormchain private key", zap.Error(err))
		}

		// Connect to wormchain.
		logger.Info("Connecting to wormchain", zap.String("wormchainURL", *wormchainURL), zap.String("wormchainKeyPath", wormchainKeyPathName))
		wormchainConn, err = wormconn.NewConn(rootCtx, *wormchainURL, wormchainKey)
		if err != nil {
			logger.Fatal("failed to connect to wormchain", zap.Error(err))
		}
	}

	var watcherConfigs = []watchers.WatcherConfig{}

	if shouldStart(ethRPC) {
		wc := &evm.WatcherConfig{
			NetworkID:              "eth",
			ChainID:                vaa.ChainIDEthereum,
			Rpc:                    *ethRPC,
			Contract:               *ethContract,
			GuardianSetUpdateChain: true,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(bscRPC) {
		wc := &evm.WatcherConfig{
			NetworkID:            "bsc",
			ChainID:              vaa.ChainIDBSC,
			Rpc:                  *bscRPC,
			Contract:             *bscContract,
			WaitForConfirmations: true,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(polygonRPC) {
		// Checkpointing is required in mainnet, so we don't need to wait for confirmations.
		waitForConfirmations := *unsafeDevMode || *testnetMode
		if !waitForConfirmations && *polygonRootChainRpc == "" {
			log.Fatal("Polygon checkpointing is required in mainnet")
		}
		wc := &evm.WatcherConfig{
			NetworkID:            "polygon",
			ChainID:              vaa.ChainIDPolygon,
			Rpc:                  *polygonRPC,
			Contract:             *polygonContract,
			WaitForConfirmations: waitForConfirmations,
			RootChainRpc:         *polygonRootChainRpc,
			RootChainContract:    *polygonRootChainContractAddress,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(avalancheRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "avalanche",
			ChainID:   vaa.ChainIDAvalanche,
			Rpc:       *avalancheRPC,
			Contract:  *avalancheContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(oasisRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "oasis",
			ChainID:   vaa.ChainIDOasis,
			Rpc:       *oasisRPC,
			Contract:  *oasisContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(auroraRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "aurora",
			ChainID:   vaa.ChainIDAurora,
			Rpc:       *auroraRPC,
			Contract:  *auroraContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(fantomRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "fantom",
			ChainID:   vaa.ChainIDFantom,
			Rpc:       *fantomRPC,
			Contract:  *fantomContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(karuraRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "karura",
			ChainID:   vaa.ChainIDKarura,
			Rpc:       *karuraRPC,
			Contract:  *karuraContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(acalaRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "acala",
			ChainID:   vaa.ChainIDAcala,
			Rpc:       *acalaRPC,
			Contract:  *acalaContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(klaytnRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "klaytn",
			ChainID:   vaa.ChainIDKlaytn,
			Rpc:       *klaytnRPC,
			Contract:  *klaytnContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(celoRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "celo",
			ChainID:   vaa.ChainIDCelo,
			Rpc:       *celoRPC,
			Contract:  *celoContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(moonbeamRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "moonbeam",
			ChainID:   vaa.ChainIDMoonbeam,
			Rpc:       *moonbeamRPC,
			Contract:  *moonbeamContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(arbitrumRPC) {
		wc := &evm.WatcherConfig{
			NetworkID:           "arbitrum",
			ChainID:             vaa.ChainIDArbitrum,
			Rpc:                 *arbitrumRPC,
			Contract:            *arbitrumContract,
			L1FinalizerRequired: "eth",
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(optimismRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "optimism",
			ChainID:   vaa.ChainIDOptimism,
			Rpc:       *optimismRPC,
			Contract:  *optimismContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(terraWS) {
		wc := &cosmwasm.WatcherConfig{
			NetworkID: "terra",
			ChainID:   vaa.ChainIDTerra,
			Websocket: *terraWS,
			Lcd:       *terraLCD,
			Contract:  *terraContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(terra2WS) {
		wc := &cosmwasm.WatcherConfig{
			NetworkID: "terra2",
			ChainID:   vaa.ChainIDTerra2,
			Websocket: *terra2WS,
			Lcd:       *terra2LCD,
			Contract:  *terra2Contract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(xplaWS) {
		wc := &cosmwasm.WatcherConfig{
			NetworkID: "xpla",
			ChainID:   vaa.ChainIDXpla,
			Websocket: *xplaWS,
			Lcd:       *xplaLCD,
			Contract:  *xplaContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(injectiveWS) {
		wc := &cosmwasm.WatcherConfig{
			NetworkID: "injective",
			ChainID:   vaa.ChainIDInjective,
			Websocket: *injectiveWS,
			Lcd:       *injectiveLCD,
			Contract:  *injectiveContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(algorandIndexerRPC) {
		wc := &algorand.WatcherConfig{
			NetworkID:    "algorand",
			ChainID:      vaa.ChainIDAlgorand,
			IndexerRPC:   *algorandIndexerRPC,
			IndexerToken: *algorandIndexerToken,
			AlgodRPC:     *algorandAlgodRPC,
			AlgodToken:   *algorandAlgodToken,
			AppID:        *algorandAppID,
		}
		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(nearRPC) {
		wc := &near.WatcherConfig{
			NetworkID: "near",
			ChainID:   vaa.ChainIDNear,
			Rpc:       *nearRPC,
			Contract:  *nearContract,
		}
		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(aptosRPC) {
		wc := &aptos.WatcherConfig{
			NetworkID: "aptos",
			ChainID:   vaa.ChainIDAptos,
			Rpc:       *aptosRPC,
			Account:   *aptosAccount,
			Handle:    *aptosHandle,
		}
		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(suiRPC) {
		wc := &sui.WatcherConfig{
			NetworkID:        "sui",
			ChainID:          vaa.ChainIDSui,
			Rpc:              *suiRPC,
			Websocket:        *suiWS,
			SuiMoveEventType: *suiMoveEventType,
		}
		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(solanaRPC) {
		// confirmed watcher
		wc := &solana.WatcherConfig{
			NetworkID:     "solana-confirmed",
			ChainID:       vaa.ChainIDSolana,
			Rpc:           *solanaRPC,
			Websocket:     "",
			Contract:      *solanaContract,
			ReceiveObsReq: false,
			Commitment:    rpc.CommitmentConfirmed,
		}

		watcherConfigs = append(watcherConfigs, wc)

		// finalized watcher
		wc = &solana.WatcherConfig{
			NetworkID:     "solana-finalized",
			ChainID:       vaa.ChainIDSolana,
			Rpc:           *solanaRPC,
			Websocket:     "",
			Contract:      *solanaContract,
			ReceiveObsReq: true,
			Commitment:    rpc.CommitmentFinalized,
		}
		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(pythnetRPC) {
		wc := &solana.WatcherConfig{
			NetworkID:     "pythnet",
			ChainID:       vaa.ChainIDPythNet,
			Rpc:           *pythnetRPC,
			Websocket:     *pythnetWS,
			Contract:      *pythnetContract,
			ReceiveObsReq: false,
			Commitment:    rpc.CommitmentConfirmed,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if *testnetMode {
		if shouldStart(neonRPC) {
			if !shouldStart(solanaRPC) {
				log.Fatalf("If neon is enabled then solana must also be enabled.")
			}
			wc := &evm.WatcherConfig{
				NetworkID:           "neon",
				ChainID:             vaa.ChainIDNeon,
				Rpc:                 *neonRPC,
				Contract:            *neonContract,
				L1FinalizerRequired: "solana-finalized",
			}

			watcherConfigs = append(watcherConfigs, wc)
		}

		if shouldStart(baseRPC) {
			wc := &evm.WatcherConfig{
				NetworkID: "base",
				ChainID:   vaa.ChainIDBase,
				Rpc:       *baseRPC,
				Contract:  *baseContract,
			}

			watcherConfigs = append(watcherConfigs, wc)
		}

		if shouldStart(sepoliaRPC) {
			wc := &evm.WatcherConfig{
				NetworkID: "sepolia",
				ChainID:   vaa.ChainIDSepolia,
				Rpc:       *sepoliaRPC,
				Contract:  *sepoliaContract,
			}

			watcherConfigs = append(watcherConfigs, wc)
		}
	}

	var ibcWatcherConfig *node.IbcWatcherConfig = nil
	if shouldStart(ibcWS) {
		ibcWatcherConfig = &node.IbcWatcherConfig{
			Websocket: *ibcWS,
			Lcd:       *ibcLCD,
			Contract:  *ibcContract,
		}
	}

	guardianNode := node.NewGuardianNode(
		env,
		db,
		gk,
		wormchainConn,
	)

	guardianOptions := []*node.GuardianOption{
		node.GuardianOptionWatchers(watcherConfigs, ibcWatcherConfig),
		node.GuardianOptionAccountant(*accountantContract, *accountantWS, *accountantCheckEnabled),
		node.GuardianOptionGovernor(*chainGovernorEnabled),
		node.GuardianOptionAdminService(*adminSocketPath, ethRPC, ethContract, rpcMap),
		node.GuardianOptionP2P(p2pKey, *p2pNetworkID, *p2pBootstrap, *nodeName, *disableHeartbeatVerify, *p2pPort, ibc.GetFeatures),
		node.GuardianOptionStatusServer(*statusAddr),
	}

	if shouldStart(publicGRPCSocketPath) {
		guardianOptions = append(guardianOptions, node.GuardianOptionPublicRpcSocket(*publicGRPCSocketPath, publicRpcLogDetail))

		if shouldStart(publicRPC) {
			guardianOptions = append(guardianOptions, node.GuardianOptionPublicrpcTcpService(*publicRPC, publicRpcLogDetail))
		}

		if shouldStart(publicWeb) {
			guardianOptions = append(guardianOptions,
				node.GuardianOptionPublicWeb(*publicWeb, *publicGRPCSocketPath, *tlsHostname, *tlsProdEnv, path.Join(*dataDir, "autocert")),
			)
		}
	}

	if *bigTablePersistenceEnabled {
		bigTableConnectionConfig := &reporter.BigTableConnectionConfig{
			GcpProjectID:    *bigTableGCPProject,
			GcpInstanceName: *bigTableInstanceName,
			TableName:       *bigTableTableName,
			TopicName:       *bigTableTopicName,
			GcpKeyFilePath:  *bigTableKeyPath,
		}

		guardianOptions = append(guardianOptions, node.GuardianOptionBigTablePersistence(bigTableConnectionConfig))
	}

	// Run supervisor with Guardian Node as root.
	supervisor.New(rootCtx, logger, guardianNode.Run(rootCtxCancel, guardianOptions...),
		// It's safer to crash and restart the process in case we encounter a panic,
		// rather than attempting to reschedule the runnable.
		supervisor.WithPropagatePanic)

	<-rootCtx.Done()
	logger.Info("root context cancelled, exiting...")
}

func decryptTelemetryServiceAccount() ([]byte, error) {
	// Decrypt service account credentials
	key, err := base64.StdEncoding.DecodeString(*telemetryKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode: %w", err)
	}

	ciphertext, err := base64.StdEncoding.DecodeString(defaultTelemetryServiceAccountEnc)
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
