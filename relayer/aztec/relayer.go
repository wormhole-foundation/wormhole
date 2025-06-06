package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	spyv1 "github.com/certusone/wormhole/node/pkg/proto/spy/v1"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	vaaLib "github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Global logger for initial setup
var logger *zap.Logger

// Initialize global logger
func initLogger() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		// Fallback to standard logger if zap fails
		fmt.Printf("Failed to initialize zap logger: %v\n", err)
		logger = zap.NewExample()
	}
}

// Config holds all configuration parameters for the relayer
type Config struct {
	SpyRPCHost       string                         // Wormhole spy service endpoint
	SourceChainID    uint16                         // Chain ID to receive VAAs from
	DestChainID      uint16                         // Chain ID to relay VAAs to
	DestRPCURL       string                         // RPC URL for the destination chain
	PrivateKey       string                         // Private key for transaction signing
	WormholeContract string                         // Wormhole core contract address
	TargetContract   string                         // Target contract to call with VAAs
	EmitterAddress   string                         // Emitter address to monitor
	vaaProcessor     func(*Relayer, *VAAData) error // Custom VAA processor function
}

// NewConfigFr\omEnv creates a Config from environment variables
func NewConfigFromEnv() Config {
	return Config{
		SpyRPCHost:       getEnvOrDefault("SPY_RPC_HOST", "localhost:7072"),
		SourceChainID:    uint16(getEnvIntOrDefault("SOURCE_CHAIN_ID", 52)),
		DestChainID:      uint16(getEnvIntOrDefault("DEST_CHAIN_ID", 10004)),
		DestRPCURL:       getEnvOrDefault("DEST_RPC_URL", "http://localhost:8545"),
		PrivateKey:       getEnvOrDefault("PRIVATE_KEY", "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"),
		WormholeContract: getEnvOrDefault("WORMHOLE_CONTRACT", "0x1b35884f8ba9371419d00ae228da9ff839edfe8fe6a804fdfcd430e0dc7e40db"),
		TargetContract:   getEnvOrDefault("TARGET_CONTRACT", "0xb592244aa6477eBDDc14475aaeF921cdDcC0170f"),
		EmitterAddress:   getEnvOrDefault("EMITTER_ADDRESS", "0d6fe810321185c97a0e94200f998bcae787aaddf953a03b14ec5da3b6838bad"),
	}
}

// VAAData encapsulates a VAA and its metadata
type VAAData struct {
	VAA        *vaaLib.VAA // The parsed VAA
	RawBytes   []byte      // Raw VAA bytes
	ChainID    uint16      // Source chain ID
	EmitterHex string      // Hex-encoded emitter address
	Sequence   uint64      // VAA sequence number
	TxID       string      // Source transaction ID
}

// SpyClient handles connections to the Wormhole spy service
type SpyClient struct {
	conn   *grpc.ClientConn
	client spyv1.SpyRPCServiceClient
	logger *zap.Logger
}

// NewSpyClient creates a new client for the Wormhole spy service
func NewSpyClient(endpoint string) (*SpyClient, error) {
	client := &SpyClient{
		logger: logger.With(zap.String("component", "SpyClient")),
	}

	client.logger.Info("Connecting to spy service", zap.String("endpoint", endpoint))
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to spy: %v", err)
	}

	client.conn = conn
	client.client = spyv1.NewSpyRPCServiceClient(conn)
	return client, nil
}

// Close closes the connection to the spy service
func (c *SpyClient) Close() {
	if c.conn != nil {
		c.conn.Close()
	}
}

// SubscribeSignedVAA subscribes to all signed VAAs with retry logic
func (c *SpyClient) SubscribeSignedVAA(ctx context.Context) (spyv1.SpyRPCService_SubscribeSignedVAAClient, error) {
	const maxRetries = 5
	const retryDelay = 2 * time.Second

	c.logger.Debug("Subscribing to signed VAAs")

	var stream spyv1.SpyRPCService_SubscribeSignedVAAClient
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		stream, err = c.client.SubscribeSignedVAA(ctx, &spyv1.SubscribeSignedVAARequest{})
		if err == nil {
			return stream, nil
		}

		if attempt < maxRetries {
			c.logger.Warn("Subscribe attempt failed",
				zap.Int("attempt", attempt),
				zap.Error(err),
				zap.Duration("retryIn", retryDelay))

			select {
			case <-time.After(retryDelay):
				// Continue to next retry
			case <-ctx.Done():
				return nil, fmt.Errorf("subscribe to signed VAAs: %v", ctx.Err())
			}
		}
	}

	return nil, fmt.Errorf("subscribe to signed VAAs after %d attempts: %v", maxRetries, err)
}

