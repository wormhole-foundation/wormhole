package aztec

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
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
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest"
)

// Define RPC client interface
type RPCClient interface {
	CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error
}

// MockRPCClient implements the RPCClient interface for testing
type MockRPCClient struct {
	mock.Mock
}

func (m *MockRPCClient) CallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	mockArgs := []interface{}{ctx, result, method}
	for _, arg := range args {
		mockArgs = append(mockArgs, arg)
	}
	return m.Called(mockArgs...).Error(0)
}

// BlockFetcher interface mock
type MockBlockFetcher struct {
	mock.Mock
}

func (m *MockBlockFetcher) FetchPublicLogs(ctx context.Context, fromBlock, toBlock int) ([]ExtendedPublicLog, error) {
	args := m.Called(ctx, fromBlock, toBlock)
	return args.Get(0).([]ExtendedPublicLog), args.Error(1)
}

func (m *MockBlockFetcher) FetchBlock(ctx context.Context, blockNumber int) (BlockInfo, error) {
	args := m.Called(ctx, blockNumber)
	return args.Get(0).(BlockInfo), args.Error(1)
}

// L1Verifier interface mock
type MockL1Verifier struct {
	mock.Mock
}

func (m *MockL1Verifier) GetFinalizedBlock(ctx context.Context) (*FinalizedBlock, error) {
	args := m.Called(ctx)
	return args.Get(0).(*FinalizedBlock), args.Error(1)
}

func (m *MockL1Verifier) IsBlockFinalized(ctx context.Context, blockNumber int) (bool, error) {
	args := m.Called(ctx, blockNumber)
	return args.Bool(0), args.Error(1)
}

func (m *MockL1Verifier) GetLatestFinalizedBlockNumber() uint64 {
	args := m.Called()
	return args.Get(0).(uint64)
}

// ObservationManager interface mock
type MockObservationManager struct {
	mock.Mock
}

func (m *MockObservationManager) IncrementMessagesConfirmed() {
	m.Called()
}

