package guardiand

import (
	"context"
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
	"github.com/certusone/wormhole/node/pkg/supervisor"
	promremotew "github.com/certusone/wormhole/node/pkg/telemetry/prom_remote_write"
	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
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

	polygonRPC      *string
	polygonContract *string

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

	gatewayWS       *string
	gatewayLCD      *string
	gatewayContract *string

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

	ibcWS             *string
	ibcLCD            *string
	ibcBlockHeightURL *string
	ibcContract       *string

	accountantContract      *string
	accountantWS            *string
	accountantCheckEnabled  *bool
	accountantKeyPath       *string
	accountantKeyPassPhrase *string

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

	scrollRPC      *string
	scrollContract *string

	mantleRPC      *string
	mantleContract *string

	sepoliaRPC      *string
	sepoliaContract *string

	holeskyRPC      *string
	holeskyContract *string

	arbitrumSepoliaRPC      *string
	arbitrumSepoliaContract *string

	baseSepoliaRPC      *string
	baseSepoliaContract *string

	optimismSepoliaRPC      *string
	optimismSepoliaContract *string

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

	// Loki cloud logging parameters
	telemetryLokiURL *string

	// Prometheus remote write URL
	promRemoteURL *string

	chainGovernorEnabled *bool

	ccqEnabled           *bool
	ccqAllowedRequesters *string
	ccqP2pPort           *uint
	ccqP2pBootstrap      *string
	ccqAllowedPeers      *string

	gatewayRelayerContract      *string
	gatewayRelayerKeyPath       *string
	gatewayRelayerKeyPassPhrase *string
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

	ethRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "ethRPC", "Ethereum RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	ethContract = NodeCmd.Flags().String("ethContract", "", "Ethereum contract address")

	bscRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "bscRPC", "Binance Smart Chain RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	bscContract = NodeCmd.Flags().String("bscContract", "", "Binance Smart Chain contract address")

	polygonRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "polygonRPC", "Polygon RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	polygonContract = NodeCmd.Flags().String("polygonContract", "", "Polygon contract address")

	avalancheRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "avalancheRPC", "Avalanche RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	avalancheContract = NodeCmd.Flags().String("avalancheContract", "", "Avalanche contract address")

	oasisRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "oasisRPC", "Oasis RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	oasisContract = NodeCmd.Flags().String("oasisContract", "", "Oasis contract address")

	auroraRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "auroraRPC", "Aurora Websocket RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	auroraContract = NodeCmd.Flags().String("auroraContract", "", "Aurora contract address")

	fantomRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "fantomRPC", "Fantom Websocket RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	fantomContract = NodeCmd.Flags().String("fantomContract", "", "Fantom contract address")

	karuraRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "karuraRPC", "Karura RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	karuraContract = NodeCmd.Flags().String("karuraContract", "", "Karura contract address")

	acalaRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "acalaRPC", "Acala RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	acalaContract = NodeCmd.Flags().String("acalaContract", "", "Acala contract address")

	klaytnRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "klaytnRPC", "Klaytn RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	klaytnContract = NodeCmd.Flags().String("klaytnContract", "", "Klaytn contract address")

	celoRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "celoRPC", "Celo RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	celoContract = NodeCmd.Flags().String("celoContract", "", "Celo contract address")

	moonbeamRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "moonbeamRPC", "Moonbeam RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	moonbeamContract = NodeCmd.Flags().String("moonbeamContract", "", "Moonbeam contract address")

	neonRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "neonRPC", "Neon RPC URL", "http://eth-devnet:8545", []string{"http", "https"})
	neonContract = NodeCmd.Flags().String("neonContract", "", "Neon contract address")

	terraWS = node.RegisterFlagWithValidationOrFail(NodeCmd, "terraWS", "Path to terrad root for websocket connection", "ws://terra-terrad:26657/websocket", []string{"ws", "wss"})
	terraLCD = node.RegisterFlagWithValidationOrFail(NodeCmd, "terraLCD", "Path to LCD service root for http calls", "http://terra-terrad:1317", []string{"http", "https"})
	terraContract = NodeCmd.Flags().String("terraContract", "", "Wormhole contract address on Terra blockchain")

	terra2WS = node.RegisterFlagWithValidationOrFail(NodeCmd, "terra2WS", "Path to terrad root for websocket connection", "ws://terra2-terrad:26657/websocket", []string{"ws", "wss"})
	terra2LCD = node.RegisterFlagWithValidationOrFail(NodeCmd, "terra2LCD", "Path to LCD service root for http calls", "http://terra2-terrad:1317", []string{"http", "https"})
	terra2Contract = NodeCmd.Flags().String("terra2Contract", "", "Wormhole contract address on Terra 2 blockchain")

	injectiveWS = node.RegisterFlagWithValidationOrFail(NodeCmd, "injectiveWS", "Path to root for Injective websocket connection", "ws://injective:26657/websocket", []string{"ws", "wss"})
	injectiveLCD = node.RegisterFlagWithValidationOrFail(NodeCmd, "injectiveLCD", "Path to LCD service root for Injective http calls", "http://injective:1317", []string{"http", "https"})
	injectiveContract = NodeCmd.Flags().String("injectiveContract", "", "Wormhole contract address on Injective blockchain")

	xplaWS = node.RegisterFlagWithValidationOrFail(NodeCmd, "xplaWS", "Path to root for XPLA websocket connection", "ws://xpla:26657/websocket", []string{"ws", "wss"})
	xplaLCD = node.RegisterFlagWithValidationOrFail(NodeCmd, "xplaLCD", "Path to LCD service root for XPLA http calls", "http://xpla:1317", []string{"http", "https"})
	xplaContract = NodeCmd.Flags().String("xplaContract", "", "Wormhole contract address on XPLA blockchain")

	gatewayWS = node.RegisterFlagWithValidationOrFail(NodeCmd, "gatewayWS", "Path to root for Gateway watcher websocket connection", "ws://wormchain:26657/websocket", []string{"ws", "wss"})
	gatewayLCD = node.RegisterFlagWithValidationOrFail(NodeCmd, "gatewayLCD", "Path to LCD service root for Gateway watcher http calls", "http://wormchain:1317", []string{"http", "https"})
	gatewayContract = NodeCmd.Flags().String("gatewayContract", "", "Wormhole contract address on Gateway blockchain")

	algorandIndexerRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "algorandIndexerRPC", "Algorand Indexer RPC URL", "http://algorand:8980", []string{"http", "https"})
	algorandIndexerToken = NodeCmd.Flags().String("algorandIndexerToken", "", "Algorand Indexer access token")
	algorandAlgodRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "algorandAlgodRPC", "Algorand Algod RPC URL", "http://algorand:4001", []string{"http", "https"})
	algorandAlgodToken = NodeCmd.Flags().String("algorandAlgodToken", "", "Algorand Algod access token")
	algorandAppID = NodeCmd.Flags().Uint64("algorandAppID", 0, "Algorand app id")

	nearRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "nearRPC", "Near RPC URL", "http://near:3030", []string{"http", "https"})
	nearContract = NodeCmd.Flags().String("nearContract", "", "Near contract")

	wormchainURL = node.RegisterFlagWithValidationOrFail(NodeCmd, "wormchainURL", "Wormhole-chain gRPC URL", "wormchain:9090", []string{""})
	// TODO: These are deprecated. Get rid of them once the guardians have had a chance to migrate off of them.
	wormchainKeyPath = NodeCmd.Flags().String("wormchainKeyPath", "", "path to wormhole-chain private key for signing transactions")
	wormchainKeyPassPhrase = NodeCmd.Flags().String("wormchainKeyPassPhrase", "", "pass phrase used to unarmor the wormchain key file")

	ibcWS = node.RegisterFlagWithValidationOrFail(NodeCmd, "ibcWS", "Websocket used to listen to the IBC receiver smart contract on wormchain", "ws://wormchain:26657/websocket", []string{"ws", "wss"})
	ibcLCD = node.RegisterFlagWithValidationOrFail(NodeCmd, "ibcLCD", "Path to LCD service root for http calls", "http://wormchain:1317", []string{"http", "https"})
	ibcBlockHeightURL = node.RegisterFlagWithValidationOrFail(NodeCmd, "ibcBlockHeightURL", "Optional URL to query for the block height (generated from ibcWS if not specified)", "http://wormchain:1317", []string{"http", "https"})
	ibcContract = NodeCmd.Flags().String("ibcContract", "", "Address of the IBC smart contract on wormchain")

	accountantWS = node.RegisterFlagWithValidationOrFail(NodeCmd, "accountantWS", "Websocket used to listen to the accountant smart contract on wormchain", "http://wormchain:26657", []string{"http", "https"})
	accountantContract = NodeCmd.Flags().String("accountantContract", "", "Address of the accountant smart contract on wormchain")
	accountantKeyPath = NodeCmd.Flags().String("accountantKeyPath", "", "path to accountant private key for signing transactions")
	accountantKeyPassPhrase = NodeCmd.Flags().String("accountantKeyPassPhrase", "", "pass phrase used to unarmor the accountant key file")
	accountantCheckEnabled = NodeCmd.Flags().Bool("accountantCheckEnabled", false, "Should accountant be enforced on transfers")

	aptosRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "aptosRPC", "Aptos RPC URL", "http://aptos:8080", []string{"http", "https"})
	aptosAccount = NodeCmd.Flags().String("aptosAccount", "", "aptos account")
	aptosHandle = NodeCmd.Flags().String("aptosHandle", "", "aptos handle")

	suiRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "suiRPC", "Sui RPC URL", "http://sui:9000", []string{"http", "https"})
	suiWS = node.RegisterFlagWithValidationOrFail(NodeCmd, "suiWS", "Sui WS URL", "ws://sui:9000", []string{"ws", "wss"})
	suiMoveEventType = NodeCmd.Flags().String("suiMoveEventType", "", "Sui move event type for publish_message")

	solanaRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "solanaRPC", "Solana RPC URL (required)", "http://solana-devnet:8899", []string{"http", "https"})

	pythnetContract = NodeCmd.Flags().String("pythnetContract", "", "Address of the PythNet program (required)")
	pythnetRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "pythnetRPC", "PythNet RPC URL (required)", "http://pythnet.rpcpool.com", []string{"http", "https"})
	pythnetWS = node.RegisterFlagWithValidationOrFail(NodeCmd, "pythnetWS", "PythNet WS URL", "wss://pythnet.rpcpool.com", []string{"ws", "wss"})

	arbitrumRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "arbitrumRPC", "Arbitrum RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	arbitrumContract = NodeCmd.Flags().String("arbitrumContract", "", "Arbitrum contract address")

	sepoliaRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "sepoliaRPC", "Sepolia RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	sepoliaContract = NodeCmd.Flags().String("sepoliaContract", "", "Sepolia contract address")

	holeskyRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "holeskyRPC", "Holesky RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	holeskyContract = NodeCmd.Flags().String("holeskyContract", "", "Holesky contract address")

	optimismRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "optimismRPC", "Optimism RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	optimismContract = NodeCmd.Flags().String("optimismContract", "", "Optimism contract address")

	scrollRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "scrollRPC", "Scroll RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	scrollContract = NodeCmd.Flags().String("scrollContract", "", "Scroll contract address")

	mantleRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "mantleRPC", "Mantle RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	mantleContract = NodeCmd.Flags().String("mantleContract", "", "Mantle contract address")

	baseRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "baseRPC", "Base RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	baseContract = NodeCmd.Flags().String("baseContract", "", "Base contract address")

	arbitrumSepoliaRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "arbitrumSepoliaRPC", "Arbitrum on Sepolia RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	arbitrumSepoliaContract = NodeCmd.Flags().String("arbitrumSepoliaContract", "", "Arbitrum on Sepolia contract address")

	baseSepoliaRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "baseSepoliaRPC", "Base on Sepolia RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	baseSepoliaContract = NodeCmd.Flags().String("baseSepoliaContract", "", "Base on Sepolia contract address")

	optimismSepoliaRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "optimismSepoliaRPC", "Optimism on Sepolia RPC URL", "ws://eth-devnet:8545", []string{"ws", "wss"})
	optimismSepoliaContract = NodeCmd.Flags().String("optimismSepoliaContract", "", "Optimism on Sepolia contract address")

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

	telemetryLokiURL = NodeCmd.Flags().String("telemetryLokiURL", "", "Loki cloud logging URL")

	promRemoteURL = NodeCmd.Flags().String("promRemoteURL", "", "Prometheus remote write URL (Grafana)")

	chainGovernorEnabled = NodeCmd.Flags().Bool("chainGovernorEnabled", false, "Run the chain governor")

	ccqEnabled = NodeCmd.Flags().Bool("ccqEnabled", false, "Enable cross chain query support")
	ccqAllowedRequesters = NodeCmd.Flags().String("ccqAllowedRequesters", "", "Comma separated list of signers allowed to submit cross chain queries")
	ccqP2pPort = NodeCmd.Flags().Uint("ccqP2pPort", 8996, "CCQ P2P UDP listener port")
	ccqP2pBootstrap = NodeCmd.Flags().String("ccqP2pBootstrap", "", "CCQ P2P bootstrap peers (comma-separated)")
	ccqAllowedPeers = NodeCmd.Flags().String("ccqAllowedPeers", "", "CCQ allowed P2P peers (comma-separated)")

	gatewayRelayerContract = NodeCmd.Flags().String("gatewayRelayerContract", "", "Address of the smart contract on wormchain to receive relayed VAAs")
	gatewayRelayerKeyPath = NodeCmd.Flags().String("gatewayRelayerKeyPath", "", "Path to gateway relayer private key for signing transactions")
	gatewayRelayerKeyPassPhrase = NodeCmd.Flags().String("gatewayRelayerKeyPassPhrase", "", "Pass phrase used to unarmor the gateway relayer key file")
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
		*ccqP2pBootstrap = fmt.Sprintf("/dns4/guardian-0.guardian/udp/%d/quic/p2p/%s", *ccqP2pPort, g0key.String())

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
		*arbitrumContract = unsafeDevModeEvmContractAddress(*arbitrumContract)
		*optimismContract = unsafeDevModeEvmContractAddress(*optimismContract)
		*baseContract = unsafeDevModeEvmContractAddress(*baseContract)
		*sepoliaContract = unsafeDevModeEvmContractAddress(*sepoliaContract)
		*holeskyContract = unsafeDevModeEvmContractAddress(*holeskyContract)
		*scrollContract = unsafeDevModeEvmContractAddress(*scrollContract)
		*mantleContract = unsafeDevModeEvmContractAddress(*mantleContract)
		*arbitrumSepoliaContract = unsafeDevModeEvmContractAddress(*arbitrumSepoliaContract)
		*baseSepoliaContract = unsafeDevModeEvmContractAddress(*baseSepoliaContract)
		*optimismSepoliaContract = unsafeDevModeEvmContractAddress(*optimismSepoliaContract)
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

	if (*baseRPC == "") != (*baseContract == "") {
		logger.Fatal("Both --baseContract and --baseRPC must be set together or both unset")
	}

	// Scroll should not be allowed in mainnet until its finality policy is understood and implemented in the watcher.
	if *scrollRPC != "" && !*testnetMode && !*unsafeDevMode {
		logger.Fatal("scroll is currently only supported in devnet and testnet")
	}

	if (*scrollRPC == "") != (*scrollContract == "") {
		logger.Fatal("Both --scrollContract and --scrollRPC must be set together or both unset")
	}

	// Mantle should not be allowed in mainnet until its finality policy is understood and implemented in the watcher.
	// Note that as of 11/9/2023 Mantle does not support querying for `finalized` or `safe`, just `latest`, so we will need to implement a finalizer.
	if *mantleRPC != "" && !*testnetMode && !*unsafeDevMode {
		logger.Fatal("mantle is currently only supported in devnet and testnet")
	}

	if (*mantleRPC == "") != (*mantleContract == "") {
		logger.Fatal("Both --mantleContract and --mantleRPC must be set together or both unset")
	}

	if *gatewayWS != "" {
		if *gatewayLCD == "" || *gatewayContract == "" {
			logger.Fatal("If --gatewayWS is specified, then --gatewayLCD and --gatewayContract must be specified")
		}
	} else if *gatewayLCD != "" || *gatewayContract != "" {
		logger.Fatal("If --gatewayWS is not specified, then --gatewayLCD and --gatewayContract must not be specified")
	}

	// These chains are only allowed in devnet and testnet.
	if *testnetMode || *unsafeDevMode {
		if (*neonRPC == "") != (*neonContract == "") {
			logger.Fatal("Both --neonRPC and --neonContract must be set together or both unset")
		}
		if (*sepoliaRPC == "") != (*sepoliaContract == "") {
			logger.Fatal("Both --sepoliaRPC and --sepoliaContract must be set together or both unset")
		}
		if (*holeskyRPC == "") != (*holeskyContract == "") {
			logger.Fatal("Both --holeskyRPC and --holeskyContract must be set together or both unset")
		}
		if (*arbitrumSepoliaRPC == "") != (*arbitrumSepoliaContract == "") {
			logger.Fatal("Both --arbitrumSepoliaRPC and --arbitrumSepoliaContract must be set together or both unset")
		}
		if (*baseSepoliaRPC == "") != (*baseSepoliaContract == "") {
			logger.Fatal("Both --baseSepoliaRPC and --baseSepoliaContract must be set together or both unset")
		}
		if (*optimismSepoliaRPC == "") != (*optimismSepoliaContract == "") {
			logger.Fatal("Both --optimismSepoliaRPC and --optimismSepoliaContract must be set together or both unset")
		}
	} else {
		if *neonRPC != "" || *neonContract != "" {
			logger.Fatal("Please do not specify --neonRPC or --neonContract")
		}
		if *sepoliaRPC != "" || *sepoliaContract != "" {
			logger.Fatal("Please do not specify --sepoliaRPC or --sepoliaContract")
		}
		if *holeskyRPC != "" || *holeskyContract != "" {
			logger.Fatal("Please do not specify --holeskyRPC or --holeskyContract")
		}
		if *arbitrumSepoliaRPC != "" || *arbitrumSepoliaContract != "" {
			logger.Fatal("Please do not specify --arbitrumSepoliaRPC or --arbitrumSepoliaContract")
		}
		if *baseSepoliaRPC != "" || *baseSepoliaContract != "" {
			logger.Fatal("Please do not specify --baseSepoliaRPC or --baseSepoliaContract")
		}
		if *optimismSepoliaRPC != "" || *optimismSepoliaContract != "" {
			logger.Fatal("Please do not specify --optimismSepoliaRPC or --optimismSepoliaContract")
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
		if *pythnetWS == "" {
			logger.Fatal("Please specify --pythnetWS")
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
		err := devnet.GenerateAndStoreDevnetGuardianKey(*guardianKeyPath)
		if err != nil {
			logger.Fatal("failed to generate devnet guardian key", zap.Error(err))
		}
	}

	// Database
	db := db.OpenDb(logger, dataDir)
	defer db.Close()

	// Guardian key
	gk, err := common.LoadGuardianKey(*guardianKeyPath, *unsafeDevMode)
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
			// TODO this is a hack. If this is not the bootstrap Guardian, we wait 10s such that the bootstrap Guardian has enough time to start.
			// This may no longer be necessary because now the p2p.go ensures that it can connect to at least one bootstrap peer and will
			// exit the whole guardian if it is unable to. Sleeping here for a bit may reduce overall startup time by preventing unnecessary restarts, though.
			logger.Info("This is not a bootstrap Guardian. Waiting another 10 seconds for the bootstrap guardian to come online.")
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
	rpcMap["accountantWS"] = *accountantWS
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
	rpcMap["ibcBlockHeightURL"] = *ibcBlockHeightURL
	rpcMap["ibcLCD"] = *ibcLCD
	rpcMap["ibcWS"] = *ibcWS
	rpcMap["injectiveLCD"] = *injectiveLCD
	rpcMap["injectiveWS"] = *injectiveWS
	rpcMap["karuraRPC"] = *karuraRPC
	rpcMap["klaytnRPC"] = *klaytnRPC
	rpcMap["mantleRPC"] = *mantleRPC
	rpcMap["moonbeamRPC"] = *moonbeamRPC
	rpcMap["nearRPC"] = *nearRPC
	rpcMap["neonRPC"] = *neonRPC
	rpcMap["oasisRPC"] = *oasisRPC
	rpcMap["optimismRPC"] = *optimismRPC
	rpcMap["polygonRPC"] = *polygonRPC
	rpcMap["pythnetRPC"] = *pythnetRPC
	rpcMap["pythnetWS"] = *pythnetWS
	if env == common.TestNet {
		rpcMap["sepoliaRPC"] = *sepoliaRPC
		rpcMap["holeskyRPC"] = *holeskyRPC
		rpcMap["arbitrumSepoliaRPC"] = *arbitrumSepoliaRPC
		rpcMap["baseSepoliaRPC"] = *baseSepoliaRPC
		rpcMap["optimismSepoliaRPC"] = *optimismSepoliaRPC
	}
	rpcMap["scrollRPC"] = *scrollRPC
	rpcMap["solanaRPC"] = *solanaRPC
	rpcMap["suiRPC"] = *suiRPC
	rpcMap["suiWS"] = *suiWS
	rpcMap["terraWS"] = *terraWS
	rpcMap["terraLCD"] = *terraLCD
	rpcMap["terra2WS"] = *terra2WS
	rpcMap["terra2LCD"] = *terra2LCD
	rpcMap["gatewayWS"] = *gatewayWS
	rpcMap["gatewayLCD"] = *gatewayLCD
	rpcMap["wormchainURL"] = *wormchainURL
	rpcMap["xplaWS"] = *xplaWS
	rpcMap["xplaLCD"] = *xplaLCD

	for _, ibcChain := range ibc.Chains {
		rpcMap[ibcChain.String()] = "IBC"
	}

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

	var hasTelemetryCredential bool = usingLoki

	// Telemetry is enabled by default in mainnet/testnet. In devnet it is disabled by default
	if !*disableTelemetry && (!*unsafeDevMode || *unsafeDevMode && hasTelemetryCredential) {
		if !hasTelemetryCredential {
			logger.Fatal("Please specify --telemetryLokiURL or set --disableTelemetry=false")
		}

		// Get libp2p peer ID from private key
		pk := p2pKey.GetPublic()
		peerID, err := peer.IDFromPublicKey(pk)
		if err != nil {
			logger.Fatal("Failed to get peer ID from private key", zap.Error(err))
		}

		labels := map[string]string{
			"node_name":     *nodeName,
			"node_key":      peerID.String(),
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

			tm, err = telemetry.NewLokiCloudLogger(context.Background(), logger, *telemetryLokiURL, "wormhole", skipPrivateLogs, labels)
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

	wormchainId := "wormchain"
	if *testnetMode {
		wormchainId = "wormchain-testnet-0"
	}

	var accountantWormchainConn *wormconn.ClientConn
	if *accountantContract != "" {
		// TODO: wormchainKeyPath and wormchainKeyPassPhrase are being replaced by accountantKeyPath and accountantKeyPassPhrase.
		//       Give the guardians time to migrate off of the old parameters, but then remove them.
		keyPath := *accountantKeyPath
		if keyPath == "" {
			if *wormchainKeyPath == "" {
				logger.Fatal("if accountantContract is specified, accountantKeyPath is required", zap.String("component", "gacct"))
			}
			logger.Error("the wormchainKeyPath parameter is deprecated, please change to accountantKeyPath", zap.String("component", "gacct"))
			keyPath = *wormchainKeyPath
		} else if *wormchainKeyPath != "" {
			logger.Fatal("the wormchainKeyPath parameter is obsolete, please remove it", zap.String("component", "gacct"))
		}

		keyPassPhrase := *accountantKeyPassPhrase
		if keyPassPhrase == "" {
			if *wormchainKeyPassPhrase == "" {
				logger.Fatal("if accountantContract is specified, accountantKeyPassPhrase is required", zap.String("component", "gacct"))
			}
			logger.Error("the wormchainKeyPassPhrase parameter is deprecated, please change to accountantKeyPassPhrase", zap.String("component", "gacct"))
			keyPassPhrase = *wormchainKeyPassPhrase
		} else if *wormchainKeyPassPhrase != "" {
			logger.Fatal("the wormchainKeyPassPhrase parameter is obsolete, please remove it", zap.String("component", "gacct"))
		}

		keyPathName := keyPath
		if *unsafeDevMode {
			idx, err := devnet.GetDevnetIndex()
			if err != nil {
				logger.Fatal("failed to get devnet index", zap.Error(err), zap.String("component", "gacct"))
			}
			keyPathName = fmt.Sprint(keyPath, idx)
		}

		wormchainKey, err := wormconn.LoadWormchainPrivKey(keyPathName, keyPassPhrase)
		if err != nil {
			logger.Fatal("failed to load wormchain private key", zap.Error(err), zap.String("component", "gacct"))
		}

		// Connect to wormchain.
		logger.Info("Connecting to wormchain", zap.String("wormchainURL", *wormchainURL), zap.String("keyPath", keyPathName), zap.String("component", "gacct"))
		accountantWormchainConn, err = wormconn.NewConn(rootCtx, *wormchainURL, wormchainKey, wormchainId)
		if err != nil {
			logger.Fatal("failed to connect to wormchain", zap.Error(err), zap.String("component", "gacct"))
		}
	}

	var gatewayRelayerWormchainConn *wormconn.ClientConn
	if *gatewayRelayerContract != "" {
		if *wormchainURL == "" {
			logger.Fatal("if gatewayRelayerContract is specified, wormchainURL is required", zap.String("component", "gwrelayer"))
		}
		if *gatewayRelayerKeyPath == "" {
			logger.Fatal("if gatewayRelayerContract is specified, gatewayRelayerKeyPath is required", zap.String("component", "gwrelayer"))
		}

		if *gatewayRelayerKeyPassPhrase == "" {
			logger.Fatal("if gatewayRelayerContract is specified, gatewayRelayerKeyPassPhrase is required", zap.String("component", "gwrelayer"))
		}

		wormchainKeyPathName := *gatewayRelayerKeyPath
		if *unsafeDevMode {
			idx, err := devnet.GetDevnetIndex()
			if err != nil {
				logger.Fatal("failed to get devnet index", zap.Error(err), zap.String("component", "gwrelayer"))
			}
			wormchainKeyPathName = fmt.Sprint(*gatewayRelayerKeyPath, idx)
		}

		wormchainKey, err := wormconn.LoadWormchainPrivKey(wormchainKeyPathName, *gatewayRelayerKeyPassPhrase)
		if err != nil {
			logger.Fatal("failed to load private key", zap.Error(err), zap.String("component", "gwrelayer"))
		}

		logger.Info("Connecting to wormchain", zap.String("wormchainURL", *wormchainURL), zap.String("keyPath", wormchainKeyPathName), zap.String("component", "gwrelayer"))
		gatewayRelayerWormchainConn, err = wormconn.NewConn(rootCtx, *wormchainURL, wormchainKey, wormchainId)
		if err != nil {
			logger.Fatal("failed to connect to wormchain", zap.Error(err), zap.String("component", "gwrelayer"))
		}

	}
	usingPromRemoteWrite := *promRemoteURL != ""
	if usingPromRemoteWrite {
		var info promremotew.PromTelemetryInfo
		info.PromRemoteURL = *promRemoteURL
		info.Labels = map[string]string{
			"node_name":     *nodeName,
			"guardian_addr": ethcrypto.PubkeyToAddress(gk.PublicKey).String(),
			"network":       *p2pNetworkID,
			"version":       version.Version(),
			"product":       "wormhole",
		}

		promLogger := logger.With(zap.String("component", "prometheus_scraper"))
		errC := make(chan error)
		common.StartRunnable(rootCtx, errC, false, "prometheus_scraper", func(ctx context.Context) error {
			t := time.NewTicker(15 * time.Second)

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-t.C:
					err := promremotew.ScrapeAndSendLocalMetrics(ctx, info, promLogger)
					if err != nil {
						promLogger.Error("ScrapeAndSendLocalMetrics error", zap.Error(err))
						continue
					}
				}
			}
		})
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
			NetworkID: "bsc",
			ChainID:   vaa.ChainIDBSC,
			Rpc:       *bscRPC,
			Contract:  *bscContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(polygonRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "polygon",
			ChainID:   vaa.ChainIDPolygon,
			Rpc:       *polygonRPC,
			Contract:  *polygonContract,
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

	if shouldStart(baseRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "base",
			ChainID:   vaa.ChainIDBase,
			Rpc:       *baseRPC,
			Contract:  *baseContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(scrollRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "scroll",
			ChainID:   vaa.ChainIDScroll,
			Rpc:       *scrollRPC,
			Contract:  *scrollContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if shouldStart(mantleRPC) {
		wc := &evm.WatcherConfig{
			NetworkID: "mantle",
			ChainID:   vaa.ChainIDMantle,
			Rpc:       *mantleRPC,
			Contract:  *mantleContract,
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

	if shouldStart(gatewayWS) {
		wc := &cosmwasm.WatcherConfig{
			NetworkID: "gateway",
			ChainID:   vaa.ChainIDWormchain,
			Websocket: *gatewayWS,
			Lcd:       *gatewayLCD,
			Contract:  *gatewayContract,
		}

		watcherConfigs = append(watcherConfigs, wc)
	}

	if *testnetMode && shouldStart(neonRPC) {
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

	if *testnetMode || *unsafeDevMode {
		if shouldStart(sepoliaRPC) {
			wc := &evm.WatcherConfig{
				NetworkID: "sepolia",
				ChainID:   vaa.ChainIDSepolia,
				Rpc:       *sepoliaRPC,
				Contract:  *sepoliaContract,
			}

			watcherConfigs = append(watcherConfigs, wc)
		}

		if shouldStart(holeskyRPC) {
			wc := &evm.WatcherConfig{
				NetworkID: "holesky",
				ChainID:   vaa.ChainIDHolesky,
				Rpc:       *holeskyRPC,
				Contract:  *holeskyContract,
			}

			watcherConfigs = append(watcherConfigs, wc)
		}

		if shouldStart(arbitrumSepoliaRPC) {
			wc := &evm.WatcherConfig{
				NetworkID: "arbitrum_sepolia",
				ChainID:   vaa.ChainIDArbitrumSepolia,
				Rpc:       *arbitrumSepoliaRPC,
				Contract:  *arbitrumSepoliaContract,
			}

			watcherConfigs = append(watcherConfigs, wc)
		}

		if shouldStart(baseSepoliaRPC) {
			wc := &evm.WatcherConfig{
				NetworkID: "base_sepolia",
				ChainID:   vaa.ChainIDBaseSepolia,
				Rpc:       *baseSepoliaRPC,
				Contract:  *baseSepoliaContract,
			}

			watcherConfigs = append(watcherConfigs, wc)
		}

		if shouldStart(optimismSepoliaRPC) {
			wc := &evm.WatcherConfig{
				NetworkID: "optimism_sepolia",
				ChainID:   vaa.ChainIDOptimismSepolia,
				Rpc:       *optimismSepoliaRPC,
				Contract:  *optimismSepoliaContract,
			}

			watcherConfigs = append(watcherConfigs, wc)
		}
	}

	var ibcWatcherConfig *node.IbcWatcherConfig = nil
	if shouldStart(ibcWS) {
		ibcWatcherConfig = &node.IbcWatcherConfig{
			Websocket:      *ibcWS,
			Lcd:            *ibcLCD,
			BlockHeightURL: *ibcBlockHeightURL,
			Contract:       *ibcContract,
		}
	}

	guardianNode := node.NewGuardianNode(
		env,
		gk,
	)

	guardianOptions := []*node.GuardianOption{
		node.GuardianOptionDatabase(db),
		node.GuardianOptionWatchers(watcherConfigs, ibcWatcherConfig),
		node.GuardianOptionAccountant(*accountantContract, *accountantWS, *accountantCheckEnabled, accountantWormchainConn),
		node.GuardianOptionGovernor(*chainGovernorEnabled),
		node.GuardianOptionGatewayRelayer(*gatewayRelayerContract, gatewayRelayerWormchainConn),
		node.GuardianOptionQueryHandler(*ccqEnabled, *ccqAllowedRequesters),
		node.GuardianOptionAdminService(*adminSocketPath, ethRPC, ethContract, rpcMap),
		node.GuardianOptionP2P(p2pKey, *p2pNetworkID, *p2pBootstrap, *nodeName, *disableHeartbeatVerify, *p2pPort, *ccqP2pBootstrap, *ccqP2pPort, *ccqAllowedPeers, ibc.GetFeatures),
		node.GuardianOptionStatusServer(*statusAddr),
		node.GuardianOptionProcessor(),
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

	// Run supervisor with Guardian Node as root.
	supervisor.New(rootCtx, logger, guardianNode.Run(rootCtxCancel, guardianOptions...),
		// It's safer to crash and restart the process in case we encounter a panic,
		// rather than attempting to reschedule the runnable.
		supervisor.WithPropagatePanic)

	<-rootCtx.Done()
	logger.Info("root context cancelled, exiting...")
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
