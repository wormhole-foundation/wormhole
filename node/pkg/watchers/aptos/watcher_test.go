package aptos

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/watchers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Test against real indexer to understand data format
func TestQueryIndexerRealData(t *testing.T) {
	// Skip this test in CI - it's for local development/exploration
	t.Skip("Skipping real indexer test - for local development only")

	indexerRPC := "https://your-indexer-endpoint.com/v1/graphql"
	indexerToken := "your-indexer-token" //nolint:gosec // This is a test token // Add your indexer token here

	// Create a minimal watcher with just the fields we need
	w := &Watcher{
		chainID:           vaa.ChainIDAptos,
		networkID:         string(watchers.NetworkID("aptos")),
		aptosIndexerRPC:   indexerRPC,
		aptosIndexerToken: indexerToken,
	}

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "GetLastEvent",
			query: "query GetLastEvent { msg(order_by: {sequence_num: desc}, limit: 1) { version sequence_num } }",
		},
		{
			name:  "GetNextEvents",
			query: "query GetNextEvents { msg(where: {sequence_num: {_gt: 173800}}, order_by: {sequence_num: asc}, limit: 5) { version sequence_num } }",
		},
		{
			name:  "GetVersionBySequence",
			query: "query GetVersionBySequence { msg(where: {sequence_num: {_eq: 173829}}) { version } }",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			headers := make(map[string]string)
			if w.aptosIndexerToken != "" {
				headers["Authorization"] = "Bearer " + w.aptosIndexerToken
			}

			result, err := w.queryIndexer(tc.query, headers)
			require.NoError(t, err, "Failed to query indexer")

			// Parse and print the result
			fmt.Printf("\n=== %s ===\n", tc.name)
			fmt.Printf("Query: %s\n", tc.query)
			fmt.Printf("Response: %s\n", string(result))

			// Validate it's valid JSON
			assert.True(t, gjson.Valid(string(result)))

			// Parse and check structure
			parsed := gjson.ParseBytes(result)
			data := parsed.Get("data.msg")
			if data.Exists() {
				fmt.Printf("Found %d results\n", len(data.Array()))
				for i, msg := range data.Array() {
					version := msg.Get("version")
					seqNum := msg.Get("sequence_num")
					fmt.Printf("  [%d] version: %s, sequence_num: %s\n", i, version.String(), seqNum.String())
				}
			}

			// Check for errors
			errors := parsed.Get("errors")
			if errors.Exists() {
				fmt.Printf("GraphQL Errors: %s\n", errors.String())
			}
		})
	}
}

// Unit tests with mocked responses
func TestQueryIndexer(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		token            string
		mockResponse     string
		mockStatusCode   int
		expectedError    bool
		expectedHeaders  map[string]string
		validateResponse func(t *testing.T, result []byte)
	}{
		{
			name:           "successful query with data",
			query:          "query GetLastEvent { msg(order_by: {sequence_num: desc}, limit: 1) { version sequence_num } }",
			token:          "test-token",
			mockResponse:   `{"data":{"msg":[{"version":"3452526584","sequence_num":"173838"}]}}`,
			mockStatusCode: 200,
			expectedError:  false,
			expectedHeaders: map[string]string{
				"Authorization": "Bearer test-token",
				"Content-Type":  "application/json",
			},
			validateResponse: func(t *testing.T, result []byte) {
				parsed := gjson.ParseBytes(result)
				assert.True(t, parsed.Get("data.msg").Exists())
				assert.Equal(t, "3452526584", parsed.Get("data.msg.0.version").String())
				assert.Equal(t, "173838", parsed.Get("data.msg.0.sequence_num").String())
			},
		},
		{
			name:           "successful query with empty result",
			query:          "query GetVersionBySequence { msg(where: {sequence_num: {_eq: 999999}}) { version } }",
			token:          "",
			mockResponse:   `{"data":{"msg":[]}}`,
			mockStatusCode: 200,
			expectedError:  false,
			expectedHeaders: map[string]string{
				"Content-Type": "application/json",
				// No Authorization header when token is empty
			},
			validateResponse: func(t *testing.T, result []byte) {
				parsed := gjson.ParseBytes(result)
				assert.True(t, parsed.Get("data.msg").Exists())
				assert.Equal(t, 0, len(parsed.Get("data.msg").Array()))
			},
		},
		{
			name:           "graphql error response",
			query:          "query Invalid { invalid }",
			token:          "test-token",
			mockResponse:   `{"errors":[{"message":"Cannot query field 'invalid' on type 'query_root'"}]}`,
			mockStatusCode: 200,  // GraphQL returns 200 even for errors
			expectedError:  true, // Now returns an error instead of the error body
			validateResponse: func(t *testing.T, result []byte) {
				// This won't be called since we expect an error
			},
		},
		{
			name:           "http error response",
			query:          "query Test { msg { version } }",
			token:          "invalid-token",
			mockResponse:   `{"error": "Unauthorized"}`,
			mockStatusCode: 401,
			expectedError:  false, // queryIndexer doesn't check status code
			validateResponse: func(t *testing.T, result []byte) {
				assert.Contains(t, string(result), "Unauthorized")
			},
		},
		{
			name:           "invalid json response",
			query:          "query Test { msg { version } }",
			token:          "",
			mockResponse:   `not json`,
			mockStatusCode: 200,
			expectedError:  false,
			validateResponse: func(t *testing.T, result []byte) {
				assert.Equal(t, "not json", string(result))
				assert.False(t, gjson.Valid(string(result)))
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify method
				assert.Equal(t, "POST", r.Method)

				// Verify headers
				for key, expected := range tc.expectedHeaders {
					actual := r.Header.Get(key)
					assert.Equal(t, expected, actual, "Header %s mismatch", key)
				}

				// Verify the request body contains our query
				var requestBody map[string]interface{}
				err := json.NewDecoder(r.Body).Decode(&requestBody)
				assert.NoError(t, err)
				assert.Equal(t, tc.query, requestBody["query"])

				// Send mock response
				w.WriteHeader(tc.mockStatusCode)
				_, _ = w.Write([]byte(tc.mockResponse)) //nolint:errcheck // Test code
			}))
			defer server.Close()

			// Create watcher with test server URL
			w := &Watcher{
				aptosIndexerRPC:   server.URL,
				aptosIndexerToken: tc.token,
			}

			// Prepare headers
			headers := make(map[string]string)
			if w.aptosIndexerToken != "" {
				headers["Authorization"] = "Bearer " + w.aptosIndexerToken
			}

			// Call queryIndexer
			result, err := w.queryIndexer(tc.query, headers)

			// Check error
			if tc.expectedError {
				assert.Error(t, err)
				// For GraphQL error test, verify error message
				if tc.name == "graphql error response" {
					assert.Contains(t, err.Error(), "invalid")
				}
			} else {
				assert.NoError(t, err)
			}

			// Validate response
			if tc.validateResponse != nil && !tc.expectedError {
				tc.validateResponse(t, result)
			}
		})
	}
}