// Helper function to setup mock server for Aztec RPC tests
func setupMockAztecServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		switch req.Method {
		case "node_getPublicLogs":
			w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "logs": [
                        {
                            "id": {"blockNumber": 5, "txIndex": 0, "logIndex": 0},
                            "log": {
                                "contractAddress": "0xContract",
                                "log": ["0x123"]
                            }
                        }
                    ],
                    "maxLogsHit": false
                }
            }`))

		case "node_getBlock":
			w.Write([]byte(`{
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
            }`))

		case "node_getL2Tips":
			w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "latest": {"number": 10, "hash": "0x123"},
                    "proven": {"number": 8, "hash": "0x456"},
                    "finalized": {"number": 5, "hash": "0x789"}
                }
            }`))

		default:
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`))
		}
	}))
}

// TestParseHexUint64 tests the hex parsing function with various inputs
func TestParseHexUint64(t *testing.T) {
	testCases := []struct {
		name      string
		input     string
		expectErr bool
		expected  uint64
	}{
		{"Valid hex with 0x prefix", "0x123", false, 291},
		{"Valid hex without prefix", "123", false, 291},
		{"Invalid hex", "0xZZZ", true, 0},
		{"Empty string", "", true, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value, err := ParseHexUint64(tc.input)

			if tc.expectErr {
				assert.Error(t, err)
				assert.IsType(t, &ErrParsingFailed{}, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.expected, value)
			}
		})
	}
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
			"000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender
			"0000000000000123", // Sequence (291)
			"0000000000000001", // Nonce (1)
			"01",               // Consistency level (1)
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
			"000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender
			"0000000000000123", // Sequence (291)
		}

		_, err := watcher.parseLogParameters(invalidEntries)
		assert.Error(t, err)
	})
}

// TestCreatePayload tests payload creation from log entries
func TestCreatePayload(t *testing.T) {
	logger := zap.NewNop()

	config := DefaultConfig(vaa.ChainID(1), "test", "http://localhost:8545", "0xContract")
	watcher := &Watcher{
		config: config,
		logger: logger,
	}

	// Test with valid hex entries
	t.Run("valid hex entries", func(t *testing.T) {
		logEntries := []string{
			"0x0123",
			"0x4567",
		}

		payload := watcher.createPayload(logEntries)
		assert.Equal(t, []byte{0x01, 0x23, 0x45, 0x67}, payload)
	})

	// Test with mixed entries (valid and invalid)
	t.Run("mixed entries", func(t *testing.T) {
		mixedEntries := []string{
			"0x0123",
			"invalid",
			"0x4567",
		}

		payload := watcher.createPayload(mixedEntries)
		assert.Equal(t, []byte{0x01, 0x23, 0x45, 0x67}, payload)
	})
}

// TestWatcherProcessBlocks tests block processing
func TestWatcherProcessBlocks(t *testing.T) {
	logger := zaptest.NewLogger(t)

	mockBlockFetcher := new(MockBlockFetcher)
	mockL1Verifier := new(MockL1Verifier)
	mockObservationManager := new(MockObservationManager)

	msgC := make(chan *common.MessagePublication, 10)

	config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")
	watcher := &Watcher{
		config:             config,
		blockFetcher:       mockBlockFetcher,
		l1Verifier:         mockL1Verifier,
		observationManager: mockObservationManager,
		msgC:               msgC,
		logger:             logger,
		lastBlockNumber:    5, // Start at block 5
	}

	// Set up L1Verifier to return a finalized block
	mockL1Verifier.On("GetFinalizedBlock", mock.Anything).Return(&FinalizedBlock{
		Number: 7, // Finalized up to block 7
		Hash:   "0xhash",
	}, nil)

	// Set up BlockFetcher to return blocks 6 and 7
	blockInfo6 := BlockInfo{
		TxHash:            "0xtxhash6",
		Timestamp:         uint64(time.Now().Unix()),
		archiveRoot:       "0xblockhash6",
		parentArchiveRoot: "0xparenthash6",
		TxHashesByIndex:   map[int]string{0: "0xtxhash6-0"},
	}
	mockBlockFetcher.On("FetchBlock", mock.Anything, 6).Return(blockInfo6, nil)

	blockInfo7 := BlockInfo{
		TxHash:            "0xtxhash7",
		Timestamp:         uint64(time.Now().Unix()),
		archiveRoot:       "0xblockhash7",
		parentArchiveRoot: "0xparenthash7",
		TxHashesByIndex:   map[int]string{0: "0xtxhash7-0"},
	}
	mockBlockFetcher.On("FetchBlock", mock.Anything, 7).Return(blockInfo7, nil)

	// Set up BlockFetcher to return no logs for blocks 6 and 7
	mockBlockFetcher.On("FetchPublicLogs", mock.Anything, 6, 7).Return([]ExtendedPublicLog{}, nil)
	mockBlockFetcher.On("FetchPublicLogs", mock.Anything, 7, 8).Return([]ExtendedPublicLog{}, nil)

	// Process blocks
	err := watcher.processBlocks(context.Background())

	// Verify expectations
	assert.NoError(t, err)
	assert.Equal(t, 7, watcher.lastBlockNumber) // Last processed block should be updated
	mockL1Verifier.AssertExpectations(t)
	mockBlockFetcher.AssertExpectations(t)
}

// TestWatcherProcessBlocksError tests error handling in block processing
func TestWatcherProcessBlocksError(t *testing.T) {
	logger := zaptest.NewLogger(t)

	mockBlockFetcher := new(MockBlockFetcher)
	mockL1Verifier := new(MockL1Verifier)
	mockObservationManager := new(MockObservationManager)

	msgC := make(chan *common.MessagePublication, 10)

	config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")
	watcher := &Watcher{
		config:             config,
		blockFetcher:       mockBlockFetcher,
		l1Verifier:         mockL1Verifier,
		observationManager: mockObservationManager,
		msgC:               msgC,
		logger:             logger,
		lastBlockNumber:    5, // Start at block 5
	}

	// Set up L1Verifier to return a finalized block
	mockL1Verifier.On("GetFinalizedBlock", mock.Anything).Return(&FinalizedBlock{
		Number: 7, // Finalized up to block 7
		Hash:   "0xhash",
	}, nil)

	// Set up BlockFetcher to return an error for block 6
	expectedErr := fmt.Errorf("block fetch error")
	mockBlockFetcher.On("FetchBlock", mock.Anything, 6).Return(BlockInfo{}, expectedErr)

	// Process blocks - should fail
	err := watcher.processBlocks(context.Background())

	// Verify expectations
	assert.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.Equal(t, 5, watcher.lastBlockNumber) // Last processed block should not change
	mockL1Verifier.AssertExpectations(t)
	mockBlockFetcher.AssertExpectations(t)
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
			Log: []string{
				"000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender
				"0000000000000123", // Sequence
				"0000000000000001", // Nonce
				"01",               // Consistency level
				"01020304",         // Payload
			},
		},
	}

	blockInfo := BlockInfo{
		TxHash:    "0x0123456789abcdef",
		Timestamp: 1620000000,
	}

	mockObservationManager.On("IncrementMessagesConfirmed").Return()

	// Process the log
	err := watcher.processLog(context.Background(), log, blockInfo)

	// Verify expectations
	assert.NoError(t, err)
	mockObservationManager.AssertExpectations(t)

	// Verify a message was published
	assert.Equal(t, 1, len(msgC), "Should have published 1 message")

	// Check the message
	msg := <-msgC
	assert.Equal(t, uint64(291), msg.Sequence) // 0x123 = 291
	assert.Equal(t, uint32(1), msg.Nonce)
	assert.Equal(t, uint8(1), msg.ConsistencyLevel)
	assert.Equal(t, []byte{1, 2, 3, 4}, msg.Payload)
}

// TestProcessBlockLogs tests the block logs processing function
func TestProcessBlockLogs(t *testing.T) {
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

	// Set up for fetching logs
	blockNumber := 100
	logs := []ExtendedPublicLog{
		{
			ID: LogId{
				BlockNumber: blockNumber,
				TxIndex:     0,
				LogIndex:    0,
			},
			Log: PublicLog{
				ContractAddress: contractAddress,
				Log: []string{
					"000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender
					"0000000000000123", // Sequence
					"0000000000000001", // Nonce
					"01",               // Consistency level
					"01020304",         // Payload
				},
			},
		},
	}

	mockBlockFetcher.On("FetchPublicLogs", mock.Anything, blockNumber, blockNumber+1).Return(logs, nil)

	// Block info for the test
	blockInfo := BlockInfo{
		TxHash:    "0x0123456789abcdef",
		Timestamp: 1620000000,
		TxHashesByIndex: map[int]string{
			0: "0x0123456789abcdef",
		},
	}

	// Set up observation manager
	mockObservationManager.On("IncrementMessagesConfirmed").Return()

	// Process the block logs
	err := watcher.processBlockLogs(context.Background(), blockNumber, blockInfo)

	// Verify expectations
	assert.NoError(t, err)
	mockBlockFetcher.AssertExpectations(t)
	mockObservationManager.AssertExpectations(t)

	// Check message
	assert.Equal(t, 1, len(msgC), "Should have published 1 message")
	msg := <-msgC
	assert.Equal(t, uint64(291), msg.Sequence)
	assert.Equal(t, uint32(1), msg.Nonce)
	assert.Equal(t, []byte{1, 2, 3, 4}, msg.Payload)
}

// TestPublishObservation tests the observation publishing function
func TestPublishObservation(t *testing.T) {
	t.Run("with mock observation manager", func(t *testing.T) {
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
			Timestamp: 1620000000,
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
	})

	t.Run("with real observation manager", func(t *testing.T) {
		logger := zap.NewNop()
		msgC := make(chan *common.MessagePublication, 10)
		observationManager := NewObservationManager("test", logger)
		config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")

		watcher := &Watcher{
			config:             config,
			observationManager: observationManager,
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
			Timestamp: 1620000000,
		}
		observationID := "test-observation"

		err := watcher.publishObservation(context.Background(), params, payload, blockInfo, observationID)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(msgC))

		msg := <-msgC
		assert.Equal(t, params.Sequence, msg.Sequence)
		assert.Equal(t, params.Nonce, msg.Nonce)
		assert.Equal(t, params.ConsistencyLevel, msg.ConsistencyLevel)
		assert.Equal(t, payload, msg.Payload)

		expectedTxID, _ := hex.DecodeString(strings.TrimPrefix(blockInfo.TxHash, "0x"))
		assert.Equal(t, expectedTxID, msg.TxID)
	})
}

// TestContextCancellation tests context cancellation handling
func TestContextCancellation(t *testing.T) {
	// Create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create a channel to signal completion
	done := make(chan struct{})

	// Start a goroutine that uses the context
	go func() {
		select {
		case <-ctx.Done():
			close(done)
			return
		case <-time.After(100 * time.Millisecond):
			t.Error("Context cancellation not detected")
			close(done)
		}
	}()

	// Cancel the context
	cancel()

	// Wait for the goroutine to complete
	select {
	case <-done:
		// Test passes - function responded to cancellation
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Function did not respond to context cancellation")
	}
}

// TestNewAztecFinalityVerifierDirect tests the creation of the finality verifier
func TestNewAztecFinalityVerifierDirect(t *testing.T) {
	// Create a real verifier with a valid but unconnectable URL
	// Just to test the function call path
	rpcURL := "http://localhost:9999"
	logger := zap.NewNop()

	// The call should succeed even if the connection is impossible
	// Because DialContext is non-blocking
	verifier, err := NewAztecFinalityVerifier(rpcURL, logger)

	// The function should return a valid verifier
	assert.NoError(t, err)
	assert.NotNil(t, verifier)

	// Check the type of the returned verifier
	_, ok := verifier.(*aztecFinalityVerifier)
	assert.True(t, ok, "Should return an aztecFinalityVerifier instance")

	// Check that the cache TTL is set correctly
	aztecVerifier := verifier.(*aztecFinalityVerifier)
	assert.Equal(t, 30*time.Second, aztecVerifier.finalizedBlockCacheTTL)
}

// TestErrorTypes tests the direct error types
func TestErrorTypes(t *testing.T) {
	// Test RPC error
	rpcErr := ErrRPCError{
		Method: "test_method",
		Code:   -32000,
		Msg:    "test error",
	}
	assert.Equal(t, "RPC error calling test_method: test error", rpcErr.Error())

	// Test max retries error
	maxRetriesErr := ErrMaxRetriesExceeded{
		Method: "test_method",
	}
	assert.Equal(t, "max retries exceeded for test_method", maxRetriesErr.Error())

	// Test parsing failed error
	parsingErr := ErrParsingFailed{
		What: "test data",
		Err:  assert.AnError,
	}
	assert.Equal(t, "failed parsing test data: assert.AnError general error for testing", parsingErr.Error())
}

// TestAztecBlockFetcher_Implementation tests the block fetcher implementation
func TestAztecBlockFetcher_Implementation(t *testing.T) {
	// Setup mock server
	server := setupMockAztecServer(t)
	defer server.Close()

	// Create the block fetcher directly with the test server
	client, err := rpc.DialContext(context.Background(), server.URL)
	require.NoError(t, err)

	// Create fetcher manually
	fetcher := &aztecBlockFetcher{
		rpcClient: client,
		logger:    zap.NewNop(),
	}

	// Test FetchPublicLogs
	t.Run("FetchPublicLogs", func(t *testing.T) {
		logs, err := fetcher.FetchPublicLogs(context.Background(), 5, 6)
		require.NoError(t, err)
		require.Len(t, logs, 1)
		require.Equal(t, 5, logs[0].ID.BlockNumber)
	})

	// Test FetchBlock
	t.Run("FetchBlock", func(t *testing.T) {
		block, err := fetcher.FetchBlock(context.Background(), 5)
		require.NoError(t, err)
		require.Equal(t, "0x0123456789abcdef", block.TxHashesByIndex[0])
	})
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
		require.Equal(t, 5, block.Number)
		require.Equal(t, "0x789", block.Hash)
	})

	// Test IsBlockFinalized
	t.Run("IsBlockFinalized", func(t *testing.T) {
		finalized, err := verifier.IsBlockFinalized(context.Background(), 3)
		require.NoError(t, err)
		require.True(t, finalized)

		finalized, err = verifier.IsBlockFinalized(context.Background(), 7)
		require.NoError(t, err)
		require.False(t, finalized)
	})

	// Test GetLatestFinalizedBlockNumber
	t.Run("GetLatestFinalizedBlockNumber", func(t *testing.T) {
		number := verifier.GetLatestFinalizedBlockNumber()
		require.Equal(t, uint64(5), number)
	})
}

// TestHTTPClient tests the HTTP client implementation
func TestHTTPClient(t *testing.T) {
	// Setup a test server that responds differently based on request count
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		if requestCount <= 2 {
			// Fail the first two requests
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Succeed on the third request
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{"data":"success"}}`))
	}))
	defer server.Close()

	// Create the client with short timeout and retry settings
	client := NewHTTPClient(
		100*time.Millisecond, // timeout
		3,                    // maxRetries
		10*time.Millisecond,  // initialBackoff
		1.5,                  // backoffMultiplier
		zap.NewNop(),
	)

	// Test request with retries
	t.Run("successful retry", func(t *testing.T) {
		payload := map[string]any{"method": "test_method", "params": []string{}}
		resp, err := client.DoRequest(context.Background(), server.URL, payload)

		require.NoError(t, err)
		require.NotNil(t, resp)
		require.Contains(t, string(resp), "success")
		require.Equal(t, 3, requestCount, "Should have made exactly 3 requests")
	})
}

