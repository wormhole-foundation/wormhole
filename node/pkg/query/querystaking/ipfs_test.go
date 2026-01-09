package querystaking

import (
	"context"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
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
			wantHash: "", // We'll discover this
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

			// Create a test hash
			var testHash [32]byte
			copy(testHash[:], []byte("test hash for fetching"))

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

	// Create mock HTTP server that counts requests
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"EVM":{"5000":{"qpm":1}},"Solana":{"12500":{"qpm":1}}}`))
	}))
	defer server.Close()

	logger := zap.NewNop()
	client := NewIPFSClient(server.URL+"/", 5*time.Second, logger)

	var testHash [32]byte
	copy(testHash[:], []byte("test hash for caching"))

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
