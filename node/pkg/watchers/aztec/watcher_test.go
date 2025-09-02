package aztec

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// MockBlockFetcher implements the BlockFetcher interface for testing
type MockBlockFetcher struct {
	mock.Mock
}

func (m *MockBlockFetcher) FetchPublicLogs(ctx context.Context, fromBlock, toBlock int) ([]ExtendedPublicLog, error) {
	args := m.Called(ctx, fromBlock, toBlock)
	result, ok := args.Get(0).([]ExtendedPublicLog)
	if !ok {
		return nil, args.Error(1)
	}
	return result, args.Error(1)
}

func (m *MockBlockFetcher) FetchBlock(ctx context.Context, blockNumber int) (BlockInfo, error) {
	args := m.Called(ctx, blockNumber)
	result, ok := args.Get(0).(BlockInfo)
	if !ok {
		return BlockInfo{}, args.Error(1)
	}
	return result, args.Error(1)
}

// MockL1Verifier implements the L1Verifier interface for testing
type MockL1Verifier struct {
	mock.Mock
}

func (m *MockL1Verifier) GetFinalizedBlock(ctx context.Context) (*FinalizedBlock, error) {
	args := m.Called(ctx)
	result, ok := args.Get(0).(*FinalizedBlock)
	if !ok {
		return nil, args.Error(1)
	}
	return result, args.Error(1)
}

func (m *MockL1Verifier) IsBlockFinalized(ctx context.Context, blockNumber int) (bool, error) {
	args := m.Called(ctx, blockNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockL1Verifier) GetLatestFinalizedBlockNumber() uint64 {
	args := m.Called()
	result, ok := args.Get(0).(uint64)
	if !ok {
		return 0
	}
	return result
}

// MockObservationManager implements the ObservationManager interface for testing
type MockObservationManager struct {
	mock.Mock
}

func (m *MockObservationManager) IncrementMessagesConfirmed() {
	m.Called()
}

// setupMockAztecServer creates a mock HTTP server for testing
func setupMockAztecServer(_ *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Failed to read body", http.StatusBadRequest)
			return
		}

		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		switch req.Method {
		case "node_getPublicLogs":
			if _, err := w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "logs": [
                        {
                            "id": {"blockNumber": 5, "txIndex": 0, "logIndex": 0},
                            "log": {
                                "contractAddress": "0xContract",
                                "fields": ["0x123"]
                            }
                        }
                    ],
                    "maxLogsHit": false
                }
            }`)); err != nil {
				http.Error(w, "Write error", http.StatusInternalServerError)
			}

		case "node_getBlock":
			if _, err := w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "archive": {"root": "0xarchive", "nextAvailableLeafIndex": 1},
                    "header": {
                        "lastArchive": {"root": "0xparent", "nextAvailableLeafIndex": 0},
                        "globalVariables": {
                            "blockNumber": "0x5",
                            "timestamp": "0x61a91c40"
                        }
                    },
                    "body": {
                        "txEffects": [
                            {"txHash": "0x0123456789abcdef", "revertCode": 0}
                        ]
                    }
                }
            }`)); err != nil {
				http.Error(w, "Write error", http.StatusInternalServerError)
			}

		case "node_getL2Tips":
			if _, err := w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "latest": {"number": 10, "hash": "0x123"},
                    "proven": {"number": 8, "hash": "0x456"},
                    "finalized": {"number": 5, "hash": "0x789"}
                }
            }`)); err != nil {
				http.Error(w, "Write error", http.StatusInternalServerError)
			}

		default:
			w.WriteHeader(http.StatusBadRequest)
			if _, err := w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`)); err != nil {
				http.Error(w, "Write error", http.StatusInternalServerError)
			}
		}
	}))
}

// TestCreateObservationID tests the observation ID creation
func TestCreateObservationID(t *testing.T) {
	id := CreateObservationID("0x123", 456, 789)
	assert.Equal(t, "0x123-456-789", id)
}

// TestProcessLogParameters tests parameter parsing
func TestProcessLogParameters(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := DefaultConfig(vaa.ChainID(1), "test", "http://localhost:8545", "0xContract")
	watcher := &Watcher{
		config: config,
		logger: logger,
	}

	// Valid parameters
	t.Run("valid parameters", func(t *testing.T) {
		logEntries := []string{
			"0000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender (32 bytes)
			"0000000000000000000000000000000000000000000000000000000000000123", // Sequence (291 in hex)
			"0000000000000000000000000000000000000000000000000000000000000001", // Nonce (1)
			"0000000000000000000000000000000000000000000000000000000000000001", // Consistency level (1)
		}

		params, err := watcher.parseLogParameters(logEntries)
		assert.NoError(t, err)
		assert.Equal(t, uint64(291), params.Sequence)
		assert.Equal(t, uint32(1), params.Nonce)
		assert.Equal(t, uint8(1), params.ConsistencyLevel)
	})

	// Invalid parameters (too few entries)
	t.Run("invalid parameters", func(t *testing.T) {
		invalidEntries := []string{
			"0000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender (32 bytes)
			"0000000000000000000000000000000000000000000000000000000000000123", // Sequence (291)
		}

		_, err := watcher.parseLogParameters(invalidEntries)
		assert.Error(t, err)
	})
}

