package main

import (
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	spyv1 "github.com/certusone/wormhole/node/pkg/proto/spy/v1"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	vaaLib "github.com/wormhole-foundation/wormhole/sdk/vaa"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds all configuration parameters for the relayer
type Config struct {
	SpyRPCHost       string
	SourceChainID    uint16
	DestChainID      uint16
	DestRPCURL       string
	PrivateKey       string
	WormholeContract string
	TargetContract   string
	EmitterAddress   string
	vaaProcessor     func(*Relayer, *VAAData) error
}

// NewConfigFromEnv creates a Config from environment variables
func NewConfigFromEnv() Config {
	return Config{
		SpyRPCHost:       getEnvOrDefault("SPY_RPC_HOST", "localhost:7072"),
		SourceChainID:    uint16(getEnvIntOrDefault("SOURCE_CHAIN_ID", 52)),
		DestChainID:      uint16(getEnvIntOrDefault("DEST_CHAIN_ID", 10004)),
		DestRPCURL:       getEnvOrDefault("DEST_RPC_URL", "http://localhost:8545"),
		PrivateKey:       getEnvOrDefault("PRIVATE_KEY", "0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"),
		WormholeContract: getEnvOrDefault("WORMHOLE_CONTRACT", "0x1b35884f8ba9371419d00ae228da9ff839edfe8fe6a804fdfcd430e0dc7e40db"),
		TargetContract:   getEnvOrDefault("TARGET_CONTRACT", "0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"),
		EmitterAddress:   getEnvOrDefault("EMITTER_ADDRESS", "1b35884f8ba9371419d00ae228da9ff839edfe8fe6a804fdfcd430e0dc7e40db"),
	}
}

// VAAData encapsulates a VAA and its metadata
type VAAData struct {
	VAA        *vaaLib.VAA
	RawBytes   []byte
	ChainID    uint16
	EmitterHex string
	Sequence   uint64
}

// SpyClient handles connections to the Wormhole spy service
type SpyClient struct {
	conn   *grpc.ClientConn
	client spyv1.SpyRPCServiceClient
}

// NewSpyClient creates a new client for the Wormhole spy service
func NewSpyClient(endpoint string) (*SpyClient, error) {
	log.Printf("[SpyClient] Connecting to spy service at %s...", endpoint)
	conn, err := grpc.Dial(endpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to spy: %v", err)
	}
	log.Println("[SpyClient] Connected successfully.")
	return &SpyClient{conn: conn, client: spyv1.NewSpyRPCServiceClient(conn)}, nil
}

// Close closes the connection to the spy service
func (c *SpyClient) Close() {
	if c.conn != nil {
		log.Println("[SpyClient] Closing connection.")
		c.conn.Close()
	}
}

// SubscribeSignedVAA subscribes to all signed VAAs with retry logic
func (c *SpyClient) SubscribeSignedVAA(ctx context.Context) (spyv1.SpyRPCService_SubscribeSignedVAAClient, error) {
	const maxRetries = 5
	const retryDelay = 2 * time.Second

	log.Println("[SpyClient] Subscribing to signed VAAs...")

	var stream spyv1.SpyRPCService_SubscribeSignedVAAClient
	var err error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		stream, err = c.client.SubscribeSignedVAA(ctx, &spyv1.SubscribeSignedVAARequest{})
		if err == nil {
			return stream, nil
		}

		if attempt < maxRetries {
			log.Printf("[SpyClient] Subscribe attempt %d failed: %v. Retrying in %v...",
				attempt, err, retryDelay)
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
}

// NewEVMClient creates a new client for EVM-compatible blockchains
func NewEVMClient(rpcURL, privateKeyHex string) (*EVMClient, error) {
	log.Printf("[EVMClient] Connecting to %s...", rpcURL)
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to EVM node: %w", err)
	}

	// Parse private key
	privateKey, err := crypto.HexToECDSA(strings.TrimPrefix(privateKeyHex, "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	// Derive public address
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	log.Printf("[EVMClient] Connected with address: %s", address.Hex())

	return &EVMClient{
		client:     client,
		privateKey: privateKey,
		address:    address,
	}, nil
}

// GetAddress returns the public address for this client
func (c *EVMClient) GetAddress() common.Address {
	return c.address
}

// parseAndVerifyVM verifies a VAA on an EVM chain
func (c *EVMClient) parseAndVerifyVM(ctx context.Context, targetContract string, vaaBytes []byte) (bool, error) {
	const abiJSON = `[{
        "inputs": [{"internalType": "bytes", "name": "encodedVM", "type": "bytes"}],
        "name": "parseAndVerifyVM",
        "outputs": [
            {"internalType": "tuple", "name": "", "type": "tuple"},
            {"internalType": "bool", "name": "", "type": "bool"},
            {"internalType": "string", "name": "", "type": "string"}
        ],
        "stateMutability": "view",
        "type": "function"
    }]`

	parsedABI, err := abi.JSON(strings.NewReader(abiJSON))
	if err != nil {
		return false, fmt.Errorf("ABI parse error: %w", err)
	}

	data, err := parsedABI.Pack("parseAndVerifyVM", vaaBytes)
	if err != nil {
		return false, fmt.Errorf("ABI pack error: %w", err)
	}

	targetAddr := common.HexToAddress(targetContract)
	msg := ethereum.CallMsg{
		To:   &targetAddr,
		Data: data,
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := c.client.CallContract(ctxWithTimeout, msg, nil)
	if err != nil {
		return false, fmt.Errorf("contract call failed: %w", err)
	}

	// Extract the "valid" boolean from the result
	if len(result) >= 64 {
		return result[63] == 1, nil // Check the last byte of the second 32-byte word
	}

	return false, fmt.Errorf("unexpected result format")
}

// Relayer coordinates processing VAAs from the spy service
type Relayer struct {
	spyClient    *SpyClient
	evmClient    *EVMClient
	config       Config
	vaaProcessor func(*Relayer, *VAAData) error
}

// NewRelayer creates a new relayer instance
func NewRelayer(config Config) (*Relayer, error) {
	log.Println("[Relayer] Initializing...")

	// Connect to the spy service
	spyClient, err := NewSpyClient(config.SpyRPCHost)
	if err != nil {
		return nil, fmt.Errorf("failed to create spy client: %w", err)
	}

	// Connect to the EVM chain
	evmClient, err := NewEVMClient(config.DestRPCURL, config.PrivateKey)
	if err != nil {
		spyClient.Close()
		return nil, fmt.Errorf("failed to create EVM client: %w", err)
	}

	relayer := &Relayer{
		spyClient:    spyClient,
		evmClient:    evmClient,
		config:       config,
		vaaProcessor: config.vaaProcessor,
	}

	// Set default VAA processor
	if relayer.vaaProcessor == nil {
		relayer.vaaProcessor = defaultVAAProcessor
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
	log.Printf("[Relayer] Starting with address: %s", r.evmClient.GetAddress().Hex())
	log.Printf("[Relayer] Filtering for VAAs from chain %d to chain %d", r.config.SourceChainID, r.config.DestChainID)
	log.Printf("[Relayer] Monitoring emitter address: %s", r.config.EmitterAddress)

	// Create a wait group to track goroutines
	var wg sync.WaitGroup

	// Subscribe to VAAs
	stream, err := r.spyClient.SubscribeSignedVAA(ctx)
	if err != nil {
		return fmt.Errorf("subscribe to VAA stream: %v", err)
	}

	log.Println("[Relayer] Listening for VAAs...")

	// Create a separate context for graceful shutdown
	processingCtx, cancelProcessing := context.WithCancel(context.Background())
	defer cancelProcessing()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Relayer] Shutting down relayer.")
			// Cancel all processing
			cancelProcessing()
			// Wait for all processing goroutines to complete
			log.Println("[Relayer] Waiting for all VAA processing to complete...")
			wg.Wait()
			log.Println("[Relayer] All VAA processing completed, shutdown complete.")
			return nil
		default:
			// Receive the next VAA
			resp, err := stream.Recv()
			if err != nil {
				log.Printf("[Relayer] Stream error: %v. Retrying in 5s...", err)
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

// processVAA processes a received VAA
func (r *Relayer) processVAA(ctx context.Context, vaaBytes []byte) {
	log.Printf("[Relayer] Received VAA (%d bytes)", len(vaaBytes))

	// Parse the VAA
	wormholeVAA, err := vaaLib.Unmarshal(vaaBytes)
	if err != nil {
		log.Printf("[Relayer] Failed to parse VAA: %v", err)
		return
	}

	// Create VAA data
	vaaData := &VAAData{
		VAA:        wormholeVAA,
		RawBytes:   vaaBytes,
		ChainID:    uint16(wormholeVAA.EmitterChain),
		EmitterHex: hex.EncodeToString(wormholeVAA.EmitterAddress[:]),
		Sequence:   wormholeVAA.Sequence,
	}

	log.Printf("[Relayer] VAA info — Chain: %d | Seq: %d | Emitter: %s",
		vaaData.ChainID, vaaData.Sequence, vaaData.EmitterHex)

	// Process the VAA with the configured processor
	if err := r.vaaProcessor(r, vaaData); err != nil {
		log.Printf("[Relayer] Error processing VAA: %v", err)
	}
}

// defaultVAAProcessor is the default VAA processing logic
func defaultVAAProcessor(r *Relayer, vaaData *VAAData) error {
	// Check if this is a VAA from the source chain (chain ID 52)
	if vaaData.ChainID == r.config.SourceChainID {
		log.Printf("[Processor] Processing VAA %d from source chain %d",
			vaaData.Sequence, vaaData.ChainID)

		// The original verification logic for source chain
		log.Println("[Processor] Verifying VAA...")
		isValid, err := r.evmClient.parseAndVerifyVM(context.Background(), r.config.TargetContract, vaaData.RawBytes)
		if err != nil {
			return fmt.Errorf("Verification failed: %w", err)
		}

		if isValid {
			log.Printf("[Processor] ✅ VAA with sequence %d is valid", vaaData.Sequence)
		} else {
			log.Printf("[Processor] ❌ VAA with sequence %d is invalid", vaaData.Sequence)
		}

		return nil
	}

	// Check if this is a VAA for the destination chain (chain ID 52)
	if vaaData.ChainID == r.config.DestChainID && r.config.DestChainID == 52 {
		log.Println("[Processor] Destination chain logic not yet implemented...")
		return nil
	}

	// If neither source nor destination match our criteria, skip this VAA
	log.Printf("[Processor] ⚠️ Skipping VAA %d from chain %d (not configured for processing)",
		vaaData.Sequence, vaaData.ChainID)
	return nil
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
	var result int
	_, err := fmt.Sscanf(val, "%d", &result)
	if err != nil {
		log.Printf("Warning: Invalid value for %s, using default", key)
		return defaultValue
	}
	return result
}

func main() {
	log.Println("[Main] Starting Wormhole relayer...")

	// Load configuration from environment
	config := NewConfigFromEnv()

	// Create relayer
	relayer, err := NewRelayer(config)
	if err != nil {
		log.Fatalf("[Main] Failed to initialize relayer: %v", err)
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
		log.Println("[Main] Received shutdown signal...")
		cancel()
	}()

	// Start the relayer
	if err := relayer.Start(ctx); err != nil {
		log.Fatalf("[Main] Relayer stopped with error: %v", err)
	}
}