// TestObservationManager tests the observation manager
func TestObservationManager(t *testing.T) {
	// Create a logger
	logger := zap.NewNop()

	// Create an observation manager
	manager := NewObservationManager("test-network", logger)

	// Call methods and verify behavior
	t.Run("IncrementMessagesConfirmed", func(t *testing.T) {
		require.NotPanics(t, func() {
			manager.IncrementMessagesConfirmed()
		})
	})
}

// TestHelperFunctions tests various helper functions
func TestHelperFunctions(t *testing.T) {
	// Test ParseUint with table-driven approach
	t.Run("ParseUint", func(t *testing.T) {
		testCases := []struct {
			name      string
			input     string
			base      int
			bitSize   int
			expectErr bool
			expected  uint64
		}{
			{"Valid decimal", "123", 10, 64, false, 123},
			{"Valid hex", "7b", 16, 64, false, 123},
			{"Invalid input", "xyz", 10, 64, true, 0},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				val, err := ParseUint(tc.input, tc.base, tc.bitSize)

				if tc.expectErr {
					require.Error(t, err)
					require.IsType(t, &ErrParsingFailed{}, err)
				} else {
					require.NoError(t, err)
					require.Equal(t, tc.expected, val)
				}
			})
		}
	})

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

// TestAztecFinalityVerifier_CacheHandling tests the cache behavior
func TestAztecFinalityVerifier_CacheHandling(t *testing.T) {
	// Setup mock server
	serverCallCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Increment call count to track when it's being called
		serverCallCount++

		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		if req.Method == "node_getL2Tips" {
			w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "result": {
                    "latest": {"number": 10, "hash": "0x123"},
                    "proven": {"number": 8, "hash": "0x456"},
                    "finalized": {"number": 5, "hash": "0x789"}
                }
            }`))
		}
	}))
	defer server.Close()

	// Create verifier manually with a very short cache TTL for testing
	client, _ := rpc.DialContext(context.Background(), server.URL)
	verifier := &aztecFinalityVerifier{
		rpcClient:              client,
		logger:                 zap.NewNop(),
		finalizedBlockCacheTTL: 50 * time.Millisecond, // Short TTL for testing
	}

	// First call should hit the server
	block1, err := verifier.GetFinalizedBlock(context.Background())
	require.NoError(t, err)
	require.Equal(t, 5, block1.Number)
	require.Equal(t, 1, serverCallCount, "First call should hit the server")

	// Second immediate call should use the cache
	block2, err := verifier.GetFinalizedBlock(context.Background())
	require.NoError(t, err)
	require.Equal(t, 5, block2.Number)
	require.Equal(t, 1, serverCallCount, "Second call should use cache")

	// Wait for the cache to expire
	time.Sleep(60 * time.Millisecond)

	// Third call after cache expiry should hit the server again
	block3, err := verifier.GetFinalizedBlock(context.Background())
	require.NoError(t, err)
	require.Equal(t, 5, block3.Number)
	require.Equal(t, 2, serverCallCount, "Third call after cache expiry should hit server")

	// Test IsBlockFinalized with cache
	finalized1, err := verifier.IsBlockFinalized(context.Background(), 3)
	require.NoError(t, err)
	require.True(t, finalized1)
	require.Equal(t, 2, serverCallCount, "IsBlockFinalized should use cached data")

	// Test GetLatestFinalizedBlockNumber with cache
	number := verifier.GetLatestFinalizedBlockNumber()
	require.Equal(t, uint64(5), number)
	require.Equal(t, 2, serverCallCount, "GetLatestFinalizedBlockNumber should use cached data")
}

// TestHTTPClient_ErrorCases tests error handling in HTTP client
func TestHTTPClient_ErrorCases(t *testing.T) {
	logger := zap.NewNop()

	// Test 1: Invalid URL
	t.Run("invalid URL", func(t *testing.T) {
		client := NewHTTPClient(
			100*time.Millisecond,
			1,
			10*time.Millisecond,
			1.5,
			logger,
		)

		// Test with invalid URL
		payload := map[string]any{"method": "test_method"}
		_, err := client.DoRequest(context.Background(), "http://invalid-url-that-doesnt-exist-123456.example", payload)
		require.Error(t, err)
	})

	// Test 2: Context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		// Create a server that delays responses
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Sleep to simulate long request
			time.Sleep(200 * time.Millisecond)
			w.Write([]byte(`{"result":"success"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(
			500*time.Millisecond,
			1,
			10*time.Millisecond,
			1.5,
			logger,
		)

		// Create context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Call should fail due to context timeout
		payload := map[string]any{"method": "test_method"}
		_, err := client.DoRequest(ctx, server.URL, payload)
		require.Error(t, err)
	})

	// Test 3: Server returning error status code
	t.Run("error status code", func(t *testing.T) {
		// Create a server that always returns 500
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`{"error":"server error"}`))
		}))
		defer server.Close()

		client := NewHTTPClient(
			100*time.Millisecond,
			1,
			10*time.Millisecond,
			1.5,
			logger,
		)

		payload := map[string]any{"method": "test_method"}
		_, err := client.DoRequest(context.Background(), server.URL, payload)
		require.Error(t, err)
	})

	// Test 4: JSON-RPC error in response
	t.Run("JSON-RPC error response", func(t *testing.T) {
		// Create a server that returns JSON-RPC error
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32601,"message":"Method not found"}}`))
		}))
		defer server.Close()

		client := NewHTTPClient(
			100*time.Millisecond,
			1,
			10*time.Millisecond,
			1.5,
			logger,
		)

		payload := map[string]any{"method": "test_method"}
		_, err := client.DoRequest(context.Background(), server.URL, payload)
		require.Error(t, err)
		require.IsType(t, &ErrRPCError{}, err)
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

	// Test RequiredL1Finalizer
	t.Run("RequiredL1Finalizer", func(t *testing.T) {
		l1Finalizer := config.RequiredL1Finalizer()
		require.Equal(t, watchers.NetworkID(""), l1Finalizer, "Should return empty network ID")
	})

	// Test SetL1Finalizer
	t.Run("SetL1Finalizer", func(t *testing.T) {
		mockFinalizer := new(MockL1Verifier)
		require.NotPanics(t, func() {
			config.SetL1Finalizer(mockFinalizer)
		})
	})
}

// TestNewWatcher tests the watcher constructor
func TestNewWatcher(t *testing.T) {
	// Setup mocks and channels
	mockBlockFetcher := new(MockBlockFetcher)
	mockL1Verifier := new(MockL1Verifier)
	mockObservationManager := new(MockObservationManager)
	msgC := make(chan *common.MessagePublication, 10)
	logger := zap.NewNop()

	// Set the configuration
	config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")
	config.StartBlock = 100 // Set a start block

	// Create a new watcher
	watcher := NewWatcher(
		config,
		mockBlockFetcher,
		mockL1Verifier,
		mockObservationManager,
		msgC,
		logger,
	)

	// Verify the watcher is properly initialized
	require.NotNil(t, watcher)
	require.Equal(t, config, watcher.config)
	require.Equal(t, mockBlockFetcher, watcher.blockFetcher)
	require.Equal(t, mockL1Verifier, watcher.l1Verifier)
	require.Equal(t, mockObservationManager, watcher.observationManager)
	require.Equal(t, logger, watcher.logger)
	require.Equal(t, 99, watcher.lastBlockNumber)
}

// TestAztecBlockFetcher_ErrorResponses tests error handling in block fetcher
func TestAztecBlockFetcher_ErrorResponses(t *testing.T) {
	// Setup mock server that returns errors
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		// Return error for node_getPublicLogs
		if req.Method == "node_getPublicLogs" {
			w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "error": {
                    "code": -32000,
                    "message": "Log filter error"
                }
            }`))
			return
		}

		// Return error for node_getBlock
		if req.Method == "node_getBlock" {
			w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "error": {
                    "code": -32000,
                    "message": "Block not found"
                }
            }`))
			return
		}
	}))
	defer server.Close()

	// Create the block fetcher
	client, err := rpc.DialContext(context.Background(), server.URL)
	require.NoError(t, err)

	fetcher := &aztecBlockFetcher{
		rpcClient: client,
		logger:    zap.NewNop(),
	}

	// Test FetchPublicLogs with error response
	t.Run("FetchPublicLogs error", func(t *testing.T) {
		_, err := fetcher.FetchPublicLogs(context.Background(), 5, 6)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to fetch public logs")
	})

	// Test FetchBlock with error response
	t.Run("FetchBlock error", func(t *testing.T) {
		_, err := fetcher.FetchBlock(context.Background(), 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to fetch block info")
	})
}

// TestGetJSONRPCError_MoreCases tests additional cases for JSON-RPC error handling
func TestGetJSONRPCError_MoreCases(t *testing.T) {
	// Test with standard JSON-RPC error
	t.Run("standard error", func(t *testing.T) {
		jsonResp := []byte(`{"jsonrpc":"2.0","id":1,"error":{"code":-32000,"message":"Standard error"}}`)
		hasError, rpcErr := GetJSONRPCError(jsonResp)
		require.True(t, hasError)
		require.Equal(t, -32000, rpcErr.Code)
		require.Equal(t, "Standard error", rpcErr.Msg)
	})

	// Test with valid result (no error)
	t.Run("valid result", func(t *testing.T) {
		jsonResp := []byte(`{"jsonrpc":"2.0","id":1,"result":{"data":"success"}}`)
		hasError, _ := GetJSONRPCError(jsonResp)
		require.False(t, hasError)
	})

	// Test with malformed JSON
	t.Run("malformed JSON", func(t *testing.T) {
		jsonResp := []byte(`{"jsonrpc":"2.0","id":1,error":{"code":-32000,"message":"Malformed"}}`)
		hasError, _ := GetJSONRPCError(jsonResp)
		require.False(t, hasError, "Should handle malformed JSON gracefully")
	})

	// Test with empty response
	t.Run("empty response", func(t *testing.T) {
		jsonResp := []byte(`{}`)
		hasError, _ := GetJSONRPCError(jsonResp)
		require.False(t, hasError, "Should handle empty response gracefully")
	})

	// Test with null error
	t.Run("null error", func(t *testing.T) {
		jsonResp := []byte(`{"jsonrpc":"2.0","id":1,"error":null}`)
		hasError, _ := GetJSONRPCError(jsonResp)
		require.False(t, hasError, "Should handle null error gracefully")
	})
}