// TestProcessLog tests the log processing function
func TestProcessLog(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockBlockFetcher := new(MockBlockFetcher)
	mockL1Verifier := new(MockL1Verifier)
	mockObservationManager := new(MockObservationManager)
	msgC := make(chan *common.MessagePublication, 10)
	config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")
	contractAddress := config.ContractAddress

	watcher := &Watcher{
		config:             config,
		blockFetcher:       mockBlockFetcher,
		l1Verifier:         mockL1Verifier,
		observationManager: mockObservationManager,
		msgC:               msgC,
		logger:             logger,
	}

	// Test with valid log
	log := ExtendedPublicLog{
		ID: LogId{
			BlockNumber: 100,
			TxIndex:     0,
			LogIndex:    0,
		},
		Log: PublicLog{
			ContractAddress: contractAddress,
			Fields: []string{
				"0000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender (32 bytes)
				"0000000000000000000000000000000000000000000000000000000000000123", // Sequence (291)
				"0000000000000000000000000000000000000000000000000000000000000001", // Nonce (1)
				"0000000000000000000000000000000000000000000000000000000000000001", // Consistency level (1)
				"0000000000000000000000000000000000000000000000000000000061a91c40", // Timestamp
				"000000000000000000000000742d35Cc6634C0532925a3b8D50C6d111111111",  // Arbitrum address
				"00000000000000000000000000000000000000000000000000000000000a4b1",  // Arbitrum chain ID
				"000000000000000000000000000000000000000000000000000000000f4240",   // Amount
				"4a61636b000000000000000000000000000000000000000000000000000000",   // "Jack" padded
			},
		},
	}

	blockInfo := BlockInfo{
		TxHash:    "0x0123456789abcdef",
		Timestamp: safeUnixToUint64(time.Now().Unix()),
	}

	mockObservationManager.On("IncrementMessagesConfirmed").Return()

	// Process the log
	err := watcher.processLog(context.Background(), log, blockInfo)

	// Verify expectations
	assert.NoError(t, err)
	mockObservationManager.AssertExpectations(t)
	assert.Equal(t, 1, len(msgC), "Should have published 1 message")

	// Check the message
	msg := <-msgC
	assert.Equal(t, uint64(291), msg.Sequence) // 0x123 = 291
	assert.Equal(t, uint32(1), msg.Nonce)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.True(t, len(msg.Payload) > 32) // Should have txID + payload data
}

// TestPublishObservation tests the observation publishing function
func TestPublishObservation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	mockObservationManager := new(MockObservationManager)
	msgC := make(chan *common.MessagePublication, 10)
	config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")

	watcher := &Watcher{
		config:             config,
		observationManager: mockObservationManager,
		msgC:               msgC,
		logger:             logger,
	}

	params := LogParameters{
		SenderAddress:    vaa.Address{1, 2, 3, 4, 5},
		Sequence:         123,
		Nonce:            456,
		ConsistencyLevel: 1,
	}

	payload := []byte{1, 2, 3, 4, 5}
	blockInfo := BlockInfo{
		TxHash:    "0x0123456789abcdef",
		Timestamp: safeUnixToUint64(time.Now().Unix()),
	}
	observationID := "test-observation"

	mockObservationManager.On("IncrementMessagesConfirmed").Return()
	err := watcher.publishObservation(context.Background(), params, payload, blockInfo, observationID)

	assert.NoError(t, err)
	mockObservationManager.AssertExpectations(t)
	assert.Equal(t, 1, len(msgC))

	msg := <-msgC
	assert.Equal(t, params.Sequence, msg.Sequence)
	assert.Equal(t, params.Nonce, msg.Nonce)
	assert.Equal(t, params.ConsistencyLevel, msg.ConsistencyLevel)
	assert.Equal(t, payload, msg.Payload)

	expectedTxID, _ := hex.DecodeString("0123456789abcdef")
	assert.Equal(t, expectedTxID, msg.TxID)
}