// Test processTransactionVersion against real Aptos RPC to understand data format
func TestProcessTransactionVersionRealData(t *testing.T) {
	// Skip this test in CI - it's for local development/exploration
	t.Skip("Skipping real Aptos RPC test - for local development only")

	aptosRPC := "https://fullnode.mainnet.aptoslabs.com"
	indexerRPC := "https://your-indexer-endpoint.com/v1/graphql"
	indexerToken := "your-indexer-token" //nolint:gosec // This is a test token
	aptosAccount := "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625"

	// Create a watcher with real endpoints
	w := &Watcher{
		chainID:           vaa.ChainIDAptos,
		networkID:         "aptos-mainnet",
		aptosRPC:          aptosRPC,
		aptosAccount:      aptosAccount,
		aptosIndexerRPC:   indexerRPC,
		aptosIndexerToken: indexerToken,
		msgC:              make(chan<- *common.MessagePublication, 10),
	}

	// First, get some version numbers from the indexer
	query := "query GetRecentEvents { msg(order_by: {sequence_num: desc}, limit: 3) { version sequence_num } }"
	headers := make(map[string]string)
	headers["Authorization"] = "Bearer " + indexerToken

	eventsJson, err := w.queryIndexer(query, headers)
	require.NoError(t, err, "Failed to query indexer for test versions")

	parsed := gjson.ParseBytes(eventsJson)
	messages := parsed.Get("data.msg")
	require.True(t, messages.Exists(), "No data.msg in indexer response")

	logger := zap.NewNop()

	// Test a few recent transactions
	for i, msg := range messages.Array() {
		if i >= 3 { // Limit to 3 tests
			break
		}

		version := msg.Get("version")
		sequenceNum := msg.Get("sequence_num")

		if !version.Exists() || !sequenceNum.Exists() {
			continue
		}

		versionNum := version.Uint()
		seqNum := sequenceNum.Uint()

		fmt.Printf("\n=== Testing Version %d (sequence %d) ===\n", versionNum, seqNum)

		// Test processTransactionVersion
		err := w.processTransactionVersion(logger, versionNum, false)

		fmt.Printf("Result: ")
		if err != nil {
			fmt.Printf("Error - %v\n", err)
		} else {
			fmt.Printf("Success\n")
		}

		// Also fetch the raw transaction to see what it looks like
		txEndpoint := fmt.Sprintf("%s/v1/transactions/by_version/%d", aptosRPC, versionNum)
		txData, err := w.retrievePayload(txEndpoint)
		if err != nil {
			fmt.Printf("Failed to fetch transaction: %v\n", err)
			continue
		}

		// Parse and show event types
		if gjson.Valid(string(txData)) {
			txResult := gjson.ParseBytes(txData)
			events := txResult.Get("events")

			fmt.Printf("Transaction events:\n")
			if events.Exists() {
				for j, event := range events.Array() {
					eventType := event.Get("type")
					fmt.Printf("  [%d] %s\n", j, eventType.String())
				}
			} else {
				fmt.Printf("  No events found\n")
			}
		}
	}
}