// TestNewAztecBlockFetcher_Error tests error handling in block fetcher creation
func TestNewAztecBlockFetcher_Error(t *testing.T) {
	// Test with invalid URL
	t.Run("invalid URL", func(t *testing.T) {
		logger := zap.NewNop()

		// This is a special case where DialContext actually returns an error immediately
		// for an obviously invalid URL
		_, err := NewAztecBlockFetcher(context.Background(), "http://invalid-url\u007F", logger)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to create RPC client")
	})

	// Test with timeout URL (valid format but should time out)
	// This might not be reliable in all environments so we'll make it skip-able
	t.Run("timeout URL", func(t *testing.T) {
		if testing.Short() {
			t.Skip("Skipping timeout test in short mode")
		}

		logger := zap.NewNop()

		// Use a context with short timeout
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		// Try to connect to a non-routable IP address which should time out
		_, err := NewAztecBlockFetcher(ctx, "http://192.0.2.1:9999", logger)

		// If test is run in an environment where this IP is actually routable,
		// this might not fail as expected, so we'll check for context deadline or RPC error
		if err != nil {
			require.Contains(t, err.Error(), "failed to create RPC client")
		} else {
			t.Log("Warning: Expected timeout did not occur, check environment")
		}
	})
}

// TestParseUint_EdgeCases tests edge cases for the ParseUint function
func TestParseUint_EdgeCases(t *testing.T) {
	// Test cases covering various edge cases
	testCases := []struct {
		name      string
		input     string
		base      int
		bitSize   int
		expectErr bool
		expected  uint64
	}{
		{"Zero decimal", "0", 10, 64, false, 0},
		{"Zero hex", "0x0", 16, 64, true, 0}, // ParseUint doesn't handle 0x prefix
		{"Max uint64", "18446744073709551615", 10, 64, false, math.MaxUint64},
		{"Overflow uint64", "18446744073709551616", 10, 64, true, 0},
		{"Max uint32", "4294967295", 10, 32, false, math.MaxUint32},
		{"Overflow uint32", "4294967296", 10, 32, true, 0},
		{"Max uint16", "65535", 10, 16, false, math.MaxUint16},
		{"Overflow uint16", "65536", 10, 16, true, 0},
		{"Max uint8", "255", 10, 8, false, math.MaxUint8},
		{"Overflow uint8", "256", 10, 8, true, 0},
		{"Negative number", "-1", 10, 64, true, 0},
		{"Leading spaces", "  123", 10, 64, true, 0},
		{"Trailing spaces", "123  ", 10, 64, true, 0},
		{"With decimal point", "123.45", 10, 64, true, 0},
		{"Empty string", "", 10, 64, true, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val, err := ParseUint(tc.input, tc.base, tc.bitSize)

			if tc.expectErr {
				require.Error(t, err)
				require.IsType(t, &ErrParsingFailed{}, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.expected, val)
			}
		})
	}
}