// TestNewAztecFinalityVerifier tests the creation of the finality verifier
func TestNewAztecFinalityVerifier(t *testing.T) {
	// Create a verifier with a test URL
	rpcURL := "http://localhost:9999"
	logger := zap.NewNop()

	verifier, err := NewAztecFinalityVerifier(context.Background(), rpcURL, logger)

	// The function should return a valid verifier
	assert.NoError(t, err)
	assert.NotNil(t, verifier)

	// Check the type of the returned verifier
	aztecVerifier, ok := verifier.(*aztecFinalityVerifier)
	assert.True(t, ok, "Should return an aztecFinalityVerifier instance")
	assert.Equal(t, 30*time.Second, aztecVerifier.finalizedBlockCacheTTL)
}

// TestErrorTypes tests the error types
func TestErrorTypes(t *testing.T) {
	// Test RPC error
	rpcErr := ErrRPCError{
		Method: "test_method",
		Code:   -32000,
		Msg:    "test error",
	}
	assert.Equal(t, "RPC error calling test_method: test error", rpcErr.Error())

	// Test parsing failed error
	parsingErr := ErrParsingFailed{
		What: "test data",
		Err:  fmt.Errorf("test error"),
	}
	assert.Equal(t, "failed parsing test data: test error", parsingErr.Error())
}

// TestAztecFinalityVerifier_Implementation tests the finality verifier implementation
func TestAztecFinalityVerifier_Implementation(t *testing.T) {
	// Setup mock server
	server := setupMockAztecServer(t)
	defer server.Close()

	// Create the verifier directly with the test server
	client, err := rpc.DialContext(context.Background(), server.URL)
	require.NoError(t, err)

	// Create verifier manually
	verifier := &aztecFinalityVerifier{
		rpcClient:              client,
		logger:                 zap.NewNop(),
		finalizedBlockCacheTTL: time.Second,
	}

	// Test GetFinalizedBlock
	t.Run("GetFinalizedBlock", func(t *testing.T) {
		block, err := verifier.GetFinalizedBlock(context.Background())
		require.NoError(t, err)
		require.Equal(t, 5, block.Number)     // Should be finalized block number
		require.Equal(t, "0x789", block.Hash) // Should be finalized hash
	})

	// Test IsBlockFinalized
	t.Run("IsBlockFinalized", func(t *testing.T) {
		finalized, err := verifier.IsBlockFinalized(context.Background(), 3)
		require.NoError(t, err)
		require.True(t, finalized)

		finalized, err = verifier.IsBlockFinalized(context.Background(), 10)
		require.NoError(t, err)
		require.False(t, finalized)
	})

	// Test GetLatestFinalizedBlockNumber
	t.Run("GetLatestFinalizedBlockNumber", func(t *testing.T) {
		number := verifier.GetLatestFinalizedBlockNumber()
		require.Equal(t, uint64(5), number) // Should match finalized block number
	})
}

// TestHelperFunctions tests various helper functions
func TestHelperFunctions(t *testing.T) {
	// Test GetJSONRPCError
	t.Run("GetJSONRPCError", func(t *testing.T) {
		// Test with error response
		jsonResp := []byte(`{"error":{"code":-32000,"message":"test error"}}`)
		hasError, rpcErr := GetJSONRPCError(jsonResp)
		require.True(t, hasError)
		require.Equal(t, -32000, rpcErr.Code)
		require.Equal(t, "test error", rpcErr.Msg)

		// Test with success response
		jsonResp = []byte(`{"result":"success"}`)
		hasError, _ = GetJSONRPCError(jsonResp)
		require.False(t, hasError)
	})
}

// TestWatcherConfig tests the WatcherConfig implementation
func TestWatcherConfig(t *testing.T) {
	// Create config instance
	config := &WatcherConfig{
		NetworkID: "aztec-test",
		ChainID:   vaa.ChainID(52),
		Rpc:       "http://test-url",
		Contract:  "0xContract",
	}

	// Test GetChainID
	t.Run("GetChainID", func(t *testing.T) {
		chainID := config.GetChainID()
		require.Equal(t, vaa.ChainID(52), chainID)
	})

	// Test GetNetworkID
	t.Run("GetNetworkID", func(t *testing.T) {
		networkID := config.GetNetworkID()
		require.Equal(t, watchers.NetworkID("aztec-test"), networkID)
	})
}

// TestDefaultConfig tests the default configuration
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")
	require.Equal(t, vaa.ChainID(52), config.ChainID)
	require.Equal(t, "test", config.NetworkID)
	require.Equal(t, "http://localhost:8545", config.RpcURL)
	require.Equal(t, "0xContract", config.ContractAddress)
	require.Equal(t, 1, config.StartBlock)
	require.Equal(t, 13, config.PayloadInitialCap)
}

// Helper function to safely convert Unix timestamp
func safeUnixToUint64(unixTime int64) uint64 {
	if unixTime < 0 {
		return 0
	}
	return uint64(unixTime)
}
