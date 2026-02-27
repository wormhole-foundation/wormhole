package ccq

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/query"
	eth_common "github.com/ethereum/go-ethereum/common"
	eth_crypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// createTestQueryRequest creates a minimal valid query request for testing
func createTestQueryRequest() *query.QueryRequest {
	return &query.QueryRequest{
		Nonce:     1,
		Timestamp: uint64(time.Now().Unix()), // #nosec G115 -- Unix timestamps are always positive
		PerChainQueries: []*query.PerChainQueryRequest{
			{
				ChainId: vaa.ChainIDEthereum,
				Query: &query.EthCallQueryRequest{
					BlockId: "0x1",
					CallData: []*query.EthCallData{
						{
							To:   make([]byte, 20),
							Data: []byte{0x01},
						},
					},
				},
			},
		},
	}
}

// signQueryRequest signs a query request with the given private key
func signQueryRequest(t *testing.T, qr *query.QueryRequest, sk *ecdsa.PrivateKey, env common.Environment) []byte {
	queryBytes, err := qr.Marshal()
	require.NoError(t, err)

	digest := query.QueryRequestDigest(env, queryBytes)
	sig, err := eth_crypto.Sign(digest.Bytes(), sk)
	require.NoError(t, err)

	return sig
}

// TestHandleQuery_ZeroAddressRejected tests that the zero address is rejected as a staker
func TestHandleQuery_ZeroAddressRejected(t *testing.T) {
	// Create query request with zero address as staker (in signed payload)
	zeroAddr := eth_common.Address{}
	qr := createTestQueryRequest()
	qr.StakerAddress = &zeroAddr // Invalid: zero address

	// Marshal should fail due to zero address validation
	_, err := qr.Marshal()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zero address", "Error should mention zero address")
}

// TestHandleQuery_InvalidJSON tests that invalid JSON is rejected
func TestHandleQuery_InvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/v1/query", bytes.NewReader([]byte("invalid json")))
	w := httptest.NewRecorder()

	server := &httpServer{
		topic:            nil,
		logger:           zap.NewNop(),
		env:              common.UnsafeDevNet,
		pendingResponses: NewPendingResponses(zap.NewNop()),
		loggingMap:       NewLoggingMap(),
	}

	server.handleQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Invalid JSON should be rejected")
}

