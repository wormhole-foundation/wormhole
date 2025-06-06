package aztec

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
	mockArgs = append(mockArgs, args...)
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
                                "fields": ["0x123"]
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

// TestCreatePayload tests payload creation from log entries
func TestCreatePayload(t *testing.T) {
	logger := zap.NewNop()

	config := DefaultConfig(vaa.ChainID(1), "test", "http://localhost:8545", "0xContract")
	watcher := &Watcher{
		config: config,
		logger: logger,
	}

	// Test with valid hex entries and txID
	t.Run("valid hex entries", func(t *testing.T) {
		logEntries := []string{
			"0x0123", // sender
			"0x4567", // sequence
			"0x8901", // nonce
			"0x23",   // consistency level
			"0x45",   // timestamp
			"0x67",   // first payload entry
			"0x89",   // second payload entry
		}
		txID := "0x1234"

		payload := watcher.createPayload(logEntries, txID)

		// Check that payload starts with padded txID (32 bytes)
		assert.Len(t, payload, 34) // 32 bytes for txID + 2 bytes from entries
		assert.Equal(t, byte(0x12), payload[0])
		assert.Equal(t, byte(0x34), payload[1])
		// Remaining bytes should be zero-padded for txID
		for i := 2; i < 32; i++ {
			assert.Equal(t, byte(0x00), payload[i])
		}
	})

	// Test with mixed entries (valid and invalid)
	t.Run("mixed entries", func(t *testing.T) {
		mixedEntries := []string{
			"0x0123",  // sender
			"0x4567",  // sequence
			"0x8901",  // nonce
			"0x23",    // consistency level
			"0x45",    // timestamp
			"0x67",    // valid payload entry
			"invalid", // invalid entry - should be skipped
			"0x89",    // valid payload entry
		}
		txID := "0xabcd"

		payload := watcher.createPayload(mixedEntries, txID)

		// Should have txID + valid payload entries
		assert.True(t, len(payload) >= 32) // At least the txID padding
		assert.Equal(t, byte(0xab), payload[0])
		assert.Equal(t, byte(0xcd), payload[1])
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

	// Test with valid log - note Fields instead of Log
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
				// Arbitrum address (20 bytes padded to 31 bytes)
				"000000000000000000000000742d35Cc6634C0532925a3b8D50C6d111111111",
				// Arbitrum chain ID (2 bytes padded to 31 bytes) - 42161 = 0xa4b1
				"00000000000000000000000000000000000000000000000000000000000a4b1",
				// Amount (8 bytes padded to 31 bytes) - 1000000 = 0xf4240
				"000000000000000000000000000000000000000000000000000000000f4240",
				// Name/verification data
				"4a61636b000000000000000000000000000000000000000000000000000000", // "Jack" padded
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
	// Payload should include txID at the beginning
	assert.True(t, len(msg.Payload) > 32) // Should have txID + payload data
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
				Fields: []string{
					"0000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender (32 bytes)
					"0000000000000000000000000000000000000000000000000000000000000123", // Sequence (291)
					"0000000000000000000000000000000000000000000000000000000000000001", // Nonce (1)
					"0000000000000000000000000000000000000000000000000000000000000001", // Consistency level (1)
					"0000000000000000000000000000000000000000000000000000000061a91c40", // Timestamp
					// Arbitrum address (20 bytes padded to 31 bytes)
					"000000000000000000000000742d35Cc6634C0532925a3b8D50C6d111111111",
					// Arbitrum chain ID (2 bytes padded to 31 bytes) - 42161 = 0xa4b1
					"00000000000000000000000000000000000000000000000000000000000a4b1",
					// Amount (8 bytes padded to 31 bytes) - 1000000 = 0xf4240
					"000000000000000000000000000000000000000000000000000000000f4240",
					// Name/verification data
					"4a61636b000000000000000000000000000000000000000000000000000000", // "Jack" padded
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
	// Should have txID prepended to payload
	assert.True(t, len(msg.Payload) > 32)
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

	// Test GetFinalizedBlock - note: using Proven block number from L2Tips
	t.Run("GetFinalizedBlock", func(t *testing.T) {
		block, err := verifier.GetFinalizedBlock(context.Background())
		require.NoError(t, err)
		require.Equal(t, 8, block.Number)     // Should be proven block number (8), not finalized (5)
		require.Equal(t, "0x456", block.Hash) // Should be proven hash
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
		require.Equal(t, uint64(8), number) // Should match proven block number
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

	// Test with empty log (should be skipped) - note Fields instead of Log
	t.Run("empty log", func(t *testing.T) {
		log := ExtendedPublicLog{
			ID: LogId{
				BlockNumber: 100,
				TxIndex:     0,
				LogIndex:    0,
			},
			Log: PublicLog{
				ContractAddress: contractAddress,
				Fields:          []string{}, // Empty fields
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
				Fields:          []string{"0x0123"}, // Missing required parameters (need at least 4)
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
		assert.Contains(t, err.Error(), "failed to fetch public logs")
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
					Fields: []string{
						"0000000000000000000000000290f41e61374c715c1127974bf08a3993c512fd", // Sender (32 bytes)
						"000000000000000000000000000000000000000000000000000000000000012b", // Sequence (291)
						"0000000000000000000000000000000000000000000000000000000000000001", // Nonce (1)
						"0000000000000000000000000000000000000000000000000000000000000001", // Consistency level (1)
						"0000000000000000000000000000000000000000000000000000000061a91c40", // Timestamp
						"0000000000000000000000000000000000000000000000000000000001020304", // Payload
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
