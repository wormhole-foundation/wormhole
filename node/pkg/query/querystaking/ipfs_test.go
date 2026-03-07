package querystaking

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"go.uber.org/zap"
)

// TestBytes32ToCIDString tests CID conversion from bytes32 to string
func TestBytes32ToCIDString(t *testing.T) {
	tests := []struct {
		name       string
		hashHex    string
		wantCID    string // Expected full CID (if known)
		wantPrefix string // CIDv1 base32 always starts with "bafk"
		wantError  bool
	}{
		{
			name:       "valid sha256 hash",
			hashHex:    "2c26b46b68ffc68ff99b453c1d30413413422d706483bfa0f98a5e886266e7ae",
			wantPrefix: "bafk",
			wantError:  false,
		},
		{
			name:       "zero hash",
			hashHex:    "0000000000000000000000000000000000000000000000000000000000000000",
			wantPrefix: "bafk",
			wantError:  false,
		},
		{
			name:       "max hash",
			hashHex:    "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
			wantPrefix: "bafk",
			wantError:  false,
		},
		{
			name:       "rate limits hash (devnet file)",
			hashHex:    "02efea897e8c6894c980442c23bf07d6a4b0266cc085e479f7d37ad8cb017b6c",
			wantCID:    "bafkreiac57vis7umnckmtacefqr36b6wusycm3gaqxsht56tplmmwal3nq",
			wantPrefix: "bafk",
			wantError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hashBytes, err := hex.DecodeString(tt.hashHex)
			if err != nil {
				t.Fatalf("failed to decode hex: %v", err)
			}

			var hash32 [32]byte
			copy(hash32[:], hashBytes)

			got, err := bytes32ToCIDString(hash32)

			if tt.wantError {
				if err == nil {
					t.Errorf("bytes32ToCIDString() error = nil, wantError = true")
				}
				return
			}

			if err != nil {
				t.Errorf("bytes32ToCIDString() unexpected error = %v", err)
				return
			}

			// Check full CID if specified
			if tt.wantCID != "" {
				if got != tt.wantCID {
					t.Errorf("bytes32ToCIDString() = %v, want %v", got, tt.wantCID)
				}
			}

			// Check that CID starts with expected prefix
			if len(got) < len(tt.wantPrefix) || got[:len(tt.wantPrefix)] != tt.wantPrefix {
				t.Errorf("bytes32ToCIDString() = %v, want prefix %v", got, tt.wantPrefix)
			}

			// Check that CID has reasonable length (base32 encoded CIDv1 should be ~59 chars)
			if len(got) < 50 || len(got) > 70 {
				t.Errorf("bytes32ToCIDString() length = %d, expected ~59 chars", len(got))
			}

			t.Logf("CID: %s", got)
		})
	}
}

// TestCIDStringToBytes32 tests CID string to bytes32 conversion (reverse operation)
func TestCIDStringToBytes32(t *testing.T) {
	tests := []struct {
		name     string
		cidStr   string
		wantHash string // Expected SHA256 hash in hex
	}{
		{
			name:     "correct CID for rate limits in CI",
			cidStr:   "bafkreihtyy6gnalxiy5lojypcebbymzlq642kz6a7lna2rq7ee6ocl3jue",
			wantHash: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Parse the CID string
			c, err := cid.Decode(tt.cidStr)
			if err != nil {
				t.Fatalf("failed to parse CID: %v", err)
			}

			// Extract the hash digest from the multihash
			mh := c.Hash()

			// Decode the multihash to get the digest
			decoded, err := multihash.Decode(mh)
			if err != nil {
				t.Fatalf("failed to decode multihash: %v", err)
			}

			digest := decoded.Digest

			if len(digest) != 32 {
				t.Fatalf("unexpected digest length: got %d, want 32", len(digest))
			}

			hashHex := hex.EncodeToString(digest)
			t.Logf("Extracted SHA256 hash: %s", hashHex)

			// Verify round-trip: convert back to CID
			var hash32 [32]byte
			copy(hash32[:], digest)
			gotCID, err := bytes32ToCIDString(hash32)
			if err != nil {
				t.Fatalf("failed to convert back to CID: %v", err)
			}

			if gotCID != tt.cidStr {
				t.Errorf("round-trip failed: got %v, want %v", gotCID, tt.cidStr)
			}

			if tt.wantHash != "" && hashHex != tt.wantHash {
				t.Errorf("hash = %v, want %v", hashHex, tt.wantHash)
			}
		})
	}
}