// EVMClient handles interactions with EVM-compatible blockchains
type EVMClient struct {
	client     *ethclient.Client
	privateKey *ecdsa.PrivateKey
	address    common.Address
	logger     *zap.Logger
}

// NewEVMClient creates a new client for EVM-compatible blockchains
func NewEVMClient(rpcURL, privateKeyHex string) (*EVMClient, error) {
	client := &EVMClient{
		logger: logger.With(zap.String("component", "EVMClient")),
	}

	client.logger.Info("Connecting to EVM chain", zap.String("rpcURL", rpcURL))
	ethClient, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EVM node: %v", err)
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	// Derive public address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)

	client.client = ethClient
	client.privateKey = privateKey
	client.address = address

	return client, nil
}

// GetAddress returns the public address for this client
func (c *EVMClient) GetAddress() common.Address {
	return c.address
}

// sendVerifyTransaction sends a transaction to the verify function to process and store a VAA
func (c *EVMClient) sendVerifyTransaction(ctx context.Context, targetContract string, vaaBytes []byte) (string, error) {
	c.logger.Debug("Sending verify transaction", zap.Int("vaaLength", len(vaaBytes)))

	// Contract ABI for the verify function
	const abiJSON = `[{
        "inputs": [
            {"internalType": "bytes", "name": "encodedVm", "type": "bytes"}
        ],
        "name": "verify",
        "outputs": [],
        "stateMutability": "nonpayable",
        "type": "function"
    }]`

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return "", fmt.Errorf("ABI parse error: %v", err)
	}

	// Pack the function call data
	data, err := parsedABI.Pack("verify", vaaBytes)
	if err != nil {
		return "", fmt.Errorf("ABI pack error: %v", err)
	}

	// Get the latest nonce for our account
	nonce, err := c.client.PendingNonceAt(ctx, c.address)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %v", err)
	}

	// Get the current gas price
	gasPrice, err := c.client.SuggestGasPrice(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %v", err)
	}

	// Create the transaction
	targetAddr := common.HexToAddress(targetContract)
	tx := types.NewTransaction(
		nonce,
		targetAddr,
		big.NewInt(0), // No ETH being sent
		3000000,       // Gas limit - adjust as needed
		gasPrice,
		data,
	)

	// Get the chain ID
	chainID, err := c.client.NetworkID(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get chain ID: %v", err)
	}

	// Sign the transaction
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), c.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %v", err)
	}

	// Send the transaction
	err = c.client.SendTransaction(ctx, signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %v", err)
	}

	return signedTx.Hash().Hex(), nil
}

// Relayer coordinates processing VAAs from the spy service
type Relayer struct {
	spyClient    *SpyClient
	evmClient    *EVMClient
	config       Config
	vaaProcessor func(*Relayer, *VAAData) error
	logger       *zap.Logger
}

// NewRelayer creates a new relayer instance
func NewRelayer(config Config) (*Relayer, error) {
	relayer := &Relayer{
		config: config,
		logger: logger.With(zap.String("component", "Relayer")),
	}

	// Connect to the spy service
	spyClient, err := NewSpyClient(config.SpyRPCHost)
	if err != nil {
		return nil, fmt.Errorf("failed to create spy client: %v", err)
	}

	// Connect to the EVM chain
	evmClient, err := NewEVMClient(config.DestRPCURL, config.PrivateKey)
	if err != nil {
		spyClient.Close()
		return nil, fmt.Errorf("failed to create EVM client: %v", err)
	}

	relayer.spyClient = spyClient
	relayer.evmClient = evmClient

	// Set default VAA processor
	if config.vaaProcessor == nil {
		relayer.vaaProcessor = defaultVAAProcessor
	} else {
		relayer.vaaProcessor = config.vaaProcessor
	}

	return relayer, nil
}

// Close cleans up resources used by the relayer
func (r *Relayer) Close() {
	if r.spyClient != nil {
		r.spyClient.Close()
	}
}