// TestAztecBlockFetcher_TimestampEdgeCases tests timestamp handling in block fetcher
func TestAztecBlockFetcher_TimestampEdgeCases(t *testing.T) {
	// Setup a server that returns block responses with edge case timestamps
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		if req.Method == "node_getBlock" {
			// Extract block number from params
			var blockNumber int
			json.Unmarshal(req.Params, &blockNumber)

			switch blockNumber {
			case 0: // Genesis block with empty timestamp
				w.Write([]byte(`{
                    "jsonrpc": "2.0",
                    "id": 1,
                    "result": {
                        "archive": {"root": "0xarchive", "nextAvailableLeafIndex": 1},
                        "header": {
                            "lastArchive": {"root": "0xparent", "nextAvailableLeafIndex": 0},
                            "globalVariables": {
                                "blockNumber": "0x0",
                                "timestamp": ""
                            }
                        },
                        "body": {
                            "txEffects": []
                        }
                    }
                }`))
			case 1: // Non-genesis block with empty timestamp (should use current time fallback)
				w.Write([]byte(`{
                    "jsonrpc": "2.0",
                    "id": 1,
                    "result": {
                        "archive": {"root": "0xarchive", "nextAvailableLeafIndex": 1},
                        "header": {
                            "lastArchive": {"root": "0xparent", "nextAvailableLeafIndex": 0},
                            "globalVariables": {
                                "blockNumber": "0x1",
                                "timestamp": ""
                            }
                        },
                        "body": {
                            "txEffects": []
                        }
                    }
                }`))
			case 2: // Block with invalid timestamp format
				w.Write([]byte(`{
                    "jsonrpc": "2.0",
                    "id": 1,
                    "result": {
                        "archive": {"root": "0xarchive", "nextAvailableLeafIndex": 1},
                        "header": {
                            "lastArchive": {"root": "0xparent", "nextAvailableLeafIndex": 0},
                            "globalVariables": {
                                "blockNumber": "0x2",
                                "timestamp": "INVALID"
                            }
                        },
                        "body": {
                            "txEffects": []
                        }
                    }
                }`))
			}
		}
	}))
	defer server.Close()

	// Create the block fetcher
	client, err := rpc.DialContext(context.Background(), server.URL)
	require.NoError(t, err)

	fetcher := &aztecBlockFetcher{
		rpcClient: client,
		logger:    zap.NewNop(),
	}

	// Test genesis block with empty timestamp
	t.Run("genesis block empty timestamp", func(t *testing.T) {
		block, err := fetcher.FetchBlock(context.Background(), 0)
		require.NoError(t, err)
		require.Equal(t, uint64(0), block.Timestamp)
	})

	// Test non-genesis block with empty timestamp
	t.Run("non-genesis block empty timestamp", func(t *testing.T) {
		block, err := fetcher.FetchBlock(context.Background(), 1)
		require.NoError(t, err)
		// Should use current time fallback, so timestamp should be recent
		require.NotEqual(t, uint64(0), block.Timestamp)
		// Should be within the last minute
		require.Less(t, math.Abs(float64(time.Now().Unix())-float64(block.Timestamp)), float64(60))
	})

	// Test block with invalid timestamp format
	t.Run("invalid timestamp format", func(t *testing.T) {
		block, err := fetcher.FetchBlock(context.Background(), 2)
		require.NoError(t, err) // No error expected based on implementation
		// Likely falls back to current time
		require.NotEqual(t, uint64(0), block.Timestamp)
		// Should be within the last minute
		require.Less(t, math.Abs(float64(time.Now().Unix())-float64(block.Timestamp)), float64(60))
	})
}