// TestIPFSClientFetch tests the IPFS client fetch functionality with a mock server
func TestIPFSClientFetch(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantError      bool
		errorType      string
	}{
		{
			name:           "valid JSON response",
			serverResponse: `{"EVM":{"5000":{"qpm":1},"50000":{"qps":1,"qpm":60}},"Solana":{"12500":{"qpm":1},"125000":{"qps":1,"qpm":60}}}`,
			serverStatus:   http.StatusOK,
			wantError:      false,
		},
		{
			name:           "invalid JSON",
			serverResponse: `{invalid json`,
			serverStatus:   http.StatusOK,
			wantError:      true,
			errorType:      "json_parse",
		},
		{
			name:           "404 not found",
			serverResponse: `not found`,
			serverStatus:   http.StatusNotFound,
			wantError:      true,
			errorType:      "http_status",
		},
		{
			name:           "500 server error",
			serverResponse: `internal error`,
			serverStatus:   http.StatusInternalServerError,
			wantError:      true,
			errorType:      "http_status",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock HTTP server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				_, _ = w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			// Create IPFS client pointing to mock server
			logger := zap.NewNop()
			client := NewIPFSClient(server.URL+"/", 5*time.Second, logger)

			// Create a test hash that matches the server response
			var testHash [32]byte
			if tt.serverStatus == http.StatusOK {
				// Hash matches the actual content for successful responses
				testHash = sha256.Sum256([]byte(tt.serverResponse))
			} else {
				// For error responses, hash doesn't matter as we won't reach verification
				copy(testHash[:], []byte("test hash for fetching"))
			}

			// Attempt to fetch
			ctx := context.Background()
			result, err := client.FetchConversionTable(ctx, testHash)

			if tt.wantError {
				if err == nil {
					t.Errorf("FetchConversionTable() error = nil, wantError = true")
				}
				return
			}

			if err != nil {
				t.Errorf("FetchConversionTable() unexpected error = %v", err)
				return
			}

			if result == nil {
				t.Errorf("FetchConversionTable() result = nil, want non-nil")
			}
		})
	}
}

// TestIPFSClientCache tests that the IPFS client properly caches results
func TestIPFSClientCache(t *testing.T) {
	requestCount := 0

	// Define the content that will be served
	content := []byte(`{"EVM":{"5000":{"qpm":1}},"Solana":{"12500":{"qpm":1}}}`)

	// Create mock HTTP server that counts requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := NewIPFSClient(server.URL+"/", 5*time.Second, logger)

	// Create test hash that matches the content
	testHash := sha256.Sum256(content)

	ctx := context.Background()

	// First fetch should hit the server
	result1, err := client.FetchConversionTable(ctx, testHash)
	if err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}
	if result1 == nil {
		t.Fatal("first fetch returned nil result")
	}
	if requestCount != 1 {
		t.Errorf("first fetch: requestCount = %d, want 1", requestCount)
	}

	// Second fetch should use cache
	result2, err := client.FetchConversionTable(ctx, testHash)
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}
	if result2 == nil {
		t.Fatal("second fetch returned nil result")
	}
	if requestCount != 1 {
		t.Errorf("second fetch: requestCount = %d, want 1 (should use cache)", requestCount)
	}

	// Results should be the same object from cache
	if result1 != result2 {
		t.Error("cached result is not the same object")
	}
}

// TestIPFSClientTimeout tests that the IPFS client respects timeouts
func TestIPFSClientTimeout(t *testing.T) {
	// Create mock HTTP server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"EVM":{"5000":"1 QPM"}}`))
	}))
	defer server.Close()

	logger := zap.NewNop()
	// Set very short timeout
	client := NewIPFSClient(server.URL+"/", 100*time.Millisecond, logger)

	var testHash [32]byte
	copy(testHash[:], []byte("test hash for timeout"))

	ctx := context.Background()
	_, err := client.FetchConversionTable(ctx, testHash)

	if err == nil {
		t.Error("FetchConversionTable() expected timeout error, got nil")
	}
}