// Start begins listening for VAAs and processing them
func (r *Relayer) Start(ctx context.Context) error {
	r.logger.Info("Starting relayer",
		zap.String("address", r.evmClient.GetAddress().Hex()),
		zap.Uint16("sourceChain", r.config.SourceChainID),
		zap.Uint16("destChain", r.config.DestChainID),
		zap.String("emitter", r.config.EmitterAddress))

	// Create a wait group to track goroutines
	var wg sync.WaitGroup

	// Subscribe to VAAs
	stream, err := r.spyClient.SubscribeSignedVAA(ctx)
	if err != nil {
		return fmt.Errorf("subscribe to VAA stream: %v", err)
	}

	r.logger.Info("Listening for VAAs")

	// Create a separate context for graceful shutdown
	processingCtx, cancelProcessing := context.WithCancel(context.Background())
	defer cancelProcessing()

	for {
		select {
		case <-ctx.Done():
			r.logger.Info("Shutting down relayer")
			// Cancel all processing
			cancelProcessing()
			// Wait for all processing goroutines to complete
			r.logger.Info("Waiting for all VAA processing to complete")
			wg.Wait()
			r.logger.Info("Shutdown complete")
			return nil
		default:
			// Receive the next VAA
			resp, err := stream.Recv()
			if err != nil {
				r.logger.Warn("Stream error, retrying in 5s", zap.Error(err))
				time.Sleep(5 * time.Second)
				stream, err = r.spyClient.SubscribeSignedVAA(ctx)
				if err != nil {
					// Cancel all processing before returning
					cancelProcessing()
					// Wait for all processing goroutines to complete
					wg.Wait()
					return fmt.Errorf("subscribe to VAA stream after retry: %v", err)
				}
				continue
			}

			// Process the VAA in a goroutine, but track it with the WaitGroup
			wg.Add(1)
			go func(vaaBytes []byte) {
				defer wg.Done()
				r.processVAA(processingCtx, vaaBytes)
			}(resp.VaaBytes)
		}
	}
}

func (r *Relayer) processVAA(ctx context.Context, vaaBytes []byte) {
	// Check for context cancellation first
	select {
	case <-ctx.Done():
		r.logger.Debug("Processing cancelled for VAA")
		return
	default:
		// Continue processing
	}

	// Parse the VAA
	wormholeVAA, err := vaaLib.Unmarshal(vaaBytes)
	if err != nil {
		r.logger.Error("Failed to parse VAA", zap.Error(err))
		return
	}

	// Extract the txID from the payload (first 32 bytes)
	txID := ""
	if len(wormholeVAA.Payload) >= 32 {
		txIDBytes := wormholeVAA.Payload[:32]
		txID = fmt.Sprintf("0x%x", txIDBytes)
		r.logger.Debug("Extracted txID from payload", zap.String("txID", txID))
	} else {
		r.logger.Debug("Payload too short to contain txID", zap.Int("payload_length", len(wormholeVAA.Payload)))
	}

	// Create VAA data with essential information
	vaaData := &VAAData{
		VAA:        wormholeVAA,
		RawBytes:   vaaBytes,
		ChainID:    uint16(wormholeVAA.EmitterChain),
		EmitterHex: fmt.Sprintf("%064x", wormholeVAA.EmitterAddress),
		Sequence:   wormholeVAA.Sequence,
		TxID:       txID,
	}

	r.logger.Info("Processing VAA",
		zap.Uint16("chain", vaaData.ChainID),
		zap.Uint64("sequence", vaaData.Sequence),
		zap.String("emitter", vaaData.EmitterHex),
		zap.String("sourceTxID", vaaData.TxID))

	// Use the passed context when calling the processor
	if err := r.vaaProcessor(r, vaaData); err != nil {
		r.logger.Error("Error processing VAA", zap.Error(err))
	}
}