// TestHandleQuery_InvalidSignature tests that requests with invalid signatures are rejected
func TestHandleQuery_InvalidSignature(t *testing.T) {
	qr := createTestQueryRequest()
	queryBytes, err := qr.Marshal()
	require.NoError(t, err)

	// Invalid signature (wrong length)
	invalidSig := make([]byte, 32)

	reqBody := queryRequest{
		Bytes:     hex.EncodeToString(queryBytes),
		Signature: hex.EncodeToString(invalidSig),
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server := &httpServer{
		topic:            nil,
		logger:           zap.NewNop(),
		env:              common.UnsafeDevNet,
		pendingResponses: NewPendingResponses(zap.NewNop()),
		loggingMap:       NewLoggingMap(),
	}

	server.handleQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Invalid signature should be rejected")
}

// TestHandleQuery_InvalidQueryRequest tests that invalid query requests are rejected
func TestHandleQuery_InvalidQueryRequest(t *testing.T) {
	// Create invalid query request (no per-chain queries)
	qr := &query.QueryRequest{
		Nonce:           1,
		Timestamp:       1704067200,
		PerChainQueries: nil, // Invalid: empty
	}
	_, err := qr.Marshal()
	// Marshal should fail for invalid request
	assert.Error(t, err, "Marshal should fail for invalid request")
}

// TestQueryRequest_StakerFieldInSignedPayload tests that the staker field in the signed payload works correctly
func TestQueryRequest_StakerFieldInSignedPayload(t *testing.T) {
	stakerAddr := eth_common.HexToAddress("0x1234567890123456789012345678901234567890")

	// Test with staker address
	qrWithStaker := createTestQueryRequest()
	qrWithStaker.StakerAddress = &stakerAddr

	bytes, err := qrWithStaker.Marshal()
	require.NoError(t, err)

	// Unmarshal and verify
	var unmarshaled query.QueryRequest
	err = unmarshaled.Unmarshal(bytes)
	require.NoError(t, err)

	assert.NotNil(t, unmarshaled.StakerAddress)
	assert.Equal(t, stakerAddr, *unmarshaled.StakerAddress)

	// Test without staker address (self-staking)
	qrWithoutStaker := createTestQueryRequest()
	qrWithoutStaker.StakerAddress = nil

	bytes, err = qrWithoutStaker.Marshal()
	require.NoError(t, err)

	err = unmarshaled.Unmarshal(bytes)
	require.NoError(t, err)

	assert.Nil(t, unmarshaled.StakerAddress, "Staker should be nil when not provided")
}

// TestHandleQuery_OversizedBodyRejected tests that oversized requests are rejected
func TestHandleQuery_OversizedBodyRejected(t *testing.T) {
	// Create a body larger than MAX_BODY_SIZE (512KB)
	largeBody := make([]byte, MAX_BODY_SIZE+1)
	for i := range largeBody {
		largeBody[i] = 'a'
	}

	req := httptest.NewRequest(http.MethodPost, "/v1/query", bytes.NewReader(largeBody))
	w := httptest.NewRecorder()

	server := &httpServer{
		logger:           zap.NewNop(),
		env:              common.UnsafeDevNet,
		pendingResponses: NewPendingResponses(zap.NewNop()),
		loggingMap:       NewLoggingMap(),
	}

	server.handleQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Oversized body should be rejected")
}

// TestHandleQuery_InvalidRecoveryID tests that signatures with invalid recovery IDs are rejected
func TestHandleQuery_InvalidRecoveryID(t *testing.T) {
	qr := createTestQueryRequest()
	queryBytes, err := qr.Marshal()
	require.NoError(t, err)

	// Create a signature with invalid recovery ID (not 0, 1, 27, or 28)
	invalidSig := make([]byte, 65)
	invalidSig[64] = 99 // Invalid recovery ID

	reqBody := queryRequest{
		Bytes:     hex.EncodeToString(queryBytes),
		Signature: hex.EncodeToString(invalidSig),
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server := &httpServer{
		logger:           zap.NewNop(),
		env:              common.UnsafeDevNet,
		pendingResponses: NewPendingResponses(zap.NewNop()),
		loggingMap:       NewLoggingMap(),
	}

	server.handleQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Invalid recovery ID should be rejected")
	assert.Contains(t, w.Body.String(), "invalid signature", "Error should mention invalid signature")
}

// TestHandleQuery_InvalidSignatureFormatHeader tests that invalid X-Signature-Format header is rejected
func TestHandleQuery_InvalidSignatureFormatHeader(t *testing.T) {
	sk, err := eth_crypto.GenerateKey()
	require.NoError(t, err)

	qr := createTestQueryRequest()
	queryBytes, err := qr.Marshal()
	require.NoError(t, err)

	sig := signQueryRequest(t, qr, sk, common.UnsafeDevNet)

	reqBody := queryRequest{
		Bytes:     hex.EncodeToString(queryBytes),
		Signature: hex.EncodeToString(sig),
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/query", bytes.NewReader(body))
	req.Header.Set("X-Signature-Format", "invalid_format") // Invalid header value
	w := httptest.NewRecorder()

	server := &httpServer{
		logger:           zap.NewNop(),
		env:              common.UnsafeDevNet,
		pendingResponses: NewPendingResponses(zap.NewNop()),
		loggingMap:       NewLoggingMap(),
	}

	server.handleQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Invalid X-Signature-Format should be rejected")
	assert.Contains(t, w.Body.String(), "invalid X-Signature-Format", "Error should mention invalid header")
}

// TestHandleQuery_SignatureMalleability tests that malleable signatures (s > n/2) are rejected
func TestHandleQuery_SignatureMalleability(t *testing.T) {
	qr := createTestQueryRequest()
	queryBytes, err := qr.Marshal()
	require.NoError(t, err)

	digest := query.QueryRequestDigest(common.UnsafeDevNet, queryBytes)

	// Create a signature with s value in upper half (malleable)
	// This requires manually crafting an invalid signature
	malleableSig := make([]byte, 65)
	// Set r to a valid value
	copy(malleableSig[0:32], digest.Bytes()[0:32])
	// Set s to a value > n/2 (0xFFFF... is definitely > n/2)
	for i := 32; i < 64; i++ {
		malleableSig[i] = 0xFF
	}
	malleableSig[64] = 0 // Valid recovery ID

	reqBody := queryRequest{
		Bytes:     hex.EncodeToString(queryBytes),
		Signature: hex.EncodeToString(malleableSig),
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server := &httpServer{
		logger:           zap.NewNop(),
		env:              common.UnsafeDevNet,
		pendingResponses: NewPendingResponses(zap.NewNop()),
		loggingMap:       NewLoggingMap(),
	}

	server.handleQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "Malleable signature should be rejected")
	assert.Contains(t, w.Body.String(), "invalid signature", "Error should mention invalid signature")
}

// TestHandleQuery_V1RequestRejected tests that the HTTP handler rejects v1 requests
func TestHandleQuery_V1RequestRejected(t *testing.T) {
	sk, err := eth_crypto.GenerateKey()
	require.NoError(t, err)

	// Build v1 wire-format bytes manually:
	// [version=1][nonce=1 as uint32][numQueries=1][per-chain-query bytes]
	qr := createTestQueryRequest()
	pcqBytes, err := qr.PerChainQueries[0].Marshal()
	require.NoError(t, err)

	var buf bytes.Buffer
	buf.WriteByte(query.MSG_VERSION_V1)
	buf.Write([]byte{0, 0, 0, 1}) // nonce = 1
	buf.WriteByte(1)               // 1 per-chain query
	buf.Write(pcqBytes)
	v1Bytes := buf.Bytes()

	// Sign the v1 bytes
	digest := query.QueryRequestDigest(common.UnsafeDevNet, v1Bytes)
	sig, err := eth_crypto.Sign(digest.Bytes(), sk)
	require.NoError(t, err)

	reqBody := queryRequest{
		Bytes:     hex.EncodeToString(v1Bytes),
		Signature: hex.EncodeToString(sig),
	}
	body, err := json.Marshal(reqBody)
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/query", bytes.NewReader(body))
	w := httptest.NewRecorder()

	server := &httpServer{
		logger:           zap.NewNop(),
		env:              common.UnsafeDevNet,
		pendingResponses: NewPendingResponses(zap.NewNop()),
		loggingMap:       NewLoggingMap(),
	}

	server.handleQuery(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code, "v1 request should be rejected")
	assert.Contains(t, w.Body.String(), "v2 query request required")
}

// TestSignatureRecovery_ValidFormats tests that signature recovery works for both formats
func TestSignatureRecovery_ValidFormats(t *testing.T) {
	sk, err := eth_crypto.GenerateKey()
	require.NoError(t, err)
	expectedAddr := eth_crypto.PubkeyToAddress(sk.PublicKey)

	qr := createTestQueryRequest()
	queryBytes, err := qr.Marshal()
	require.NoError(t, err)

	digest := query.QueryRequestDigest(common.UnsafeDevNet, queryBytes)

	testCases := []struct {
		name     string
		signFunc func() []byte
	}{
		{
			name: "raw format",
			signFunc: func() []byte {
				sig, _ := eth_crypto.Sign(digest.Bytes(), sk)
				return sig
			},
		},
		{
			name: "eip191 format",
			signFunc: func() []byte {
				// Sign with EIP-191 prefix
				prefixed := eth_crypto.Keccak256(
					[]byte("\x19Ethereum Signed Message:\n32"),
					digest.Bytes(),
				)
				sig, _ := eth_crypto.Sign(prefixed, sk)
				return sig
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sig := tc.signFunc()

			// Test raw recovery
			if tc.name == "raw format" {
				recovered, err := query.RecoverQueryRequestSigner(digest.Bytes(), sig)
				require.NoError(t, err)
				assert.Equal(t, expectedAddr, recovered, "Raw signature should recover to correct address")
			}

			// Test EIP-191 recovery
			if tc.name == "eip191 format" {
				recovered, err := query.RecoverPrefixedSigner(digest.Bytes(), sig)
				require.NoError(t, err)
				assert.Equal(t, expectedAddr, recovered, "EIP-191 signature should recover to correct address")
			}
		})
	}
}