// TestVerifyContentHash tests the hash verification function with various edge cases
func TestVerifyContentHash(t *testing.T) {
	// Create valid test content
	validContent := []byte(`{"EVM":{"5000":{"qpm":1}},"Solana":{"12500":{"qpm":1}}}`)
	validHash := sha256.Sum256(validContent)

	// Create CID from the valid hash
	validMH, err := multihash.Encode(validHash[:], multihash.SHA2_256)
	if err != nil {
		t.Fatalf("failed to create multihash: %v", err)
	}
	validCID := cid.NewCidV1(cid.Raw, validMH).String()

	// Create CID with SHA2-512 (unsupported)
	sha512Hash := make([]byte, 64)
	copy(sha512Hash, validHash[:])
	unsupportedMH, err := multihash.Encode(sha512Hash, multihash.SHA2_512)
	if err != nil {
		t.Fatalf("failed to create SHA512 multihash: %v", err)
	}
	unsupportedCID := cid.NewCidV1(cid.Raw, unsupportedMH).String()

	tests := []struct {
		name      string
		content   []byte
		cidStr    string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid content and hash",
			content:   validContent,
			cidStr:    validCID,
			wantError: false,
		},
		{
			name:    "empty content with matching CID",
			content: []byte{},
			cidStr: func() string {
				h := sha256.Sum256([]byte{})
				mh, _ := multihash.Encode(h[:], multihash.SHA2_256)
				return cid.NewCidV1(cid.Raw, mh).String()
			}(),
			wantError: false,
		},
		{
			name:      "tampered content - hash mismatch",
			content:   []byte(`{"EVM":{"5000":{"qpm":999}}}`), // Different content
			cidStr:    validCID,
			wantError: true,
			errorMsg:  "hash mismatch",
		},
		{
			name:      "invalid CID format",
			content:   validContent,
			cidStr:    "invalid-cid-string",
			wantError: true,
			errorMsg:  "failed to decode CID",
		},
		{
			name:      "empty CID string",
			content:   validContent,
			cidStr:    "",
			wantError: true,
			errorMsg:  "failed to decode CID",
		},
		{
			name:      "unsupported hash algorithm (SHA2-512)",
			content:   validContent,
			cidStr:    unsupportedCID,
			wantError: true,
			errorMsg:  "unsupported hash algorithm",
		},
		{
			name:    "large content with correct hash",
			content: []byte(strings.Repeat("a", 1000000)), // 1MB
			cidStr: func() string {
				h := sha256.Sum256([]byte(strings.Repeat("a", 1000000)))
				mh, _ := multihash.Encode(h[:], multihash.SHA2_256)
				return cid.NewCidV1(cid.Raw, mh).String()
			}(),
			wantError: false,
		},
		{
			name:    "CIDv0 format (legacy)",
			content: validContent,
			cidStr: func() string {
				// CIDv0 uses base58 and different format
				h := sha256.Sum256(validContent)
				mh, _ := multihash.Encode(h[:], multihash.SHA2_256)
				return cid.NewCidV0(mh).String()
			}(),
			wantError: false, // Should still work with CIDv0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := verifyContentHash(tt.content, tt.cidStr)

			if tt.wantError {
				if err == nil {
					t.Errorf("verifyContentHash() error = nil, wantError = true")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("verifyContentHash() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("verifyContentHash() unexpected error = %v", err)
			}
		})
	}
}

// TestIPFSClientHashVerificationFailure tests that hash verification failures are properly handled
func TestIPFSClientHashVerificationFailure(t *testing.T) {
	// Create valid content and its hash
	validContent := []byte(`{"EVM":{"5000":{"qpm":1}},"Solana":{"12500":{"qpm":1}}}`)
	validHash := sha256.Sum256(validContent)

	// Server returns tampered content
	tamperedContent := []byte(`{"EVM":{"5000":{"qpm":999}},"Solana":{"12500":{"qpm":999}}}`)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(tamperedContent)
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := NewIPFSClient(server.URL+"/", 5*time.Second, logger)

	// Convert valid hash to bytes32
	var testHash [32]byte
	copy(testHash[:], validHash[:])

	ctx := context.Background()
	_, err := client.FetchConversionTable(ctx, testHash)

	if err == nil {
		t.Error("FetchConversionTable() expected hash verification error, got nil")
		return
	}

	if !strings.Contains(err.Error(), "integrity check failed") {
		t.Errorf("expected integrity check error, got: %v", err)
	}

	t.Logf("Expected error received: %v", err)
}