// defaultVAAProcessor is the default VAA processing logic
func defaultVAAProcessor(r *Relayer, vaaData *VAAData) error {
	// Create a context with timeout for processing operations
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Log essential VAA information
	r.logger.Info("VAA Details",
		zap.Uint16("emitterChain", vaaData.ChainID),
		zap.String("emitterAddress", vaaData.EmitterHex),
		zap.Uint64("sequence", vaaData.Sequence),
		zap.Time("timestamp", vaaData.VAA.Timestamp),
		zap.Int("payloadLength", len(vaaData.VAA.Payload)),
		zap.String("sourceTxID", vaaData.TxID))

	// Extract and log key payload information at debug level
	r.logger.Debug("VAA Payload", zap.String("payloadHex", fmt.Sprintf("%x", vaaData.VAA.Payload)))

	// Parse payload structure at debug level
	if len(vaaData.VAA.Payload) >= 32 {
		r.parseAndLogPayload(vaaData.VAA.Payload)
	}

	// Check if this is a VAA from the source chain
	if vaaData.ChainID == r.config.SourceChainID {
		r.logger.Info("Processing VAA from source chain",
			zap.Uint64("sequence", vaaData.Sequence),
			zap.String("sourceTxID", vaaData.TxID))

		// Send the transaction to verify and store the VAA on-chain
		txHash, err := r.evmClient.sendVerifyTransaction(ctx, r.config.TargetContract, vaaData.RawBytes)
		if err != nil {
			// Check if the context was cancelled or timed out
			if ctx.Err() != nil {
				r.logger.Warn("Transaction sending cancelled or timed out", zap.Error(ctx.Err()))
				return fmt.Errorf("transaction interrupted: %v", ctx.Err())
			}

			r.logger.Error("Failed to send verify transaction",
				zap.Uint64("sequence", vaaData.Sequence),
				zap.String("sourceTxID", vaaData.TxID),
				zap.Error(err))
			return fmt.Errorf("transaction failed: %v", err)
		}

		r.logger.Info("VAA verification completed",
			zap.Uint64("sequence", vaaData.Sequence),
			zap.String("txHash", txHash),
			zap.String("sourceTxID", vaaData.TxID))

		return nil
	}

	// Check if this is a VAA for the destination chain
	if vaaData.ChainID == r.config.DestChainID {
		r.logger.Info("Received VAA for destination chain",
			zap.Uint16("chain", vaaData.ChainID),
			zap.Uint64("sequence", vaaData.Sequence))
		return nil
	}

	// Skip VAAs not configured for processing
	r.logger.Debug("Skipping VAA (not configured for processing)",
		zap.Uint64("sequence", vaaData.Sequence),
		zap.Uint16("chain", vaaData.ChainID))
	return nil
}

// parseAndLogPayload parses and logs payload structure at debug level
func (r *Relayer) parseAndLogPayload(payload []byte) {
	const txIDOffset = 32
	const arraySize = 31

	// Log the transaction ID from the first 32 bytes
	if len(payload) >= 32 {
		txIDBytes := payload[:32]
		r.logger.Debug("Source Transaction ID", zap.String("txID", fmt.Sprintf("0x%x", txIDBytes)))
	}

	// Parse payload arrays (skip the txID)
	for i := txIDOffset; i < len(payload); i += arraySize {
		end := i + arraySize
		if end > len(payload) {
			end = len(payload)
		}

		arrayIndex := (i - txIDOffset) / arraySize
		r.logger.Debug(fmt.Sprintf("Payload array %d", arrayIndex),
			zap.String("hex", fmt.Sprintf("0x%x", payload[i:end])))

		// Parse specific fields at debug level
		switch arrayIndex {
		case 0:
			if i+20 <= end {
				r.logger.Debug("Arbitrum address", zap.String("address", fmt.Sprintf("0x%x", payload[i:i+20])))
			}
		case 1:
			if i+2 <= end {
				chainIDLower := uint16(payload[i])
				chainIDUpper := uint16(payload[i+1])
				chainID := (chainIDUpper << 8) | chainIDLower
				r.logger.Debug("Arbitrum chain ID", zap.Uint16("chainID", chainID))
			}
		case 2:
			if i < end {
				amount := uint64(payload[i])
				r.logger.Debug("Amount", zap.Uint64("amount", amount))
			}
		}
	}
}

// Environment variable helpers
func getEnvOrDefault(key, defaultValue string) string {
	val, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}
	return val
}

func getEnvIntOrDefault(key string, defaultValue int) int {
	val, exists := os.LookupEnv(key)
	if !exists {
		return defaultValue
	}

	result, err := strconv.Atoi(val)
	if err != nil {
		logger.Warn("Invalid environment variable value, using default",
			zap.String("key", key),
			zap.Int("default", defaultValue))
		return defaultValue
	}
	return result
}

func main() {
	// Initialize the logger first
	initLogger()
	defer logger.Sync()

	logger.Info("Starting Wormhole relayer")

	// Load configuration from environment
	config := NewConfigFromEnv()

	// Create relayer
	relayer, err := NewRelayer(config)
	if err != nil {
		logger.Fatal("Failed to initialize relayer", zap.Error(err))
	}
	defer relayer.Close()

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		logger.Info("Received shutdown signal")
		cancel()
	}()

	// Start the relayer
	if err := relayer.Start(ctx); err != nil {
		logger.Fatal("Relayer stopped with error", zap.Error(err))
	}
}