// Test processTransactionVersion with mocked responses
func TestProcessTransactionVersion(t *testing.T) {
	// Mock transaction response based on txResult.json
	mockTxWithEvent := `{
		"version": "3432334170",
		"hash": "0xb56e99068a50e98d9eb3a447854c265d0ab472b5ecf6d583c71152e96fbf0e84",
		"success": true,
		"events": [
			{
				"guid": {
					"creation_number": "0",
					"account_address": "0x0"
				},
				"sequence_number": "0",
				"type": "0x1::fungible_asset::Withdraw",
				"data": {
					"amount": "1000000",
					"store": "0xbb3e678df60319ddb9e7c570e5a01bc2d57b38d4f0fc6b571d140c6a40528918"
				}
			},
			{
				"guid": {
					"creation_number": "2",
					"account_address": "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625"
				},
				"sequence_number": "173829",
				"type": "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625::state::WormholeMessage",
				"data": {
					"consistency_level": 0,
					"nonce": "0",
					"payload": "0x0100000000000000000000000000000000000000000000000000000000000f4240a867703f5395cb2965feb7ebff5cdf39b771fc6156085da3ae4147a00be91b380016b730bbb0b27faef6cb524bd733649ddf1a91f3639eee1d300eff231fda49d64000150000000000000000000000000000000000000000000000000000000000000000",
					"sender": "1",
					"sequence": "170690",
					"timestamp": "1758377559"
				}
			}
		]
	}`

	mockTxWithoutEvent := `{
		"version": "3432334171",
		"hash": "0xtest123",
		"success": true,
		"events": [
			{
				"type": "0x1::coin::WithdrawEvent",
				"data": {
					"amount": "100"
				}
			}
		]
	}`

	mockInvalidJson := `{invalid json}`

	tests := []struct {
		name               string
		versionNum         uint64
		sequenceForObserve uint64
		isReobservation    bool
		mockResponse       string
		mockStatusCode     int
		expectedError      bool
		expectObserveCall  bool
		aptosAccount       string
	}{
		{
			name:               "successful transaction with WormholeMessage event",
			versionNum:         3432334170,
			sequenceForObserve: 173829,
			isReobservation:    false,
			mockResponse:       mockTxWithEvent,
			mockStatusCode:     200,
			expectedError:      false,
			expectObserveCall:  true,
			aptosAccount:       "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
		},
		{
			name:               "successful transaction without WormholeMessage event",
			versionNum:         3432334171,
			sequenceForObserve: 173830,
			isReobservation:    false,
			mockResponse:       mockTxWithoutEvent,
			mockStatusCode:     200,
			expectedError:      false,
			expectObserveCall:  false,
			aptosAccount:       "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
		},
		{
			name:               "reobservation request",
			versionNum:         3432334170,
			sequenceForObserve: 173829,
			isReobservation:    true,
			mockResponse:       mockTxWithEvent,
			mockStatusCode:     200,
			expectedError:      false,
			expectObserveCall:  true,
			aptosAccount:       "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
		},
		{
			name:               "server error with empty response",
			versionNum:         3432334172,
			sequenceForObserve: 173831,
			isReobservation:    false,
			mockResponse:       "",
			mockStatusCode:     500,
			expectedError:      true, // empty response is invalid JSON
			expectObserveCall:  false,
			aptosAccount:       "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
		},
		{
			name:               "invalid JSON response",
			versionNum:         3432334173,
			sequenceForObserve: 173832,
			isReobservation:    false,
			mockResponse:       mockInvalidJson,
			mockStatusCode:     200,
			expectedError:      true,
			expectObserveCall:  false,
			aptosAccount:       "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
		},
		{
			name:               "different account address - no event match",
			versionNum:         3432334170,
			sequenceForObserve: 173829,
			isReobservation:    false,
			mockResponse:       mockTxWithEvent,
			mockStatusCode:     200,
			expectedError:      false,
			expectObserveCall:  false,
			aptosAccount:       "0xdifferentaccount",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Track if observeData was called
			observeCalled := false
			var observedSequence uint64
			var observedIsReobservation bool

			// Create a test server for the Aptos RPC
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify the URL matches expected pattern
				expectedPath := fmt.Sprintf("/v1/transactions/by_version/%d", tc.versionNum)
				assert.Equal(t, expectedPath, r.URL.Path)

				// Send mock response
				w.WriteHeader(tc.mockStatusCode)
				_, _ = w.Write([]byte(tc.mockResponse)) //nolint:errcheck // Test code
			}))
			defer server.Close()

			// Create watcher with test server URL
			w := &Watcher{
				chainID:      vaa.ChainIDAptos,
				networkID:    "aptos-test",
				aptosRPC:     server.URL,
				aptosAccount: tc.aptosAccount,
				// Create a mock message channel to capture observations
				msgC: make(chan<- *common.MessagePublication, 1),
			}

			// Override observeData for testing (we'd need to refactor for proper mocking)
			// For now, we can test the function returns the expected error

			logger := zap.NewNop()
			err := w.processTransactionVersion(logger, tc.versionNum, tc.isReobservation)

			if tc.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Note: In a real test, we'd want to mock observeData to verify it's called
			// This would require refactoring to use an interface or function parameter
			_ = observeCalled
			_ = observedSequence
			_ = observedIsReobservation
		})
	}
}