// TestConversionTableGetTranches tests tranche extraction
func TestConversionTableGetTranches(t *testing.T) {
	tests := []struct {
		name      string
		table     *ConversionTable
		chain     string
		wantError bool
		errorMsg  string
		validate  func(t *testing.T, tranches []ConversionTranche)
	}{
		{
			name: "valid EVM tranches",
			table: &ConversionTable{
				EVM: map[string]RateConfig{
					"5000":  {QPM: ptr(uint64(1))},
					"50000": {QPS: ptr(uint64(1)), QPM: ptr(uint64(60))},
				},
			},
			chain:     "EVM",
			wantError: false,
			validate: func(t *testing.T, tranches []ConversionTranche) {
				if len(tranches) != 2 {
					t.Errorf("expected 2 tranches, got %d", len(tranches))
				}
				// Should be sorted by tranche amount
				if tranches[0].Tranche != 5000 || tranches[1].Tranche != 50000 {
					t.Errorf("tranches not sorted correctly: %+v", tranches)
				}
			},
		},
		{
			name: "only QPS specified - QPM should be derived",
			table: &ConversionTable{
				EVM: map[string]RateConfig{
					"10000": {QPS: ptr(uint64(5))},
				},
			},
			chain:     "EVM",
			wantError: false,
			validate: func(t *testing.T, tranches []ConversionTranche) {
				if len(tranches) != 1 {
					t.Errorf("expected 1 tranche, got %d", len(tranches))
					return
				}
				if tranches[0].RatePerSecond != 5 {
					t.Errorf("expected QPS=5, got %d", tranches[0].RatePerSecond)
				}
				if tranches[0].RatePerMinute != 300 {
					t.Errorf("expected QPM=300 (5*60), got %d", tranches[0].RatePerMinute)
				}
			},
		},
		{
			name: "only QPM specified - QPS should be 0",
			table: &ConversionTable{
				EVM: map[string]RateConfig{
					"10000": {QPM: ptr(uint64(120))},
				},
			},
			chain:     "EVM",
			wantError: false,
			validate: func(t *testing.T, tranches []ConversionTranche) {
				if len(tranches) != 1 {
					t.Errorf("expected 1 tranche, got %d", len(tranches))
					return
				}
				if tranches[0].RatePerSecond != 0 {
					t.Errorf("expected QPS=0, got %d", tranches[0].RatePerSecond)
				}
				if tranches[0].RatePerMinute != 120 {
					t.Errorf("expected QPM=120, got %d", tranches[0].RatePerMinute)
				}
			},
		},
		{
			name: "both QPS and QPM zero",
			table: &ConversionTable{
				EVM: map[string]RateConfig{
					"10000": {},
				},
			},
			chain:     "EVM",
			wantError: true,
			errorMsg:  "no rate specified",
		},
		{
			name: "invalid tranche amount (not a number)",
			table: &ConversionTable{
				EVM: map[string]RateConfig{
					"not-a-number": {QPM: ptr(uint64(1))},
				},
			},
			chain:     "EVM",
			wantError: true,
			errorMsg:  "invalid tranche amount",
		},
		{
			name: "unknown chain",
			table: &ConversionTable{
				EVM: map[string]RateConfig{
					"5000": {QPM: ptr(uint64(1))},
				},
			},
			chain:     "Unknown",
			wantError: true,
			errorMsg:  "unknown chain",
		},
		{
			name: "empty tranches for chain",
			table: &ConversionTable{
				EVM: map[string]RateConfig{},
			},
			chain:     "EVM",
			wantError: true,
			errorMsg:  "no valid tranches found",
		},
		{
			name: "chain not present in table",
			table: &ConversionTable{
				EVM: map[string]RateConfig{
					"5000": {QPM: ptr(uint64(1))},
				},
			},
			chain:     "Solana",
			wantError: true,
			errorMsg:  "no rates found for chain",
		},
		{
			name: "multiple tranches sorted correctly",
			table: &ConversionTable{
				Solana: map[string]RateConfig{
					"100000": {QPS: ptr(uint64(10))},
					"50000":  {QPS: ptr(uint64(5))},
					"200000": {QPS: ptr(uint64(20))},
					"25000":  {QPS: ptr(uint64(2))},
				},
			},
			chain:     "Solana",
			wantError: false,
			validate: func(t *testing.T, tranches []ConversionTranche) {
				if len(tranches) != 4 {
					t.Errorf("expected 4 tranches, got %d", len(tranches))
					return
				}
				// Verify sorted order
				expected := []uint64{25000, 50000, 100000, 200000}
				for i, exp := range expected {
					if tranches[i].Tranche != exp {
						t.Errorf("tranche[%d]: expected %d, got %d", i, exp, tranches[i].Tranche)
					}
				}
			},
		},
		{
			name: "very large tranche amounts",
			table: &ConversionTable{
				EVM: map[string]RateConfig{
					"18446744073709551615": {QPM: ptr(uint64(1000))}, // max uint64
				},
			},
			chain:     "EVM",
			wantError: false,
			validate: func(t *testing.T, tranches []ConversionTranche) {
				if len(tranches) != 1 {
					t.Errorf("expected 1 tranche, got %d", len(tranches))
					return
				}
				if tranches[0].Tranche != 18446744073709551615 {
					t.Errorf("expected max uint64, got %d", tranches[0].Tranche)
				}
			},
		},
		{
			name: "zero tranche amount",
			table: &ConversionTable{
				EVM: map[string]RateConfig{
					"0": {QPM: ptr(uint64(1))},
				},
			},
			chain:     "EVM",
			wantError: true,
			errorMsg:  "tranche amount cannot be 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tranches, err := tt.table.GetTranchesByChain(tt.chain)

			if tt.wantError {
				if err == nil {
					t.Errorf("GetTranchesByChain() error = nil, wantError = true")
					return
				}
				if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("GetTranchesByChain() error = %v, want error containing %q", err, tt.errorMsg)
				}
				return
			}

			if err != nil {
				t.Errorf("GetTranchesByChain() unexpected error = %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, tranches)
			}
		})
	}
}