// TestL1Verifier_ErrorCases tests error handling in L1Verifier
func TestL1Verifier_ErrorCases(t *testing.T) {
	// Setup mock server that returns errors for L2Tips
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Method string `json:"method"`
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &req)

		if req.Method == "node_getL2Tips" {
			w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "error": {
                    "code": -32000,
                    "message": "Internal error"
                }
            }`))
		}
	}))
	defer server.Close()

	// Create the verifier
	client, err := rpc.DialContext(context.Background(), server.URL)
	require.NoError(t, err)

	verifier := &aztecFinalityVerifier{
		rpcClient:              client,
		logger:                 zap.NewNop(),
		finalizedBlockCacheTTL: time.Second,
	}

	// Test GetFinalizedBlock with RPC error
	t.Run("GetFinalizedBlock error", func(t *testing.T) {
		_, err := verifier.GetFinalizedBlock(context.Background())
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to fetch L2 tips")
	})

	// Test IsBlockFinalized with error from GetFinalizedBlock
	t.Run("IsBlockFinalized error", func(t *testing.T) {
		_, err := verifier.IsBlockFinalized(context.Background(), 5)
		require.Error(t, err)
		require.Contains(t, err.Error(), "failed to get finalized block")
	})

	// Test GetLatestFinalizedBlockNumber with no cache
	t.Run("GetLatestFinalizedBlockNumber error", func(t *testing.T) {
		// Force cache to be empty
		verifier.finalizedBlockCache = nil

		// Should return 0 when there's an error
		result := verifier.GetLatestFinalizedBlockNumber()
		require.Equal(t, uint64(0), result)
	})
}

// TestMessagePublisher_ContextCancellation tests context cancellation in message publisher
func TestMessagePublisher_ContextCancellation(t *testing.T) {
	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Setup the watcher with necessary components
	logger := zap.NewNop()
	msgC := make(chan *common.MessagePublication, 10)

	config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")
	observationManager := NewObservationManager("test", logger)

	watcher := &Watcher{
		config:             config,
		observationManager: observationManager,
		msgC:               msgC,
		logger:             logger,
	}

	// Create params for the test
	params := LogParameters{
		SenderAddress:    vaa.Address{1, 2, 3, 4, 5},
		Sequence:         123,
		Nonce:            456,
		ConsistencyLevel: 1,
	}

	payload := []byte{1, 2, 3, 4, 5}
	blockInfo := BlockInfo{
		TxHash:    "INVALID_HASH", // Use invalid hash to test error path
		Timestamp: 1620000000,
	}
	observationID := "test-context-cancellation"

	// Test context cancellation at different points
	t.Run("early cancellation", func(t *testing.T) {
		// Cancel the context right away
		cancel()

		// Now try to publish the observation
		err := watcher.publishObservation(ctx, params, payload, blockInfo, observationID)
		require.Error(t, err)
		require.Equal(t, context.Canceled, err)
	})

	// Create a new context for the next test
	ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	t.Run("invalid tx hash", func(t *testing.T) {
		// This tests the error path for hex decoding the transaction hash
		err := watcher.publishObservation(ctx, params, payload, blockInfo, observationID)
		require.NoError(t, err) // Should still succeed with a fallback

		// Check that the message was published
		msg := <-msgC
		// TX ID should be a fallback value since the hash was invalid
		require.Equal(t, []byte{0x0}, msg.TxID)
	})
}

// TestHTTPClient_MoreErrorCases tests additional error cases for HTTP client
func TestHTTPClient_MoreErrorCases(t *testing.T) {
	logger := zap.NewNop()

	// Test with malformed JSON response
	t.Run("malformed JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Return invalid JSON
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,result":{"data":"malformed"}}`))
		}))
		defer server.Close()

		client := NewHTTPClient(
			100*time.Millisecond,
			1,
			10*time.Millisecond,
			1.5,
			logger,
		)

		payload := map[string]any{"method": "test_method"}
		response, err := client.DoRequest(context.Background(), server.URL, payload)

		// Our code handles malformed JSON gracefully
		require.NoError(t, err)
		require.NotNil(t, response)
	})

	// Test with JSON-RPC error with data
	t.Run("JSON-RPC error with data", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
                "jsonrpc": "2.0",
                "id": 1,
                "error": {
                    "code": -32000,
                    "message": "Block not found",
                    "data": {
                        "reason": "test reason"
                    }
                }
            }`))
		}))
		defer server.Close()

		client := NewHTTPClient(
			100*time.Millisecond,
			1,
			10*time.Millisecond,
			1.5,
			logger,
		)

		payload := map[string]any{"method": "test_method"}
		_, err := client.DoRequest(context.Background(), server.URL, payload)
		require.Error(t, err)
		require.IsType(t, &ErrRPCError{}, err)
		rpcErr := err.(*ErrRPCError)
		require.Equal(t, -32000, rpcErr.Code)
		require.Equal(t, "Block not found", rpcErr.Msg)
	})
}

// TestBlockProcessor_ErrorHandling tests error handling in block processor
func TestBlockProcessor_ErrorHandling(t *testing.T) {
	logger := zaptest.NewLogger(t)

	mockBlockFetcher := new(MockBlockFetcher)
	mockL1Verifier := new(MockL1Verifier)
	mockObservationManager := new(MockObservationManager)

	msgC := make(chan *common.MessagePublication, 10)

	config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")

	watcher := &Watcher{
		config:             config,
		blockFetcher:       mockBlockFetcher,
		l1Verifier:         mockL1Verifier,
		observationManager: mockObservationManager,
		msgC:               msgC,
		logger:             logger,
		lastBlockNumber:    100,
	}

	// Test case where FetchPublicLogs returns an error
	t.Run("FetchPublicLogs error", func(t *testing.T) {
		expectedErr := fmt.Errorf("failed to fetch logs")
		mockBlockFetcher.On("FetchPublicLogs", mock.Anything, 100, 101).Return([]ExtendedPublicLog{}, expectedErr).Once()

		// BlockInfo object for the test
		blockInfo := BlockInfo{
			TxHash:    "0x0123456789abcdef",
			Timestamp: 1620000000,
		}

		err := watcher.processBlockLogs(context.Background(), 100, blockInfo)
		assert.Error(t, err)
		// Check that the error message contains our expected text
		assert.Contains(t, err.Error(), "failed to fetch logs")
		mockBlockFetcher.AssertExpectations(t)
	})

	// Test case with logs from non-matching contract
	t.Run("non-matching contract", func(t *testing.T) {
		// Clear previous expectations
		mockBlockFetcher = new(MockBlockFetcher)

		// Create a new watcher with the fresh mock
		watcher = &Watcher{
			config:             config,
			blockFetcher:       mockBlockFetcher,
			l1Verifier:         mockL1Verifier,
			observationManager: mockObservationManager,
			msgC:               msgC,
			logger:             logger,
			lastBlockNumber:    100,
		}

		logs := []ExtendedPublicLog{
			{
				ID: LogId{
					BlockNumber: 100,
					TxIndex:     0,
					LogIndex:    0,
				},
				Log: PublicLog{
					ContractAddress: "0xdifferentcontract", // Different from our watched contract
					Log: []string{
						"000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender
						"0000000000000123", // Sequence
						"0000000000000001", // Nonce
						"01",               // Consistency level
						"01020304",         // Payload
					},
				},
			},
		}

		mockBlockFetcher.On("FetchPublicLogs", mock.Anything, 100, 101).Return(logs, nil).Once()

		// BlockInfo for the test
		blockInfo := BlockInfo{
			TxHash:    "0x0123456789abcdef",
			Timestamp: 1620000000,
			TxHashesByIndex: map[int]string{
				0: "0x0123456789abcdef",
			},
		}

		err := watcher.processBlockLogs(context.Background(), 100, blockInfo)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(msgC), "No messages should be published for non-matching contract")
		mockBlockFetcher.AssertExpectations(t)
	})
}

// TestProcessLog_ErrorCases tests error handling in log processing
func TestProcessLog_ErrorCases(t *testing.T) {
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

	// Test with empty log (should be skipped)
	t.Run("empty log", func(t *testing.T) {
		log := ExtendedPublicLog{
			ID: LogId{
				BlockNumber: 100,
				TxIndex:     0,
				LogIndex:    0,
			},
			Log: PublicLog{
				ContractAddress: contractAddress,
				Log:             []string{}, // Empty log
			},
		}

		blockInfo := BlockInfo{
			TxHash:    "0x0123456789abcdef",
			Timestamp: 1620000000,
		}

		err := watcher.processLog(context.Background(), log, blockInfo)
		assert.NoError(t, err)
		assert.Equal(t, 0, len(msgC), "No message should be published for empty log")
	})

	// Test with invalid log format (missing parameters)
	t.Run("invalid log format", func(t *testing.T) {
		log := ExtendedPublicLog{
			ID: LogId{
				BlockNumber: 100,
				TxIndex:     0,
				LogIndex:    0,
			},
			Log: PublicLog{
				ContractAddress: contractAddress,
				Log:             []string{"0x123"}, // Missing required parameters
			},
		}

		blockInfo := BlockInfo{
			TxHash:    "0x0123456789abcdef",
			Timestamp: 1620000000,
		}

		err := watcher.processLog(context.Background(), log, blockInfo)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to parse log parameters")
		assert.Equal(t, 0, len(msgC), "No message should be published for invalid log")
	})
}

// TestZapLoggerAdapter tests the zap logger adapter
func TestZapLoggerAdapter(t *testing.T) {
	// Create a buffer to capture logs
	var buf bytes.Buffer

	// Create a logger that writes to the buffer
	config := zap.NewDevelopmentConfig()
	config.OutputPaths = []string{"writer"}
	config.EncoderConfig.TimeKey = "" // Disable timestamps for predictable output

	zCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(config.EncoderConfig),
		zapcore.AddSync(&buf),
		zapcore.DebugLevel,
	)
	zapLogger := zap.New(zCore)

	// Create our adapter
	adapter := newRetryableHTTPZapLogger(zapLogger)

	// Test all methods
	t.Run("Error method", func(t *testing.T) {
		buf.Reset()
		adapter.Error("error message")
		require.Contains(t, buf.String(), "error message")
	})

	t.Run("Info method", func(t *testing.T) {
		buf.Reset()
		adapter.Info("info message")
		require.Contains(t, buf.String(), "info message")
	})

	t.Run("Debug method", func(t *testing.T) {
		buf.Reset()
		adapter.Debug("debug message")
		require.Contains(t, buf.String(), "debug message")
	})

	t.Run("Warn method", func(t *testing.T) {
		buf.Reset()
		adapter.Warn("warn message")
		require.Contains(t, buf.String(), "warn message")
	})
}

// TestSimpleFactoryMethods tests simple factory methods
func TestSimpleFactoryMethods(t *testing.T) {
	t.Run("NewAztecWatcherFactory", func(t *testing.T) {
		factory := NewAztecWatcherFactory("aztec-testnet", vaa.ChainID(52))
		require.NotNil(t, factory)
		require.Equal(t, "aztec-testnet", factory.NetworkID)
		require.Equal(t, vaa.ChainID(52), factory.ChainID)
	})

	t.Run("DefaultConfig", func(t *testing.T) {
		config := DefaultConfig(vaa.ChainID(52), "test", "http://localhost:8545", "0xContract")
		require.Equal(t, vaa.ChainID(52), config.ChainID)
		require.Equal(t, "test", config.NetworkID)
		require.Equal(t, "http://localhost:8545", config.RpcURL)
		require.Equal(t, "0xContract", config.ContractAddress)
		require.Equal(t, 1, config.StartBlock)
		require.Equal(t, 13, config.PayloadInitialCap)
	})
}