// Test NewWatcher with different mode configurations
func TestNewWatcherModes(t *testing.T) {
	tests := []struct {
		name               string
		chainID            vaa.ChainID
		networkID          watchers.NetworkID
		rpc                string
		account            string
		handle             string
		indexerRpc         string
		indexerToken       string
		useIndexer         bool
		expectedUseIndexer bool
	}{
		{
			name:               "legacy mode - all basic params",
			chainID:            vaa.ChainIDAptos,
			networkID:          "aptos",
			rpc:                "https://fullnode.mainnet.aptoslabs.com",
			account:            "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
			handle:             "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002::state::WormholeMessage",
			indexerRpc:         "",
			indexerToken:       "",
			useIndexer:         false,
			expectedUseIndexer: false,
		},
		{
			name:               "indexer mode - with indexer params",
			chainID:            vaa.ChainIDAptos,
			networkID:          "aptos",
			rpc:                "https://fullnode.mainnet.aptoslabs.com",
			account:            "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
			handle:             "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002::state::WormholeMessage",
			indexerRpc:         "https://example.com/v1/graphql",
			indexerToken:       "test-token",
			useIndexer:         true,
			expectedUseIndexer: true,
		},
		{
			name:               "indexer mode without token",
			chainID:            vaa.ChainIDAptos,
			networkID:          "aptos",
			rpc:                "https://fullnode.mainnet.aptoslabs.com",
			account:            "0x5bc11445584a763c1fa7ed39081f1b920954da14e04b32440cba863d03e19625",
			handle:             "0x0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000002::state::WormholeMessage",
			indexerRpc:         "https://example.com/v1/graphql",
			indexerToken:       "",
			useIndexer:         true,
			expectedUseIndexer: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock channels
			msgC := make(chan *common.MessagePublication, 10)
			obsvReqC := make(chan *gossipv1.ObservationRequest, 10) // Use interface{} to match signature

			// Create watcher
			watcher := NewWatcher(
				tc.chainID,
				tc.networkID,
				tc.rpc,
				tc.account,
				tc.handle,
				tc.indexerRpc,
				tc.indexerToken,
				tc.useIndexer,
				msgC,
				obsvReqC,
			)

			// Verify watcher configuration
			assert.Equal(t, tc.chainID, watcher.chainID)
			assert.Equal(t, string(tc.networkID), watcher.networkID)
			assert.Equal(t, tc.rpc, watcher.aptosRPC)
			assert.Equal(t, tc.account, watcher.aptosAccount)
			assert.Equal(t, tc.handle, watcher.aptosHandle)
			assert.Equal(t, tc.indexerRpc, watcher.aptosIndexerRPC)
			assert.Equal(t, tc.indexerToken, watcher.aptosIndexerToken)
			assert.Equal(t, tc.expectedUseIndexer, watcher.useIndexer)
		})
	}
}

// Test that mode detection works correctly in NewWatcher
func TestModeDetection(t *testing.T) {
	tests := []struct {
		name               string
		useIndexer         bool
		expectedUseIndexer bool
	}{
		{
			name:               "legacy mode",
			useIndexer:         false,
			expectedUseIndexer: false,
		},
		{
			name:               "indexer mode",
			useIndexer:         true,
			expectedUseIndexer: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock channels
			msgC := make(chan *common.MessagePublication, 10)
			obsvReqC := make(chan *gossipv1.ObservationRequest, 10)

			// Create watcher with minimal config
			watcher := NewWatcher(
				vaa.ChainIDAptos,
				"aptos",
				"https://example.com",
				"0x1",
				"handle",
				"https://indexer.com",
				"token",
				tc.useIndexer,
				msgC,
				obsvReqC,
			)

			// Verify mode is set correctly
			assert.Equal(t, tc.expectedUseIndexer, watcher.useIndexer)
		})
	}
}