// TestConversionTableGetSupportedChains tests the GetSupportedChains method
func TestConversionTableGetSupportedChains(t *testing.T) {
	tests := []struct {
		name       string
		table      *ConversionTable
		wantChains []string
	}{
		{
			name: "both EVM and Solana",
			table: &ConversionTable{
				EVM:    map[string]RateConfig{"5000": {QPM: ptr(uint64(1))}},
				Solana: map[string]RateConfig{"12500": {QPM: ptr(uint64(1))}},
			},
			wantChains: []string{"EVM", "Solana"},
		},
		{
			name: "only EVM",
			table: &ConversionTable{
				EVM: map[string]RateConfig{"5000": {QPM: ptr(uint64(1))}},
			},
			wantChains: []string{"EVM"},
		},
		{
			name: "only Solana",
			table: &ConversionTable{
				Solana: map[string]RateConfig{"12500": {QPM: ptr(uint64(1))}},
			},
			wantChains: []string{"Solana"},
		},
		{
			name:       "empty table",
			table:      &ConversionTable{},
			wantChains: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chains := tt.table.GetSupportedChains()
			if len(chains) != len(tt.wantChains) {
				t.Errorf("GetSupportedChains() = %v, want %v", chains, tt.wantChains)
				return
			}
			for i, want := range tt.wantChains {
				if chains[i] != want {
					t.Errorf("GetSupportedChains()[%d] = %v, want %v", i, chains[i], want)
				}
			}
		})
	}
}

// TestIPFSClientConcurrentAccess tests concurrent cache access
func TestIPFSClientConcurrentAccess(t *testing.T) {
	validContent := []byte(`{"EVM":{"5000":{"qpm":1}},"Solana":{"12500":{"qpm":1}}}`)
	validHash := sha256.Sum256(validContent)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		time.Sleep(10 * time.Millisecond) // Simulate network delay
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(validContent)
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := NewIPFSClient(server.URL+"/", 5*time.Second, logger)

	var testHash [32]byte
	copy(testHash[:], validHash[:])

	ctx := context.Background()

	// Launch 10 concurrent requests for the same CID
	const numGoroutines = 10
	errChan := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			_, err := client.FetchConversionTable(ctx, testHash)
			errChan <- err
		}()
	}

	// Collect results
	for i := range numGoroutines {
		err := <-errChan
		if err != nil {
			t.Errorf("concurrent fetch %d failed: %v", i, err)
		}
	}

	// Due to caching and race conditions, we should have fewer requests than goroutines
	// But at least one request should have been made
	if requestCount < 1 {
		t.Errorf("expected at least 1 request, got %d", requestCount)
	}

	t.Logf("Made %d requests for %d concurrent fetches (cache reduced load)", requestCount, numGoroutines)
}

// TestIPFSClientContextCancellation tests that context cancellation is respected
func TestIPFSClientContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"EVM":{"5000":{"qpm":1}}}`))
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := NewIPFSClient(server.URL+"/", 5*time.Second, logger)

	var testHash [32]byte
	copy(testHash[:], []byte("test hash"))

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	_, err := client.FetchConversionTable(ctx, testHash)

	if err == nil {
		t.Error("expected context cancellation error, got nil")
	}
}

// ptr is a helper function to create pointers to uint64 values
func ptr(v uint64) *uint64 {
	return &v
}
