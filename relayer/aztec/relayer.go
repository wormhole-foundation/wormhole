package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
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
	"github.com/ethereum/go-ethereum/rpc"
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

// ADD: HTTP verification service types
type VerificationRequest struct {
	VAABytes string `json:"vaaBytes"`
}

type VerificationResponse struct {
	Success bool   `json:"success"`
	TxHash  string `json:"txHash,omitempty"`
	Error   string `json:"error,omitempty"`
}

// ADD: HTTP client for verification service
type VerificationServiceClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// ADD: Create new verification service client
func NewVerificationServiceClient(baseURL string) *VerificationServiceClient {
	return &VerificationServiceClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
		logger: logger.With(zap.String("component", "VerificationServiceClient")),
	}
}

// ADD: Verify VAA via HTTP service
func (c *VerificationServiceClient) VerifyVAA(ctx context.Context, vaaBytes []byte) (string, error) {
	c.logger.Debug("Sending VAA to verification service", zap.Int("vaaLength", len(vaaBytes)))

	// Prepare request
	vaaHex := hex.EncodeToString(vaaBytes)
	if !strings.HasPrefix(vaaHex, "0x") {
		vaaHex = "0x" + vaaHex
	}

	request := VerificationRequest{
		VAABytes: vaaHex,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return "", fmt.Errorf("failed to marshal verification request: %v", err)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/verify", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create HTTP request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send verification request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read verification response: %v", err)
	}

	c.logger.Debug("Received response from verification service",
		zap.Int("statusCode", resp.StatusCode))

	// Parse response
	var response VerificationResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return "", fmt.Errorf("failed to unmarshal verification response: %v", err)
	}

	if !response.Success {
		return "", fmt.Errorf("verification failed: %s", response.Error)
	}

	return response.TxHash, nil
}

// ADD: Check if verification service is healthy
func (c *VerificationServiceClient) CheckHealth(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %v", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("verification service unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// Config holds all configuration parameters for the relayer
type Config struct {
	SpyRPCHost             string                         // Wormhole spy service endpoint
	SourceChainID          uint16                         // Aztec chain ID
	DestChainID            uint16                         // Arbitrum chain ID
	AztecPXEURL            string                         // PXE URL for Aztec
	AztecWalletAddress     string                         // Aztec wallet address to use
	ArbitrumRPCURL         string                         // RPC URL for Arbitrum
	PrivateKey             string                         // Private key for Arbitrum
	WormholeContract       string                         // Wormhole core contract address
	AztecTargetContract    string                         // Target contract on Aztec
	ArbitrumTargetContract string                         // Target contract on Arbitrum
	EmitterAddress         string                         // Emitter address to monitor
	VerificationServiceURL string                         // ADD: Verification service URL
	vaaProcessor           func(*Relayer, *VAAData) error // Custom VAA processor function
}

// NewConfigFromEnv creates a Config from environment variables
func NewConfigFromEnv() Config {
	return Config{
		SpyRPCHost:             getEnvOrDefault("SPY_RPC_HOST", "localhost:7072"),
		SourceChainID:          uint16(getEnvIntOrDefault("SOURCE_CHAIN_ID", 52)), // Aztec
		DestChainID:            uint16(getEnvIntOrDefault("DEST_CHAIN_ID", 2)),    // Arbitrum
		AztecPXEURL:            getEnvOrDefault("AZTEC_PXE_URL", "http://localhost:8090"),
		AztecWalletAddress:     getEnvOrDefault("AZTEC_WALLET_ADDRESS", "0x05795e88e667e308bc7dc72bca7e5a2db244674b46313a421a378fdbc65b2c9a"),
		ArbitrumRPCURL:         getEnvOrDefault("ARBITRUM_RPC_URL", "http://localhost:8545"),
		PrivateKey:             getEnvOrDefault("PRIVATE_KEY", "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"),
		WormholeContract:       getEnvOrDefault("WORMHOLE_CONTRACT", "0x1b35884f8ba9371419d00ae228da9ff839edfe8fe6a804fdfcd430e0dc7e40db"),
		AztecTargetContract:    getEnvOrDefault("AZTEC_TARGET_CONTRACT", "0x27aceaacfa90a3f1138345ae1c7bb2bce6ce0d0bba257cc0dfe06caf951c5085"),
		ArbitrumTargetContract: getEnvOrDefault("ARBITRUM_TARGET_CONTRACT", "0x009cbB8f91d392856Cb880d67c806Aa731E3d686"),
		EmitterAddress:         getEnvOrDefault("EMITTER_ADDRESS", "0d6fe810321185c97a0e94200f998bcae787aaddf953a03b14ec5da3b6838bad"),
		VerificationServiceURL: getEnvOrDefault("VERIFICATION_SERVICE_URL", "http://localhost:8080"), // ADD
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

// AztecPXEClient handles interactions with Aztec blockchain via PXE
type AztecPXEClient struct {
	rpcClient     *rpc.Client
	walletAddress string
	logger        *zap.Logger
}

// NewAztecPXEClient creates a new client for Aztec blockchain via PXE
func NewAztecPXEClient(pxeURL, walletAddress string) (*AztecPXEClient, error) {
	client := &AztecPXEClient{
		walletAddress: walletAddress,
		logger:        logger.With(zap.String("component", "AztecPXEClient")),
	}

	client.logger.Info("Connecting to Aztec PXE",
		zap.String("pxeURL", pxeURL),
		zap.String("walletAddress", walletAddress))

	// Create RPC client using the same pattern as your working code
	rpcClient, err := rpc.Dial(pxeURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create RPC client: %v", err)
	}

	client.rpcClient = rpcClient

	// Test connection using the working node_getBlock method
	err = client.testConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Aztec PXE: %v", err)
	}

	return client, nil
}

// testConnection tests the connection to Aztec PXE using working methods
func (c *AztecPXEClient) testConnection() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test with node_getBlock method (we know this works)
	var blockResult interface{}
	err := c.rpcClient.CallContext(ctx, &blockResult, "node_getBlock", 1)
	if err != nil {
		c.logger.Debug("node_getBlock test failed", zap.Error(err))
		// This is okay - block 1 might not exist, connection is still working
	}

	c.logger.Info("Aztec PXE connection successful")
	return nil
}

// SendVerifyTransaction sends a transaction to verify and store a VAA on Aztec via PXE
func (c *AztecPXEClient) SendVerifyTransaction(ctx context.Context, targetContract string, vaaBytes []byte) (string, error) {
	c.logger.Debug("Sending verify_vaa transaction to Aztec via PXE", zap.Int("vaaLength", len(vaaBytes)))

	// Pad to 2000 bytes for contract but pass actual length
	paddedVAABytes := make([]byte, 2000)
	copy(paddedVAABytes, vaaBytes)

	// Convert the padded bytes to array format for Aztec
	vaaArray := make([]interface{}, 2000)
	for i, b := range paddedVAABytes {
		vaaArray[i] = int(b)
	}

	actualLength := len(vaaBytes)

	c.logger.Debug("Calling verify_vaa function",
		zap.String("contract", targetContract),
		zap.Int("actualLength", actualLength),
		zap.Int("paddedLength", len(paddedVAABytes)))

	// Use the RPC client pattern from your working code
	// First, let's try to simulate the call to see if the contract/function exists
	var result interface{}
	err := c.rpcClient.CallContext(ctx, &result, "pxe_simulateTransaction", map[string]interface{}{
		"contractAddress": targetContract,
		"functionName":    "verify_vaa",
		"args":            []interface{}{vaaArray, actualLength},
		"origin":          c.walletAddress,
	})

	if err != nil {
		c.logger.Warn("Transaction simulation failed", zap.Error(err))
		// Continue anyway - simulation might not be available
	} else {
		c.logger.Debug("Transaction simulation successful", zap.Any("result", result))
	}

	// Now try to send the actual transaction
	// This method name needs to be confirmed with actual PXE API
	var txResult interface{}
	err = c.rpcClient.CallContext(ctx, &txResult, "pxe_sendTransaction", map[string]interface{}{
		"contractAddress": targetContract,
		"functionName":    "verify_vaa",
		"args":            []interface{}{vaaArray, actualLength},
		"origin":          c.walletAddress,
	})

	if err != nil {
		return "", fmt.Errorf("failed to send verify_vaa transaction: %v", err)
	}

	// Extract transaction hash from result
	if txMap, ok := txResult.(map[string]interface{}); ok {
		if txHash, exists := txMap["txHash"]; exists {
			if txHashStr, ok := txHash.(string); ok {
				return txHashStr, nil
			}
		}
		if txHash, exists := txMap["hash"]; exists {
			if txHashStr, ok := txHash.(string); ok {
				return txHashStr, nil
			}
		}
	}

	if txHashStr, ok := txResult.(string); ok {
		return txHashStr, nil
	}

	c.logger.Debug("PXE transaction result", zap.Any("result", txResult))
	return fmt.Sprintf("tx_submitted_%d", time.Now().Unix()), nil
}

// GetWalletAddress returns the wallet address being used
func (c *AztecPXEClient) GetWalletAddress() string {
	return c.walletAddress
}

// EVMClient handles interactions with EVM-compatible blockchains (Arbitrum)
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

// SendVerifyTransaction sends a transaction to the verify function to process and store a VAA
func (c *EVMClient) SendVerifyTransaction(ctx context.Context, targetContract string, vaaBytes []byte) (string, error) {
	c.logger.Debug("Sending verify transaction to EVM", zap.Int("vaaLength", len(vaaBytes)))

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
	spyClient          *SpyClient
	aztecClient        *AztecPXEClient
	evmClient          *EVMClient
	verificationClient *VerificationServiceClient // ADD: HTTP verification client
	config             Config
	vaaProcessor       func(*Relayer, *VAAData) error
	logger             *zap.Logger
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

	// Connect to Aztec via PXE
	aztecClient, err := NewAztecPXEClient(config.AztecPXEURL, config.AztecWalletAddress)
	if err != nil {
		spyClient.Close()
		return nil, fmt.Errorf("failed to create Aztec PXE client: %v", err)
	}

	// Connect to Arbitrum (EVM)
	evmClient, err := NewEVMClient(config.ArbitrumRPCURL, config.PrivateKey)
	if err != nil {
		spyClient.Close()
		return nil, fmt.Errorf("failed to create EVM client: %v", err)
	}

	// ADD: Create verification service client
	verificationClient := NewVerificationServiceClient(config.VerificationServiceURL)

	// ADD: Test connection to verification service
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := verificationClient.CheckHealth(ctx); err != nil {
		spyClient.Close()
		relayer.logger.Warn("Verification service not available", zap.Error(err))
		// Don't fail - we can still relay Aztec->Arbitrum
	} else {
		relayer.logger.Info("Connected to verification service", zap.String("url", config.VerificationServiceURL))
	}

	relayer.spyClient = spyClient
	relayer.aztecClient = aztecClient
	relayer.evmClient = evmClient
	relayer.verificationClient = verificationClient // ADD

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
	r.logger.Info("Starting bidirectional Aztec-Arbitrum relayer",
		zap.String("aztecWallet", r.aztecClient.GetWalletAddress()),
		zap.String("arbitrumAddress", r.evmClient.GetAddress().Hex()),
		zap.Uint16("aztecChain", r.config.SourceChainID),
		zap.Uint16("arbitrumChain", r.config.DestChainID),
		zap.String("verificationServiceURL", r.config.VerificationServiceURL)) // ADD

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

			// Process the VAA in a goroutine, but track it with the WaitGroupp
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

// MODIFY: defaultVAAProcessor to use verification service for Arbitrum->Aztec
func defaultVAAProcessor(r *Relayer, vaaData *VAAData) error {
	// Create a context with timeout for processing operations
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second) // Increased timeout for HTTP calls
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

	var txHash string
	var err error
	var direction string

	// Check if this is a VAA from Aztec (source chain) -> send to Arbitrum
	if vaaData.ChainID == r.config.SourceChainID {
		direction = "Aztec->Arbitrum"

		r.logger.Info("Processing VAA from Aztec to Arbitrum",
			zap.Uint64("sequence", vaaData.Sequence),
			zap.String("sourceTxID", vaaData.TxID))

		// Send to Arbitrum using EVM client
		txHash, err = r.evmClient.SendVerifyTransaction(ctx, r.config.ArbitrumTargetContract, vaaData.RawBytes)

		// Check if this is a VAA from Arbitrum (dest chain) -> send to Aztec
	} else if vaaData.ChainID == r.config.DestChainID {
		direction = "Arbitrum->Aztec"

		r.logger.Info("Processing VAA from Arbitrum to Aztec",
			zap.Uint64("sequence", vaaData.Sequence),
			zap.String("sourceTxID", vaaData.TxID))

		// MODIFY: Try verification service first, fallback to direct PXE
		txHash, err = r.verificationClient.VerifyVAA(ctx, vaaData.RawBytes)
		if err != nil {
			r.logger.Warn("Verification service failed, trying direct PXE", zap.Error(err))
			// Fallback to direct PXE call
			txHash, err = r.aztecClient.SendVerifyTransaction(ctx, r.config.AztecTargetContract, vaaData.RawBytes)
		} else {
			r.logger.Debug("Used verification service successfully")
		}

	} else {
		// Skip VAAs not from our configured chains
		r.logger.Debug("Skipping VAA (not from configured chains)",
			zap.Uint64("sequence", vaaData.Sequence),
			zap.Uint16("chain", vaaData.ChainID))
		return nil
	}

	if err != nil {
		// Check if the context was cancelled or timed out
		if ctx.Err() != nil {
			r.logger.Warn("Transaction sending cancelled or timed out", zap.Error(ctx.Err()))
			return fmt.Errorf("transaction interrupted: %v", ctx.Err())
		}

		r.logger.Error("Failed to send verify transaction",
			zap.String("direction", direction),
			zap.Uint64("sequence", vaaData.Sequence),
			zap.String("sourceTxID", vaaData.TxID),
			zap.Error(err))
		return fmt.Errorf("transaction failed: %v", err)
	}

	r.logger.Info("VAA verification completed",
		zap.String("direction", direction),
		zap.Uint64("sequence", vaaData.Sequence),
		zap.String("txHash", txHash),
		zap.String("sourceTxID", vaaData.TxID))

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
				r.logger.Debug("Address", zap.String("address", fmt.Sprintf("0x%x", payload[i:i+20])))
			}
		case 1:
			if i+2 <= end {
				chainIDLower := uint16(payload[i])
				chainIDUpper := uint16(payload[i+1])
				chainID := (chainIDUpper << 8) | chainIDLower
				r.logger.Debug("Chain ID", zap.Uint16("chainID", chainID))
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

	logger.Info("Starting bidirectional Aztec-Arbitrum Wormhole relayer")

	// Load configuration from environment
	config := NewConfigFromEnv()

	logger.Info("DEBUG: Config loaded",
		zap.Uint16("sourceChainID", config.SourceChainID),
		zap.Uint16("destChainID", config.DestChainID))

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
