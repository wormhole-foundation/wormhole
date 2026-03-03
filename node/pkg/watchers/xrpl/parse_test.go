package xrpl

import (
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	ledger "github.com/Peersyst/xrpl-go/xrpl/ledger-entry-types"
	streamtypes "github.com/Peersyst/xrpl-go/xrpl/queries/subscription/types"
	"github.com/Peersyst/xrpl-go/xrpl/queries/transactions"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/Peersyst/xrpl-go/xrpl/transaction/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Sample NTT MemoData (hex-encoded, 72 bytes)
// Format: prefix(4) + recipientNTTManager(32) + recipientAddress(32) + recipientChain(2) + fromDecimals(1) + toDecimals(1)
// Prefix: 994E5454 (NTT)
// RecipientNTTManager: 0000000000000000000000001234567890abcdef1234567890abcdef12345678 (32 bytes)
// RecipientAddress: 000000000000000000000000D8DA6BF26964AF9D7EED9E03E53415D37AA96045 (32 bytes)
// RecipientChain: 0002 (Ethereum)
// FromDecimals: 06 (XRP has 6 decimals)
// ToDecimals: 08
const sampleNTTMemoData = "994E54540000000000000000000000001234567890abcdef1234567890abcdef12345678000000000000000000000000D8DA6BF26964AF9D7EED9E03E53415D37AA9604500020608"

// nttMemoFormat hex-encoded: "application/x-ntt-transfer"
const testNTTMemoFormat = "6170706C69636174696F6E2F782D6E74742D7472616E73666572"

// Helper to create a FlatTransaction with memos
func createFlatTransactionWithMemos(memoFormat, memoData string) transaction.FlatTransaction {
	return transaction.FlatTransaction{
		"Memos": []any{
			map[string]any{
				"Memo": map[string]any{
					"MemoFormat": memoFormat,
					"MemoData":   memoData,
				},
			},
		},
	}
}

// createValidNTTTransaction creates a FlatTransaction with all fields required
// for strict validation: meta with TransactionResult, Destination, TransactionType, Account, and NTT Memos.
func createValidNTTTransaction() transaction.FlatTransaction {
	return transaction.FlatTransaction{
		"TransactionType": "Payment",
		"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
		"meta": map[string]any{
			"TransactionResult": "tesSUCCESS",
		},
		"Memos": []any{
			map[string]any{
				"Memo": map[string]any{
					"MemoFormat": testNTTMemoFormat,
					"MemoData":   sampleNTTMemoData,
				},
			},
		},
	}
}

// createSampleMemoData creates a 72-byte NTT memo with specified parameters
func createSampleMemoData(recipientChain uint16, fromDecimals, toDecimals uint8) string {
	data := make([]byte, 72)
	copy(data[0:4], nttPrefix[:])
	// recipientNTTManager at 4-35 (32 bytes) - use sample address
	data[35] = 0x01 // non-zero byte in recipient NTT manager
	// recipientAddress at 36-67 (32 bytes) - use sample address
	data[67] = 0x45 // non-zero byte in recipient
	// recipientChain at 68-69 (2 bytes)
	binary.BigEndian.PutUint16(data[68:70], recipientChain)
	// fromDecimals at 70 (1 byte)
	data[70] = fromDecimals
	// toDecimals at 71 (1 byte)
	data[71] = toDecimals
	return hex.EncodeToString(data)
}

// =============================================================================
// parseMemoData tests
// =============================================================================

func TestParseMemoData_ValidNTTMemo(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := createFlatTransactionWithMemos(testNTTMemoFormat, sampleNTTMemoData)

	memo, err := p.parseMemoData(tx)

	require.NoError(t, err)
	require.NotNil(t, memo)

	// Verify recipient chain
	assert.Equal(t, uint16(2), memo.recipientChain) // Ethereum

	// Verify fromDecimals
	assert.Equal(t, uint8(6), memo.fromDecimals) // XRP has 6 decimals

	// Verify toDecimals
	assert.Equal(t, uint8(8), memo.toDecimals)
}

func TestParseMemoData_NoMemos(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := transaction.FlatTransaction{
		"Account": "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	memo, err := p.parseMemoData(tx)

	require.NoError(t, err)
	assert.Nil(t, memo, "Should return nil when no Memos field")
}

func TestParseMemoData_EmptyMemos(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := transaction.FlatTransaction{
		"Memos": []any{},
	}

	memo, err := p.parseMemoData(tx)

	require.NoError(t, err)
	assert.Nil(t, memo)
}

func TestParseMemoData_WrongMemoFormat(t *testing.T) {
	p := NewParser("", nil, nil)
	// Use a different MemoFormat (hex for "text/plain")
	wrongMemoFormat := "746578742F706C61696E"
	tx := createFlatTransactionWithMemos(wrongMemoFormat, sampleNTTMemoData)

	memo, err := p.parseMemoData(tx)

	require.NoError(t, err)
	assert.Nil(t, memo, "Should return nil for wrong MemoFormat")
}

func TestParseMemoData_InvalidNTTPrefix(t *testing.T) {
	p := NewParser("", nil, nil)
	// Valid hex but wrong prefix (not 994E5454), 72 bytes
	wrongPrefixData := "DEADBEEF" + "0000000000000000000000001234567890abcdef1234567890abcdef12345678" +
		"000000000000000000000000D8DA6BF26964AF9D7EED9E03E53415D37AA96045" + "0002" + "06" + "08"
	tx := createFlatTransactionWithMemos(testNTTMemoFormat, wrongPrefixData)

	memo, err := p.parseMemoData(tx)

	require.NoError(t, err)
	assert.Nil(t, memo, "Should return nil for wrong NTT prefix")
}

func TestParseMemoData_InvalidHexMemoData(t *testing.T) {
	p := NewParser("", nil, nil)
	// Invalid hex string
	tx := createFlatTransactionWithMemos(testNTTMemoFormat, "NOTVALIDHEX!!!")

	memo, err := p.parseMemoData(tx)

	require.Error(t, err, "Should return error for invalid hex")
	assert.Nil(t, memo)
}

func TestParseMemoData_WrongLength(t *testing.T) {
	p := NewParser("", nil, nil)
	// Too short (only 50 bytes)
	shortData := "994E5454" + "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"
	tx := createFlatTransactionWithMemos(testNTTMemoFormat, shortData)

	memo, err := p.parseMemoData(tx)

	require.Error(t, err, "Should return error for wrong length")
	assert.Nil(t, memo)
	assert.Contains(t, err.Error(), "invalid memo data length")
}

func TestParseMemoData_MultipleMemos_ValidAtIndex0(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := transaction.FlatTransaction{
		"Memos": []any{
			// First memo (index 0): valid NTT
			map[string]any{
				"Memo": map[string]any{
					"MemoFormat": testNTTMemoFormat,
					"MemoData":   sampleNTTMemoData,
				},
			},
			// Second memo: something else
			map[string]any{
				"Memo": map[string]any{
					"MemoFormat": "746578742F706C61696E", // "text/plain"
					"MemoData":   "48656C6C6F",           // "Hello"
				},
			},
		},
	}

	memo, err := p.parseMemoData(tx)

	require.NoError(t, err)
	require.NotNil(t, memo, "Should find the valid NTT memo at index 0")
	assert.Equal(t, uint16(2), memo.recipientChain)
	assert.Equal(t, uint8(6), memo.fromDecimals)
}

func TestParseMemoData_MultipleMemos_ValidNotAtIndex0(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := transaction.FlatTransaction{
		"Memos": []any{
			// First memo (index 0): wrong format
			map[string]any{
				"Memo": map[string]any{
					"MemoFormat": "746578742F706C61696E", // "text/plain"
					"MemoData":   "48656C6C6F",           // "Hello"
				},
			},
			// Second memo (index 1): valid NTT — should be ignored
			map[string]any{
				"Memo": map[string]any{
					"MemoFormat": testNTTMemoFormat,
					"MemoData":   sampleNTTMemoData,
				},
			},
		},
	}

	memo, err := p.parseMemoData(tx)

	require.NoError(t, err)
	require.Nil(t, memo, "Should not find NTT memo when it is not at index 0")
}

func TestParseMemoData_MalformedMemoStructure(t *testing.T) {
	p := NewParser("", nil, nil)

	testCases := []struct {
		name string
		tx   transaction.FlatTransaction
	}{
		{
			name: "Memos not an array",
			tx: transaction.FlatTransaction{
				"Memos": "not an array",
			},
		},
		{
			name: "Memo wrapper not a map",
			tx: transaction.FlatTransaction{
				"Memos": []any{"not a map"},
			},
		},
		{
			name: "Memo field missing",
			tx: transaction.FlatTransaction{
				"Memos": []any{
					map[string]any{
						"NotMemo": map[string]any{},
					},
				},
			},
		},
		{
			name: "Memo not a map",
			tx: transaction.FlatTransaction{
				"Memos": []any{
					map[string]any{
						"Memo": "not a map",
					},
				},
			},
		},
		{
			name: "MemoFormat not a string",
			tx: transaction.FlatTransaction{
				"Memos": []any{
					map[string]any{
						"Memo": map[string]any{
							"MemoFormat": 12345,
							"MemoData":   sampleNTTMemoData,
						},
					},
				},
			},
		},
		{
			name: "MemoData not a string",
			tx: transaction.FlatTransaction{
				"Memos": []any{
					map[string]any{
						"Memo": map[string]any{
							"MemoFormat": testNTTMemoFormat,
							"MemoData":   12345,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			memo, err := p.parseMemoData(tc.tx)

			// Malformed structures should not cause errors, just return nil
			require.NoError(t, err)
			assert.Nil(t, memo)
		})
	}
}

// =============================================================================
// addressToEmitter tests
// =============================================================================

func TestAddressToEmitter_ValidAddress(t *testing.T) {
	p := NewParser("", nil, nil)

	// Standard XRPL r-address
	address := "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"

	emitter, err := p.addressToEmitter(address)

	require.NoError(t, err)

	// Emitter should be 32 bytes
	assert.Equal(t, 32, len(emitter))

	// First 12 bytes should be zeros (left-padding)
	for i := 0; i < 12; i++ {
		assert.Equal(t, byte(0), emitter[i], "Byte %d should be zero padding", i)
	}

	// Last 20 bytes should be non-zero (the account ID)
	hasNonZero := false
	for i := 12; i < 32; i++ {
		if emitter[i] != 0 {
			hasNonZero = true
			break
		}
	}
	assert.True(t, hasNonZero, "Account ID portion should have non-zero bytes")
}

func TestAddressToEmitter_InvalidAddress(t *testing.T) {
	p := NewParser("", nil, nil)

	testCases := []struct {
		name    string
		address string
	}{
		{"Empty string", ""},
		{"Invalid format", "not-an-xrpl-address"},
		{"Too short", "rN7n"},
		{"Invalid checksum", "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D0"}, // Changed last char
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := p.addressToEmitter(tc.address)
			assert.Error(t, err, "Should return error for invalid address")
		})
	}
}

func TestAddressToEmitter_ConsistentResults(t *testing.T) {
	p := NewParser("", nil, nil)
	address := "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"

	// Call twice and verify same result
	emitter1, err1 := p.addressToEmitter(address)
	emitter2, err2 := p.addressToEmitter(address)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, emitter1, emitter2, "Same address should produce same emitter")
}

func TestAddressToEmitter_DifferentAddresses(t *testing.T) {
	p := NewParser("", nil, nil)

	addr1 := "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"
	addr2 := "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh" // Genesis account

	emitter1, err1 := p.addressToEmitter(addr1)
	emitter2, err2 := p.addressToEmitter(addr2)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, emitter1, emitter2, "Different addresses should produce different emitters")
}

// =============================================================================
// parseDeliveredAmount tests - XRP
// =============================================================================

func TestParseDeliveredAmount_XRP_Valid(t *testing.T) {
	p := NewParser("", nil, nil)
	// For XRP, memo.fromDecimals must be 6
	memo := &memoData{fromDecimals: 6}

	testCases := []struct {
		name           string
		drops          string
		expectedAmount uint64
	}{
		{"1 XRP", "1000000", 1000000},
		{"0.1 XRP", "100000", 100000},
		{"10 XRP", "10000000", 10000000},
		{"1 drop", "1", 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			info, err := p.parseDeliveredAmount(tc.drops, memo)
			require.NoError(t, err)
			assert.Equal(t, tokenTypeXRP, int(info.tokenType))
			assert.Equal(t, tc.expectedAmount, info.amount)
			assert.Equal(t, uint8(6), info.fromDecimals)
			assert.Equal(t, [32]byte{}, info.sourceToken) // Zero address for XRP
		})
	}
}

func TestParseDeliveredAmount_XRP_Invalid(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 6}

	_, err := p.parseDeliveredAmount("not_a_number", memo)
	require.Error(t, err)
}

func TestParseDeliveredAmount_XRP_ValidatesFromDecimals(t *testing.T) {
	p := NewParser("", nil, nil)

	// Valid: memo fromDecimals == 6 for XRP
	memo := &memoData{fromDecimals: 6}
	info, err := p.parseDeliveredAmount("1000000", memo)
	require.NoError(t, err)
	assert.Equal(t, uint64(1000000), info.amount)

	// Invalid: memo fromDecimals != 6
	memo = &memoData{fromDecimals: 8}
	_, err = p.parseDeliveredAmount("1000000", memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fromDecimals mismatch for XRP")
}

func TestParseDeliveredAmount_RejectsZeroAmount(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 6}

	// XRP zero amount
	_, err := p.parseDeliveredAmount("0", memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zero amount transfers are not allowed")

	// Trust Line zero amount
	tokenAmount := map[string]any{
		"currency": "USD",
		"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"value":    "0",
	}
	_, err = p.parseDeliveredAmount(tokenAmount, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zero amount transfers are not allowed")
}

// =============================================================================
// parseDeliveredAmount tests - Trust Lines
// =============================================================================

func TestParseDeliveredAmount_TrustLine_AnyFromDecimalsValid(t *testing.T) {
	p := NewParser("", nil, nil)

	tokenAmount := map[string]any{
		"currency": "USD",
		"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"value":    "100",
	}

	// Trust Lines accept any fromDecimals (sender specifies arbitrarily)
	for _, decimals := range []uint8{0, 6, 8, 15} {
		memo := &memoData{fromDecimals: decimals}
		info, err := p.parseDeliveredAmount(tokenAmount, memo)
		require.NoError(t, err, "fromDecimals=%d should be valid", decimals)
		assert.Equal(t, uint8(decimals), info.fromDecimals)
	}
}

// TestParseDeliveredAmount_TrustLine_HighPrecisionWithLowFromDecimals proves that
// a high-precision Trust Line value (15 decimals in the string) can be successfully
// parsed when the sender specifies a lower fromDecimals in the memo.
//
// This test proves the fix is working: without passing fromDecimals to the parser,
// the code would try to parse at 15 decimals, causing uint64 overflow.
func TestParseDeliveredAmount_TrustLine_HighPrecisionWithLowFromDecimals(t *testing.T) {
	p := NewParser("", nil, nil)

	// Sender specifies 6 decimals in memo, even though the value string has 15 decimals
	memo := &memoData{fromDecimals: 6}

	// This value at 15 decimals = 1000000000123456789012345 (overflows uint64!)
	// This value at 6 decimals  = 1000000000123456 (fits in uint64)
	tokenAmount := map[string]any{
		"currency": "USD",
		"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"value":    "1000000000.123456789012345",
	}

	info, err := p.parseDeliveredAmount(tokenAmount, memo)
	require.NoError(t, err)
	assert.Equal(t, tokenTypeIssued, int(info.tokenType))
	assert.Equal(t, uint64(1000000000123456), info.amount)
	assert.Equal(t, uint8(6), info.fromDecimals)
}

func TestParseDeliveredAmount_TrustLine_Valid(t *testing.T) {
	p := NewParser("", nil, nil)
	// Sender specifies 6 decimals in memo
	memo := &memoData{fromDecimals: 6}

	tokenAmount := map[string]any{
		"currency": "USD",
		"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"value":    "100.5",
	}

	info, err := p.parseDeliveredAmount(tokenAmount, memo)
	require.NoError(t, err)
	assert.Equal(t, tokenTypeIssued, int(info.tokenType))
	// 100.5 at 6 decimals = 100500000
	assert.Equal(t, uint64(100500000), info.amount)
	assert.Equal(t, uint8(6), info.fromDecimals)
	// First byte should be token type
	assert.Equal(t, byte(tokenTypeIssued), info.sourceToken[0])
}

func TestParseDeliveredAmount_TrustLine_HexCurrency(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 0}

	// RLUSD hex representation
	tokenAmount := map[string]any{
		"currency": "524C555344000000000000000000000000000000",
		"issuer":   "rMxCKbEDwqr76QuheSUMdEGf4B9xJ8m5De",
		"value":    "1000",
	}

	info, err := p.parseDeliveredAmount(tokenAmount, memo)
	require.NoError(t, err)
	assert.Equal(t, tokenTypeIssued, int(info.tokenType))
	assert.Equal(t, uint64(1000), info.amount)
}

func TestParseDeliveredAmount_TrustLine_ScientificNotation(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 0}

	tokenAmount := map[string]any{
		"currency": "USD",
		"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"value":    "1.23e6",
	}

	info, err := p.parseDeliveredAmount(tokenAmount, memo)
	require.NoError(t, err)
	assert.Equal(t, uint64(1230000), info.amount)
	assert.Equal(t, uint8(0), info.fromDecimals)
}

func TestParseDeliveredAmount_TrustLine_InvalidIssuer(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 6}

	tokenAmount := map[string]any{
		"currency": "USD",
		"issuer":   "not_a_valid_address",
		"value":    "100",
	}

	_, err := p.parseDeliveredAmount(tokenAmount, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to calculate trust line source token")
}

func TestParseDeliveredAmount_TrustLine_XRPCurrencyDisallowed(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 6}

	tokenAmount := map[string]any{
		"currency": "XRP",
		"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"value":    "100",
	}

	_, err := p.parseDeliveredAmount(tokenAmount, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "XRP is not a valid currency code")
}

func TestParseDeliveredAmount_TrustLine(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 0}

	// Trust Line delivered as object
	tokenAmount := map[string]any{
		"currency": "USD",
		"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"value":    "100",
	}

	info, err := p.parseDeliveredAmount(tokenAmount, memo)
	require.NoError(t, err)
	assert.Equal(t, tokenTypeIssued, int(info.tokenType))
}

func TestParseDeliveredAmount_UnexpectedType(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 6}

	// Passing an integer will fail during JSON unmarshaling
	_, err := p.parseDeliveredAmount(12345, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestParseDeliveredAmount_DispatchToTrustLine(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 2}

	// Token amount without mpt_issuance_id is a Trust Line
	tokenAmount := map[string]any{
		"currency": "EUR",
		"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"value":    "50.25",
	}

	info, err := p.parseDeliveredAmount(tokenAmount, memo)
	require.NoError(t, err)
	assert.Equal(t, tokenTypeIssued, int(info.tokenType))
	assert.Equal(t, uint64(5025), info.amount)
	assert.Equal(t, uint8(2), info.fromDecimals)
}

// =============================================================================
// scaleAmount tests
// =============================================================================

func TestScaleAmount_NoScaling(t *testing.T) {
	p := NewParser("", nil, nil)

	// fromDecimals = 6, toDecimals = 8, max = 8
	// result = min(min(8, 6), 8) = 6
	amount, decimals := p.scaleAmount(1000000, 6, 8)
	assert.Equal(t, uint64(1000000), amount)
	assert.Equal(t, uint8(6), decimals)
}

func TestScaleAmount_ScaleDown(t *testing.T) {
	p := NewParser("", nil, nil)

	// fromDecimals = 18, toDecimals = 8, max = 8
	// result = min(min(8, 18), 8) = 8
	// Need to scale from 18 to 8 decimals (divide by 10^10)
	amount, decimals := p.scaleAmount(1000000000000000000, 18, 8)
	assert.Equal(t, uint64(100000000), amount)
	assert.Equal(t, uint8(8), decimals)
}

func TestScaleAmount_ScaleToLowerToDecimals(t *testing.T) {
	p := NewParser("", nil, nil)

	// fromDecimals = 8, toDecimals = 4, max = 8
	// result = min(min(8, 8), 4) = 4
	// Scale from 8 to 4 (divide by 10^4)
	amount, decimals := p.scaleAmount(100000000, 8, 4)
	assert.Equal(t, uint64(10000), amount)
	assert.Equal(t, uint8(4), decimals)
}

func TestScaleAmount_MaxDecimals(t *testing.T) {
	p := NewParser("", nil, nil)

	// fromDecimals = 10, toDecimals = 10, max = 8
	// result = min(min(8, 10), 10) = 8
	amount, decimals := p.scaleAmount(10000000000, 10, 10)
	assert.Equal(t, uint64(100000000), amount)
	assert.Equal(t, uint8(8), decimals)
}

func TestScaleAmount_LargeAmount(t *testing.T) {
	p := NewParser("", nil, nil)

	// Large amount that doesn't overflow (just scales down)
	// targetDecimals = min(min(8, 18), 6) = 6
	// Scale from 18 to 6: divide by 10^12
	amount, decimals := p.scaleAmount(math.MaxUint64, 18, 6)
	// math.MaxUint64 = 18446744073709551615
	// Divided by 10^12 = 18446744
	assert.Equal(t, uint64(18446744), amount)
	assert.Equal(t, uint8(6), decimals)
}

func TestScaleAmount_ZeroAmount(t *testing.T) {
	p := NewParser("", nil, nil)

	amount, decimals := p.scaleAmount(0, 6, 8)
	assert.Equal(t, uint64(0), amount)
	assert.Equal(t, uint8(6), decimals)
}

func TestScaleAmount_AllMaxDecimals(t *testing.T) {
	p := NewParser("", nil, nil)

	// fromDecimals = 10, toDecimals = 10
	// target = min(min(8, 10), 10) = 8
	// scale from 10 to 8: divide by 100
	amount, decimals := p.scaleAmount(1234567890, 10, 10)
	assert.Equal(t, uint64(12345678), amount)
	assert.Equal(t, uint8(8), decimals)
}

func TestScaleAmount_ToZeroDecimals(t *testing.T) {
	p := NewParser("", nil, nil)

	// 0.01 represented with fromDecimals=21 means raw amount = 10^19
	// target = min(min(8, 21), 0) = 0
	// scale from 21 to 0: divide by 10^21
	// 10^19 / 10^21 = 0 (truncated)
	amount, decimals := p.scaleAmount(10000000000000000000, 21, 0)
	assert.Equal(t, uint64(0), amount)
	assert.Equal(t, uint8(0), decimals)
}

// =============================================================================
// calculateEmitterAddress tests
// =============================================================================

func TestCalculateEmitterAddress_Deterministic(t *testing.T) {
	p := NewParser("", nil, nil)

	var sourceNTTManager [32]byte
	var sourceToken [32]byte
	sourceNTTManager[31] = 0x01
	sourceToken[0] = tokenTypeXRP

	// Call twice and verify same result
	emitter1 := p.calculateEmitterAddress(sourceNTTManager, sourceToken)
	emitter2 := p.calculateEmitterAddress(sourceNTTManager, sourceToken)

	assert.Equal(t, emitter1, emitter2)
}

func TestCalculateEmitterAddress_DifferentTokens(t *testing.T) {
	p := NewParser("", nil, nil)

	var sourceNTTManager [32]byte
	sourceNTTManager[31] = 0x01

	var xrpToken [32]byte
	xrpToken[0] = tokenTypeXRP

	var issuedToken [32]byte
	issuedToken[0] = tokenTypeIssued
	issuedToken[1] = 0xAB

	emitter1 := p.calculateEmitterAddress(sourceNTTManager, xrpToken)
	emitter2 := p.calculateEmitterAddress(sourceNTTManager, issuedToken)

	assert.NotEqual(t, emitter1, emitter2, "Different tokens should produce different emitters")
}

// =============================================================================
// buildNTTPayload tests
// =============================================================================

func TestBuildNTTPayload_Structure(t *testing.T) {
	p := NewParser("", nil, nil)

	var sourceNTTManager [32]byte
	var recipientNTTManager [32]byte
	var sender [32]byte
	var sourceToken [32]byte
	var recipientAddress [32]byte

	sourceNTTManager[31] = 0x01
	recipientNTTManager[31] = 0x02
	sender[31] = 0x03
	sourceToken[0] = tokenTypeXRP
	recipientAddress[31] = 0x04

	payload := p.buildNTTPayload(
		sourceNTTManager,
		recipientNTTManager,
		12345,
		sender,
		6,
		1000000,
		sourceToken,
		recipientAddress,
		2, // Ethereum
	)

	// Verify payload length
	assert.Equal(t, 217, len(payload))

	// Verify transceiver prefix
	assert.Equal(t, transceiverPrefix[:], payload[0:4])

	// Verify source NTT manager
	assert.Equal(t, sourceNTTManager[:], payload[4:36])

	// Verify recipient NTT manager
	assert.Equal(t, recipientNTTManager[:], payload[36:68])

	// Verify ntt_manager_payload_length
	payloadLen := binary.BigEndian.Uint16(payload[68:70])
	assert.Equal(t, uint16(145), payloadLen)

	// Verify payload_length (internal NTT payload length)
	internalPayloadLen := binary.BigEndian.Uint16(payload[134:136])
	assert.Equal(t, uint16(79), internalPayloadLen)

	// Verify NTT prefix in manager payload
	assert.Equal(t, nttPrefix[:], payload[136:140])

	// Verify decimals
	assert.Equal(t, uint8(6), payload[140])

	// Verify amount
	amount := binary.BigEndian.Uint64(payload[141:149])
	assert.Equal(t, uint64(1000000), amount)

	// Verify recipient chain at the end of manager payload
	recipientChain := binary.BigEndian.Uint16(payload[213:215])
	assert.Equal(t, uint16(2), recipientChain)

	// Verify transceiver payload length is 0
	transceiverPayloadLen := binary.BigEndian.Uint16(payload[215:217])
	assert.Equal(t, uint16(0), transceiverPayloadLen)
}

// =============================================================================
// normalizeCurrency tests
// =============================================================================

func TestNormalizeCurrency_StandardCode(t *testing.T) {
	p := NewParser("", nil, nil)

	testCases := []struct {
		name     string
		currency string
	}{
		{"USD", "USD"},
		{"EUR", "EUR"},
		{"BTC", "BTC"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := p.normalizeCurrency(tc.currency)
			require.NoError(t, err)

			// First byte should be 0x00 for standard codes
			assert.Equal(t, byte(0x00), result[0])

			// Currency should be at bytes 12-14 (after leading zeros)
			assert.Equal(t, []byte(tc.currency), result[12:12+len(tc.currency)])
		})
	}
}

func TestNormalizeCurrency_HexCode(t *testing.T) {
	p := NewParser("", nil, nil)

	// RLUSD hex representation (40 chars)
	hexCurrency := "524C555344000000000000000000000000000000"

	result, err := p.normalizeCurrency(hexCurrency)
	require.NoError(t, err)

	// Should be decoded directly
	expected, _ := hex.DecodeString(hexCurrency)
	assert.Equal(t, expected, result[:])
}

func TestNormalizeCurrency_XRPDisallowed(t *testing.T) {
	p := NewParser("", nil, nil)

	_, err := p.normalizeCurrency("XRP")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "XRP is not a valid currency code")

	_, err = p.normalizeCurrency("xrp")
	require.Error(t, err)
}

func TestNormalizeCurrency_SingleCharCode(t *testing.T) {
	p := NewParser("", nil, nil)

	result, err := p.normalizeCurrency("X")
	require.NoError(t, err)
	assert.Equal(t, byte(0x00), result[0])
	assert.Equal(t, []byte("X"), result[12:13])
}

func TestNormalizeCurrency_InvalidHexLength(t *testing.T) {
	p := NewParser("", nil, nil)

	// 38 characters - invalid hex length (should be 40)
	_, err := p.normalizeCurrency("524C5553440000000000000000000000000000")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid standard currency code length")
}

func TestNormalizeCurrency_InvalidHexChars(t *testing.T) {
	p := NewParser("", nil, nil)

	// 40 characters but invalid hex
	_, err := p.normalizeCurrency("524C55534400000000000000000000000000ZZZZ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode hex currency")
}

func TestNormalizeCurrency_EmptyString(t *testing.T) {
	p := NewParser("", nil, nil)

	_, err := p.normalizeCurrency("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid standard currency code length")
}

// =============================================================================
// Trust Line source token calculation tests
// =============================================================================

func TestCalculateTrustLineSourceToken_Deterministic(t *testing.T) {
	p := NewParser("", nil, nil)

	token1, err1 := p.calculateTrustLineSourceToken("USD", "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh")
	token2, err2 := p.calculateTrustLineSourceToken("USD", "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, token1, token2)

	// First byte should be token type
	assert.Equal(t, byte(tokenTypeIssued), token1[0])
}

func TestCalculateTrustLineSourceToken_DifferentCurrencies(t *testing.T) {
	p := NewParser("", nil, nil)

	token1, err1 := p.calculateTrustLineSourceToken("USD", "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh")
	token2, err2 := p.calculateTrustLineSourceToken("EUR", "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, token1, token2)
}

func TestCalculateTrustLineSourceToken_DifferentIssuers(t *testing.T) {
	p := NewParser("", nil, nil)

	token1, err1 := p.calculateTrustLineSourceToken("USD", "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh")
	token2, err2 := p.calculateTrustLineSourceToken("USD", "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9")

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, token1, token2)
}

// =============================================================================
// MPT source token calculation tests
// =============================================================================

func TestCalculateMPTSourceToken(t *testing.T) {
	p := NewParser("", nil, nil)

	// Sample MPT issuance ID (24 bytes = 48 hex chars)
	mptID := "000000000000000000000000000000000000000000000001"

	token, err := p.calculateMPTSourceToken(mptID)
	require.NoError(t, err)

	// First byte should be token type
	assert.Equal(t, byte(tokenTypeMPT), token[0])

	// The MPT ID should be in the last 24 bytes (after 1 byte prefix + 7 bytes padding)
	expectedID, _ := hex.DecodeString(mptID)
	assert.Equal(t, expectedID, token[8:32])
}

func TestCalculateMPTSourceToken_InvalidLength(t *testing.T) {
	p := NewParser("", nil, nil)

	// Too short
	_, err := p.calculateMPTSourceToken("0001")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid mpt_issuance_id length")
}

// =============================================================================
// parseDecimalToUint64 tests
// =============================================================================

func TestParseDecimalToUint64_Integers(t *testing.T) {
	p := NewParser("", nil, nil)

	// Parse 1000000 with 0 target decimals
	amount, err := p.parseDecimalToUint64("1000000", 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(1000000), amount)

	// Parse 1000000 with 6 target decimals
	amount, err = p.parseDecimalToUint64("1000000", 6)
	require.NoError(t, err)
	assert.Equal(t, uint64(1000000000000), amount)
}

func TestParseDecimalToUint64_Decimals(t *testing.T) {
	p := NewParser("", nil, nil)

	// Parse 123.456 with 3 target decimals -> 123456
	amount, err := p.parseDecimalToUint64("123.456", 3)
	require.NoError(t, err)
	assert.Equal(t, uint64(123456), amount)

	// Parse 123.456 with 6 target decimals -> 123456000
	amount, err = p.parseDecimalToUint64("123.456", 6)
	require.NoError(t, err)
	assert.Equal(t, uint64(123456000), amount)

	// Parse 123.456789 with 3 target decimals -> 123456 (truncated)
	amount, err = p.parseDecimalToUint64("123.456789", 3)
	require.NoError(t, err)
	assert.Equal(t, uint64(123456), amount)
}

func TestParseDecimalToUint64_ScientificNotation(t *testing.T) {
	p := NewParser("", nil, nil)

	// 1.23e6 = 1230000 (lowercase e)
	amount, err := p.parseDecimalToUint64("1.23e6", 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(1230000), amount)

	// 1.23E6 = 1230000 (uppercase E, per XRPL docs both are valid)
	amount, err = p.parseDecimalToUint64("1.23E6", 0)
	require.NoError(t, err)
	assert.Equal(t, uint64(1230000), amount)

	// 1.5e-3 = 0.0015 with 4 decimals -> 15
	amount, err = p.parseDecimalToUint64("1.5e-3", 4)
	require.NoError(t, err)
	assert.Equal(t, uint64(15), amount)

	// 1.5E-3 = 0.0015 with 4 decimals -> 15 (uppercase E)
	amount, err = p.parseDecimalToUint64("1.5E-3", 4)
	require.NoError(t, err)
	assert.Equal(t, uint64(15), amount)
}

func TestParseDecimalToUint64_Negative(t *testing.T) {
	p := NewParser("", nil, nil)

	_, err := p.parseDecimalToUint64("-100", 6)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "negative values not allowed")
}

func TestParseDecimalToUint64_Zero(t *testing.T) {
	p := NewParser("", nil, nil)

	amount, err := p.parseDecimalToUint64("0", 6)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), amount)
}

func TestParseDecimalToUint64_Overflow(t *testing.T) {
	p := NewParser("", nil, nil)

	// This would overflow if parsed at 15 decimals first, but not at 6
	// A value like 999999999999.123456789012345 at 15 decimals would overflow
	_, err := p.parseDecimalToUint64("99999999999999999999", 6)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds uint64 max")
}

func TestParseDecimalToUint64_HighPrecisionNoOverflow(t *testing.T) {
	p := NewParser("", nil, nil)

	// A high-precision value that would overflow at natural precision (15 decimals)
	// but fits fine at 6 decimals.
	//
	// At 15 decimals: 1000000000.123456789012345 * 10^15 = 1000000000123456789012345
	// uint64 max:                                          18446744073709551615
	// This would OVERFLOW at 15 decimals!
	//
	// At 6 decimals: 1000000000.123456789012345 * 10^6 = 1000000000123456
	// This fits comfortably in uint64.
	amount, err := p.parseDecimalToUint64("1000000000.123456789012345", 6)
	require.NoError(t, err)
	assert.Equal(t, uint64(1000000000123456), amount)

	// Verify that parsing at 15 decimals would fail (proving the fix is necessary)
	_, err = p.parseDecimalToUint64("1000000000.123456789012345", 15)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "exceeds uint64 max")
}

func TestParseDecimalToUint64_Invalid(t *testing.T) {
	p := NewParser("", nil, nil)

	_, err := p.parseDecimalToUint64("not_a_number", 6)
	require.Error(t, err)
}

// =============================================================================
// validateTransactionType tests
// =============================================================================

func TestValidateTransactionType_Payment(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := transaction.FlatTransaction{
		"TransactionType": "Payment",
	}

	err := p.validateTransactionType(tx)
	require.NoError(t, err)
}

func TestValidateTransactionType_NotPayment(t *testing.T) {
	p := NewParser("", nil, nil)

	nonPaymentTypes := []string{
		"OfferCreate",
		"TrustSet",
		"AccountSet",
		"EscrowCreate",
	}

	for _, txType := range nonPaymentTypes {
		t.Run(txType, func(t *testing.T) {
			tx := transaction.FlatTransaction{
				"TransactionType": txType,
			}

			err := p.validateTransactionType(tx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "not Payment")
		})
	}
}

func TestValidateTransactionType_Missing(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := transaction.FlatTransaction{
		"Account": "rSomeAccount",
	}

	err := p.validateTransactionType(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no TransactionType field")
}

// =============================================================================
// validateTransactionResult tests
// =============================================================================

func TestValidateTransactionResult_Success(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := GenericTx{
		MetaTransactionResult: "tesSUCCESS",
	}

	err := p.validateTransactionResult(tx)
	require.NoError(t, err)
}

func TestValidateTransactionResult_Failed(t *testing.T) {
	p := NewParser("", nil, nil)

	failureCodes := []string{
		"tecUNFUNDED_PAYMENT",
		"tecNO_DST",
		"tecNO_DST_INSUF_XRP",
		"tecPATH_DRY",
		"tefPAST_SEQ",
	}

	for _, code := range failureCodes {
		t.Run(code, func(t *testing.T) {
			tx := GenericTx{
				MetaTransactionResult: code,
			}

			err := p.validateTransactionResult(tx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), code)
		})
	}
}

func TestValidateTransactionResult_EmptyResult(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := GenericTx{
		MetaTransactionResult: "",
	}

	// Empty result should fail (not tesSUCCESS)
	err := p.validateTransactionResult(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not tesSUCCESS")
}

// =============================================================================
// extractSender tests
// =============================================================================

func TestExtractSender_Valid(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := transaction.FlatTransaction{
		"Account": "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
	}

	sender, err := p.extractSender(tx)
	require.NoError(t, err)

	// First 12 bytes should be zero (left-padding)
	// sender is [32]byte (vaa.Address), so indices 0-11 are always valid
	for i := range 12 {
		assert.Equal(t, byte(0), sender[i]) //nolint:gosec // sender is [32]byte, index always in bounds
	}

	// Should have non-zero bytes in the account ID portion
	hasNonZero := false
	for i := 12; i < 32; i++ {
		if sender[i] != 0 {
			hasNonZero = true
			break
		}
	}
	assert.True(t, hasNonZero)
}

func TestExtractSender_MissingAccount(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := transaction.FlatTransaction{
		"Destination": "rSomeDestination",
	}

	_, err := p.extractSender(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Account field")
}

func TestExtractSender_InvalidAccount(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := transaction.FlatTransaction{
		"Account": "not_a_valid_address",
	}

	_, err := p.extractSender(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert sender address")
}

func TestExtractSender_AccountNotString(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := transaction.FlatTransaction{
		"Account": 12345,
	}

	_, err := p.extractSender(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Account field is not a string")
}

// =============================================================================
// extractDestination tests
// =============================================================================

func TestExtractDestination_Valid(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := transaction.FlatTransaction{
		"Destination": "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	dest, err := p.extractDestination(tx)
	require.NoError(t, err)
	assert.Equal(t, "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9", dest)
}

func TestExtractDestination_Missing(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := transaction.FlatTransaction{
		"Account": "rSomeAccount",
	}

	_, err := p.extractDestination(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Destination field")
}

func TestExtractDestination_NotString(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := transaction.FlatTransaction{
		"Destination": 12345,
	}

	_, err := p.extractDestination(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Destination field is not a string")
}

// =============================================================================
// ParseTransactionStream tests
// =============================================================================

func TestParseTransactionStream_ValidTransaction(t *testing.T) {
	p := NewParser("", nil, nil)

	// Create a mock TransactionStream matching real XRPL transaction structure
	txHash := "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D"
	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  createValidNTTTransaction(),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  7,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000", // XRP drops as string
		},
	}

	msg, err := p.ParseTransactionStream(tx)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify TxID
	expectedTxHash, _ := hex.DecodeString(txHash)
	assert.Equal(t, expectedTxHash, msg.TxID)

	// Verify sequence encoding: (ledgerIndex << 32) | txIndex
	expectedSequence := (uint64(12345) << 32) | 7
	assert.Equal(t, expectedSequence, msg.Sequence)

	// Verify chain
	assert.Equal(t, vaa.ChainIDXRPL, msg.EmitterChain)

	// Verify consistency level (finalized)
	assert.Equal(t, uint8(0), msg.ConsistencyLevel)

	// Verify not a reobservation
	assert.False(t, msg.IsReobservation)

	// Verify timestamp
	expectedTime, _ := time.Parse(time.RFC3339, "2024-01-15T10:30:00Z")
	assert.Equal(t, expectedTime, msg.Timestamp)

	// Verify payload is present and has correct length
	assert.NotNil(t, msg.Payload)
	assert.Equal(t, 217, len(msg.Payload))
}

func TestParseTransactionStream_NoNTTMemo(t *testing.T) {
	p := NewParser("", nil, nil)
	contract := "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"

	// Transaction with valid meta and destination but no Memos (no NTT payload)
	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rSomeOtherAccount",
			"Destination":     contract,
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
		},
	}

	msg, err := p.ParseTransactionStream(tx)

	require.NoError(t, err)
	assert.Nil(t, msg, "Should return nil for transaction without NTT memo")
}

func TestParseTransactionStream_InvalidTimestamp(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "not-a-valid-timestamp",
		Validated:    true,
		Transaction:  createValidNTTTransaction(),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)

	assert.Error(t, err, "Should return error for invalid timestamp")
}

func TestParseTransactionStream_InvalidTxHash(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := &streamtypes.TransactionStream{
		Hash:         "NOT_VALID_HEX!!!",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  createValidNTTTransaction(),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)

	assert.Error(t, err, "Should return error for invalid tx hash")
}

func TestParseTransactionStream_NotPaymentType(t *testing.T) {
	p := NewParser("", nil, nil)

	validTx := createValidNTTTransaction()
	validTx["TransactionType"] = "OfferCreate" // Not a Payment

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  validTx,
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)

	assert.Error(t, err, "Should return error for non-Payment transaction")
	assert.Contains(t, err.Error(), "not Payment")
}

func TestParseTransactionStream_FailsOnNonSuccessResult(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   sampleNTTMemoData,
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionResult: "tecUNFUNDED_PAYMENT",
		},
	}

	msg, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "tecUNFUNDED_PAYMENT")
}

// =============================================================================
// Sequence encoding tests
// =============================================================================

func TestSequenceEncoding(t *testing.T) {
	testCases := []struct {
		name             string
		ledgerIndex      uint64
		txIndex          uint64
		expectedSequence uint64
	}{
		{"zero values", 0, 0, 0},
		{"ledger only", 1, 0, 1 << 32},
		{"tx index only", 0, 5, 5},
		{"both values", 12345, 7, (12345 << 32) | 7},
		{"max ledger index", 0xFFFFFFFF, 0, 0xFFFFFFFF << 32},
		{"max tx index", 0, 0xFFFFFFFF, 0xFFFFFFFF},
		{"both max", 0xFFFFFFFF, 0xFFFFFFFF, (uint64(0xFFFFFFFF) << 32) | 0xFFFFFFFF},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			sequence := (tc.ledgerIndex << 32) | tc.txIndex
			assert.Equal(t, tc.expectedSequence, sequence)

			// Verify we can extract both values back
			extractedLedger := sequence >> 32
			extractedTxIndex := sequence & 0xFFFFFFFF
			assert.Equal(t, tc.ledgerIndex, extractedLedger)
			assert.Equal(t, tc.txIndex, extractedTxIndex)
		})
	}
}

// =============================================================================
// Integration tests: Full transaction flow
// =============================================================================

func TestParseTransactionStream_XRPPayment_FullFlow(t *testing.T) {
	p := NewParser("", nil, nil)

	// Create memo with Ethereum as recipient chain, fromDecimals=6 (XRP), toDecimals=8
	memoData := createSampleMemoData(2, 6, 8)

	tx := &streamtypes.TransactionStream{
		Hash:         "ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
		LedgerIndex:  99999,
		CloseTimeISO: "2024-06-15T14:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   memoData,
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  15,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "5000000", // 5 XRP in drops
		},
	}

	msg, err := p.ParseTransactionStream(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify sequence encoding
	expectedSequence := (uint64(99999) << 32) | 15
	assert.Equal(t, expectedSequence, msg.Sequence)

	// Verify chain
	assert.Equal(t, vaa.ChainIDXRPL, msg.EmitterChain)

	// Verify payload
	assert.Equal(t, 217, len(msg.Payload))

	// Verify transceiver prefix in payload
	assert.Equal(t, transceiverPrefix[:], msg.Payload[0:4])

	// Verify decimals in payload (at offset 140)
	// For XRP: min(min(8, 6), 8) = 6
	assert.Equal(t, uint8(6), msg.Payload[140])

	// Verify amount in payload (at offset 141)
	amountInPayload := binary.BigEndian.Uint64(msg.Payload[141:149])
	assert.Equal(t, uint64(5000000), amountInPayload)
}

func TestParseTransactionStream_TrustLinePayment_FullFlow(t *testing.T) {
	p := NewParser("", nil, nil)

	// Create memo with Solana as recipient chain, fromDecimals=6, toDecimals=9
	memoData := createSampleMemoData(1, 6, 9)

	tx := &streamtypes.TransactionStream{
		Hash:         "1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF",
		LedgerIndex:  50000,
		CloseTimeISO: "2024-07-20T08:00:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   memoData,
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  7,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount: map[string]any{
				"currency": "USD",
				"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
				"value":    "250.123456", // 6 decimals
			},
		},
	}

	msg, err := p.ParseTransactionStream(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify payload structure
	assert.Equal(t, 217, len(msg.Payload))

	// Verify decimals: min(min(8, 6), 9) = 6
	assert.Equal(t, uint8(6), msg.Payload[140])

	// Verify amount: 250123456 (value * 10^6)
	amountInPayload := binary.BigEndian.Uint64(msg.Payload[141:149])
	assert.Equal(t, uint64(250123456), amountInPayload)

	// Verify source token starts with tokenTypeIssued
	assert.Equal(t, byte(tokenTypeIssued), msg.Payload[149])
}

func TestParseTransactionStream_FromDecimalsMismatch(t *testing.T) {
	p := NewParser("", nil, nil)

	// Create memo with wrong fromDecimals (8 instead of 6 for XRP)
	memoData := createSampleMemoData(2, 8, 8)

	tx := &streamtypes.TransactionStream{
		Hash:         "ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   memoData,
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000", // XRP
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fromDecimals mismatch for XRP")
}

// =============================================================================
// ParseTxResponse tests (reobservation path)
// =============================================================================

func TestParseTxResponse_ValidTransaction(t *testing.T) {
	p := NewParser("", nil, nil)

	txHash := "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D"
	tx := &transactions.TxResponse{
		Hash:        types.Hash256(txHash),
		LedgerIndex: 12345,
		Date:        784111200, // Ripple epoch timestamp (seconds since 2000-01-01)
		Validated:   true,
		TxJSON:      createValidNTTTransaction(),
		Meta: transaction.TxMetadataBuilder{
			TransactionIndex:  7,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000", // XRP drops as string
		},
	}

	msg, err := p.ParseTxResponse(tx)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify TxID
	expectedTxHash, _ := hex.DecodeString(txHash)
	assert.Equal(t, expectedTxHash, msg.TxID)

	// Verify sequence encoding: (ledgerIndex << 32) | txIndex
	expectedSequence := (uint64(12345) << 32) | 7
	assert.Equal(t, expectedSequence, msg.Sequence)

	// Verify chain
	assert.Equal(t, vaa.ChainIDXRPL, msg.EmitterChain)

	// Verify timestamp conversion (Ripple epoch + offset = Unix timestamp)
	// 784111200 + 946684800 = 1730796000 = 2024-11-05T06:00:00Z
	expectedTime := time.Unix(784111200+946684800, 0)
	assert.Equal(t, expectedTime, msg.Timestamp)
}

func TestParseTxResponse_NoNTTMemo(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := &transactions.TxResponse{
		Hash:        types.Hash256("8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D"),
		LedgerIndex: 12345,
		Date:        784111200,
		Validated:   true,
		TxJSON: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
		},
	}

	msg, err := p.ParseTxResponse(tx)

	require.NoError(t, err)
	assert.Nil(t, msg, "Should return nil for transaction without NTT memo")
}

func TestParseTxResponse_DateOverflow(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := &transactions.TxResponse{
		Hash:        types.Hash256("8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D"),
		LedgerIndex: 12345,
		Date:        math.MaxInt64, // This would overflow when adding rippleEpochOffset
		Validated:   true,
		TxJSON:      createValidNTTTransaction(),
		Meta: transaction.TxMetadataBuilder{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTxResponse(tx)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "would overflow")
}

// =============================================================================
// parseMPTCurrencyAmount tests
// =============================================================================

func TestParseMPTCurrencyAmount_Valid(t *testing.T) {
	// Mock MPT asset scale fetcher
	mockFetcher := func(mptID string) (uint8, error) {
		return 6, nil // Return asset scale of 6
	}
	p := NewParser("", nil, mockFetcher)

	// MPT delivered amount
	mptAmount := map[string]any{
		"mpt_issuance_id": "000000000000000000000000000000000000000000000001",
		"value":           "1000000",
	}
	memo := &memoData{fromDecimals: 6}

	info, err := p.parseDeliveredAmount(mptAmount, memo)
	require.NoError(t, err)
	assert.Equal(t, tokenTypeMPT, int(info.tokenType))
	assert.Equal(t, uint64(1000000), info.amount)
	assert.Equal(t, uint8(6), info.fromDecimals)
	assert.Equal(t, byte(tokenTypeMPT), info.sourceToken[0])
}

func TestParseMPTCurrencyAmount_InvalidValue(t *testing.T) {
	mockFetcher := func(mptID string) (uint8, error) {
		return 6, nil
	}
	p := NewParser("", nil, mockFetcher)

	mptAmount := map[string]any{
		"mpt_issuance_id": "000000000000000000000000000000000000000000000001",
		"value":           "not_a_number",
	}
	memo := &memoData{fromDecimals: 6}

	_, err := p.parseDeliveredAmount(mptAmount, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse MPT value")
}

func TestParseMPTCurrencyAmount_FetchError(t *testing.T) {
	mockFetcher := func(mptID string) (uint8, error) {
		return 0, fmt.Errorf("network error")
	}
	p := NewParser("", nil, mockFetcher)

	mptAmount := map[string]any{
		"mpt_issuance_id": "000000000000000000000000000000000000000000000001",
		"value":           "1000000",
	}
	memo := &memoData{fromDecimals: 6}

	_, err := p.parseDeliveredAmount(mptAmount, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to fetch MPT asset scale")
}

func TestParseMPTCurrencyAmount_DecimalsMismatch(t *testing.T) {
	mockFetcher := func(mptID string) (uint8, error) {
		return 8, nil // Return asset scale of 8, but memo says 6
	}
	p := NewParser("", nil, mockFetcher)

	mptAmount := map[string]any{
		"mpt_issuance_id": "000000000000000000000000000000000000000000000001",
		"value":           "1000000",
	}
	memo := &memoData{fromDecimals: 6}

	_, err := p.parseDeliveredAmount(mptAmount, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "fromDecimals mismatch for MPT")
}

func TestParseMPTCurrencyAmount_ZeroAmount(t *testing.T) {
	mockFetcher := func(mptID string) (uint8, error) {
		return 6, nil
	}
	p := NewParser("", nil, mockFetcher)

	mptAmount := map[string]any{
		"mpt_issuance_id": "000000000000000000000000000000000000000000000001",
		"value":           "0",
	}
	memo := &memoData{fromDecimals: 6}

	_, err := p.parseDeliveredAmount(mptAmount, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "zero amount transfers are not allowed")
}

// =============================================================================
// Additional edge case tests for 100% coverage
// =============================================================================

func TestValidateTransactionType_NotString(t *testing.T) {
	p := NewParser("", nil, nil)
	tx := transaction.FlatTransaction{
		"TransactionType": 12345, // Not a string
	}

	err := p.validateTransactionType(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "TransactionType field is not a string")
}

func TestParseTransaction_ScaledAmountZero(t *testing.T) {
	p := NewParser("", nil, nil)

	// Create memo with high toDecimals but very small amount
	// Amount of 1 drop (1e-6 XRP) scaled to 0 decimals becomes 0
	memoData := createSampleMemoData(2, 6, 0) // toDecimals=0 means we divide by 10^6

	tx := &streamtypes.TransactionStream{
		Hash:         "ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   memoData,
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1", // 1 drop, will become 0 when scaled to 0 decimals
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "scaled amount is zero")
}

func TestCalculateMPTSourceToken_InvalidHex(t *testing.T) {
	p := NewParser("", nil, nil)

	// Invalid hex characters
	_, err := p.calculateMPTSourceToken("ZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZZ")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode mpt_issuance_id")
}

func TestParseTransactionStream_InvalidDestination(t *testing.T) {
	p := NewParser("", nil, nil)

	validTx := createValidNTTTransaction()
	validTx["Destination"] = "invalid_address" // Invalid XRPL address

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  validTx,
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert source NTT manager address")
}

func TestParseTransactionStream_InvalidSender(t *testing.T) {
	p := NewParser("", nil, nil)

	validTx := createValidNTTTransaction()
	validTx["Account"] = "invalid_sender_address" // Invalid XRPL address

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  validTx,
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to convert sender address")
}

func TestParseTransactionStream_MissingDestination(t *testing.T) {
	p := NewParser("", nil, nil)

	validTx := createValidNTTTransaction()
	delete(validTx, "Destination") // Remove destination

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  validTx,
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Destination field")
}

func TestParseTransactionStream_MissingSender(t *testing.T) {
	p := NewParser("", nil, nil)

	validTx := createValidNTTTransaction()
	delete(validTx, "Account") // Remove account

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  validTx,
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no Account field")
}

func TestParseTransactionStream_InvalidDeliveredAmount(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  createValidNTTTransaction(),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "not_a_valid_amount", // Invalid amount format
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse delivered amount")
}

func TestParseIssuedCurrencyAmount_InvalidValue(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 6}

	// A trust line with an invalid value that will fail to parse
	tokenAmount := map[string]any{
		"currency": "USD",
		"issuer":   "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"value":    "not_a_number",
	}

	_, err := p.parseDeliveredAmount(tokenAmount, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse trust line value")
}

func TestParseMPTCurrencyAmount_InvalidSourceToken(t *testing.T) {
	mockFetcher := func(mptID string) (uint8, error) {
		return 6, nil
	}
	p := NewParser("", nil, mockFetcher)

	// MPT with invalid mpt_issuance_id (wrong length)
	mptAmount := map[string]any{
		"mpt_issuance_id": "0001", // Too short - should be 48 hex chars
		"value":           "1000000",
	}
	memo := &memoData{fromDecimals: 6}

	_, err := p.parseDeliveredAmount(mptAmount, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to calculate MPT source token")
}

func TestParseTransactionStream_MPTPayment_FullFlow(t *testing.T) {
	// Mock MPT asset scale fetcher
	mockFetcher := func(mptID string) (uint8, error) {
		return 8, nil // Return asset scale of 8
	}
	p := NewParser("", nil, mockFetcher)

	// Create memo with fromDecimals matching the MPT asset scale
	memoData := createSampleMemoData(2, 8, 8) // fromDecimals=8 matches MPT AssetScale

	tx := &streamtypes.TransactionStream{
		Hash:         "ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   memoData,
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount: map[string]any{
				"mpt_issuance_id": "000000000000000000000000000000000000000000000001",
				"value":           "100000000", // 1 token with 8 decimals
			},
		},
	}

	msg, err := p.ParseTransactionStream(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify payload
	assert.Equal(t, 217, len(msg.Payload))

	// Verify decimals: min(min(8, 8), 8) = 8
	assert.Equal(t, uint8(8), msg.Payload[140])

	// Verify source token starts with tokenTypeMPT
	assert.Equal(t, byte(tokenTypeMPT), msg.Payload[149])
}

// =============================================================================
// Additional tests for remaining edge cases
// =============================================================================

func TestParseDeliveredAmount_MarshalError(t *testing.T) {
	p := NewParser("", nil, nil)
	memo := &memoData{fromDecimals: 6}

	// Create a value that can't be marshaled to JSON properly
	// Using a channel which can't be JSON marshaled
	ch := make(chan int)
	_, err := p.parseDeliveredAmount(ch, memo)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to marshal delivered amount")
}

func TestParseTransaction_ParseMemoError(t *testing.T) {
	p := NewParser("", nil, nil)

	// Create a transaction with an NTT memo format but invalid hex in MemoData
	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   "INVALID_HEX!!!", // Invalid hex
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode MemoData")
}

func TestParseTransaction_ValidationResultError(t *testing.T) {
	p := NewParser("", nil, nil)

	// Create transaction with NTT memo but failed result
	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]any{
				"TransactionResult": "tecUNFUNDED",
			},
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   sampleNTTMemoData,
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tecUNFUNDED",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tecUNFUNDED")
}

func TestParseTransaction_ValidationTypeError(t *testing.T) {
	p := NewParser("", nil, nil)

	// Create transaction with NTT memo but non-Payment type
	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "OfferCreate",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]any{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   sampleNTTMemoData,
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not Payment")
}

func TestParseTransaction_TransactionIndexOverflow(t *testing.T) {
	p := NewParser("", nil, nil)

	// Create a valid transaction but with TransactionIndex > MaxUint32
	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testNTTMemoFormat,
						"MemoData":   sampleNTTMemoData,
					},
				},
			},
		},
		Meta: transaction.TxObjMeta{
			TransactionIndex:  math.MaxUint32 + 1, // Exceeds uint32 max
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := p.ParseTransactionStream(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid transaction index")
}

// =============================================================================
// Core message (generic Wormhole message) tests
// =============================================================================

// testCoreMemoFormat is hex-encoded: "application/x-wormhole-publish"
const testCoreMemoFormat = "6170706C69636174696F6E2F782D776F726D686F6C652D7075626C697368"

// testCoreAccount is a sample XRPL core account address
const testCoreAccount = "rPT1Sjq2YGrBMTttX4GZHjKu9dyfzbpAYe"

// createCoreMemoData creates a hex-encoded core memo: version(1) + nonce(4) + payload
func createCoreMemoData(version uint8, nonce uint32, payload []byte) string {
	data := make([]byte, 1+4+len(payload))
	data[0] = version
	binary.BigEndian.PutUint32(data[1:5], nonce)
	copy(data[5:], payload)
	return hex.EncodeToString(data)
}

func createFlatTransactionWithCoreMemo(memoData, destination string) transaction.FlatTransaction {
	return transaction.FlatTransaction{
		"TransactionType": "Payment",
		"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
		"Destination":     destination,
		"Memos": []any{
			map[string]any{
				"Memo": map[string]any{
					"MemoFormat": testCoreMemoFormat,
					"MemoData":   memoData,
				},
			},
		},
	}
}

func TestParseCoreMessageMemoData_Valid(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)
	payload := []byte("hello wormhole")
	memoData := createCoreMemoData(1, 42, payload)
	tx := createFlatTransactionWithMemos(testCoreMemoFormat, memoData)

	result, err := p.parseCoreMessageMemoData(tx)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, uint32(42), result.nonce)
	assert.Equal(t, payload, result.payload)
}

func TestParseCoreMessageMemoData_NoMemo(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)
	tx := transaction.FlatTransaction{
		"Account": "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
	}

	result, err := p.parseCoreMessageMemoData(tx)

	require.NoError(t, err)
	assert.Nil(t, result, "Should return nil when no matching memo")
}

func TestParseCoreMessageMemoData_WrongVersion(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)
	payload := []byte("test")
	memoData := createCoreMemoData(2, 0, payload) // version 2, not supported
	tx := createFlatTransactionWithMemos(testCoreMemoFormat, memoData)

	result, err := p.parseCoreMessageMemoData(tx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "unsupported core memo version")
}

func TestParseCoreMessageMemoData_TooShort(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)
	// Only 3 bytes — less than the required 5
	memoData := hex.EncodeToString([]byte{0x01, 0x00, 0x00})
	tx := createFlatTransactionWithMemos(testCoreMemoFormat, memoData)

	result, err := p.parseCoreMessageMemoData(tx)

	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "core memo data too short")
}

func TestParseCoreTransaction_Valid(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)
	payload := []byte("cross-chain message")
	memoData := createCoreMemoData(1, 99, payload)

	gtx := GenericTx{
		Transaction:           createFlatTransactionWithCoreMemo(memoData, testCoreAccount),
		Timestamp:             time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Hash:                  "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:           12345,
		MetaTransactionIndex:  7,
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseCoreTransaction(gtx)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify nonce
	assert.Equal(t, uint32(99), msg.Nonce)

	// Verify payload
	assert.Equal(t, payload, msg.Payload)

	// Verify emitter chain
	assert.Equal(t, vaa.ChainIDXRPL, msg.EmitterChain)

	// Verify emitter is sender (rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh), not core account
	senderEmitter, err := p.addressToEmitter("rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh")
	require.NoError(t, err)
	assert.Equal(t, senderEmitter, msg.EmitterAddress)

	// Verify sequence: (12345 << 32) | 7
	expectedSequence := (uint64(12345) << 32) | 7
	assert.Equal(t, expectedSequence, msg.Sequence)

	// Verify consistency level
	assert.Equal(t, uint8(0), msg.ConsistencyLevel)
}

func TestParseCoreTransaction_NotCoreAccount(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)
	payload := []byte("test")
	memoData := createCoreMemoData(1, 1, payload)

	gtx := GenericTx{
		Transaction:           createFlatTransactionWithCoreMemo(memoData, "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"), // not core account
		Timestamp:             time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Hash:                  "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:           12345,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseCoreTransaction(gtx)

	require.NoError(t, err)
	assert.Nil(t, msg, "Should return nil for payment not to core account")
}

func TestParseCoreTransaction_NonPayment(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)
	payload := []byte("test")
	memoData := createCoreMemoData(1, 1, payload)

	ftx := createFlatTransactionWithCoreMemo(memoData, testCoreAccount)
	ftx["TransactionType"] = "OfferCreate" // not a Payment

	gtx := GenericTx{
		Transaction:           ftx,
		Timestamp:             time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Hash:                  "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:           12345,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseCoreTransaction(gtx)

	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "not Payment")
}

func TestParseCoreTransaction_FailedResult(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)
	payload := []byte("test")
	memoData := createCoreMemoData(1, 1, payload)

	gtx := GenericTx{
		Transaction:           createFlatTransactionWithCoreMemo(memoData, testCoreAccount),
		Timestamp:             time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Hash:                  "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:           12345,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tecUNFUNDED_PAYMENT",
	}

	msg, err := p.parseCoreTransaction(gtx)

	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "tecUNFUNDED_PAYMENT")
}

func TestParseTransactionStream_CoreMessage(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)
	payload := []byte("end-to-end core message")
	memoData := createCoreMemoData(1, 123, payload)

	tx := &streamtypes.TransactionStream{
		Hash:         "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233",
		LedgerIndex:  50000,
		CloseTimeISO: "2024-06-01T12:00:00Z",
		Validated:    true,
		Transaction:  createFlatTransactionWithCoreMemo(memoData, testCoreAccount),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  3,
			TransactionResult: "tesSUCCESS",
		},
	}

	msg, err := p.ParseTransactionStream(tx)

	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify nonce from memo
	assert.Equal(t, uint32(123), msg.Nonce)

	// Verify payload from memo
	assert.Equal(t, payload, msg.Payload)

	// Verify emitter chain
	assert.Equal(t, vaa.ChainIDXRPL, msg.EmitterChain)

	// Verify sequence: (50000 << 32) | 3
	expectedSequence := (uint64(50000) << 32) | 3
	assert.Equal(t, expectedSequence, msg.Sequence)

	// Verify timestamp
	expectedTime, _ := time.Parse(time.RFC3339, "2024-06-01T12:00:00Z")
	assert.Equal(t, expectedTime, msg.Timestamp)

	// Verify consistency level
	assert.Equal(t, uint8(0), msg.ConsistencyLevel)

	// Verify TxID
	expectedTxHash, _ := hex.DecodeString("AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233")
	assert.Equal(t, expectedTxHash, msg.TxID)
}

// --- XTCF (Ticket Refill Confirmation) Tests ---

const testManagedAccount = "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh"

func createTicketCreateTx(account string, ticketSequences []float64) GenericTx {
	affectedNodes := make([]transaction.AffectedNode, 0, len(ticketSequences))
	for _, seq := range ticketSequences {
		affectedNodes = append(affectedNodes, transaction.AffectedNode{
			CreatedNode: &transaction.CreatedNode{
				LedgerEntryType: ledger.TicketEntry,
				NewFields: ledger.FlatLedgerObject{
					"TicketSequence": seq,
				},
			},
		})
	}

	return GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         account,
			"TicketCount":     float64(len(ticketSequences)),
		},
		Hash:                  "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233",
		LedgerIndex:           50000,
		MetaTransactionIndex:  3,
		MetaTransactionResult: "tesSUCCESS",
		MetaAffectedNodes:     affectedNodes,
		Timestamp:             time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
	}
}

func TestParseTicketCreateTransaction_Success(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := createTicketCreateTx(testManagedAccount, []float64{100, 101, 102, 103, 104})
	msg, err := p.parseTicketCreateTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify XTCF prefix
	assert.Equal(t, byte('X'), msg.Payload[0])
	assert.Equal(t, byte('T'), msg.Payload[1])
	assert.Equal(t, byte('C'), msg.Payload[2])
	assert.Equal(t, byte('F'), msg.Payload[3])

	// Verify payload length (20 bytes)
	assert.Equal(t, 20, len(msg.Payload))

	// ticket_start = 100
	ticketStart := binary.BigEndian.Uint64(msg.Payload[4:12])
	assert.Equal(t, uint64(100), ticketStart)

	// ticket_count = 5
	ticketCount := binary.BigEndian.Uint64(msg.Payload[12:20])
	assert.Equal(t, uint64(5), ticketCount)

	// Verify chain and sequence
	assert.Equal(t, vaa.ChainIDXRPL, msg.EmitterChain)
	expectedSequence := (uint64(50000) << 32) | 3
	assert.Equal(t, expectedSequence, msg.Sequence)
}

func TestParseTicketCreateTransaction_NotManagedAccount(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := createTicketCreateTx("rDifferentAccount1234567890123456", []float64{100})
	msg, err := p.parseTicketCreateTransaction(tx)
	assert.NoError(t, err)
	assert.Nil(t, msg)
}

func TestParseTicketCreateTransaction_NotTicketCreate(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         testManagedAccount,
		},
		MetaTransactionResult: "tesSUCCESS",
	}
	msg, err := p.parseTicketCreateTransaction(tx)
	assert.NoError(t, err)
	assert.Nil(t, msg)
}

func TestParseTicketCreateTransaction_FailedTx(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := createTicketCreateTx(testManagedAccount, []float64{100})
	tx.MetaTransactionResult = "tecNO_PERMISSION"

	msg, err := p.parseTicketCreateTransaction(tx)
	assert.Error(t, err)
	assert.Nil(t, msg)
}

func TestParseTicketCreateTransaction_UnsortedSequences(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	// Tickets may appear in any order in AffectedNodes
	tx := createTicketCreateTx(testManagedAccount, []float64{105, 102, 108, 101, 103})
	msg, err := p.parseTicketCreateTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// ticket_start should be the minimum = 101
	ticketStart := binary.BigEndian.Uint64(msg.Payload[4:12])
	assert.Equal(t, uint64(101), ticketStart)

	ticketCount := binary.BigEndian.Uint64(msg.Payload[12:20])
	assert.Equal(t, uint64(5), ticketCount)
}

func TestParseTicketCreateTransaction_NoManagedAccounts(t *testing.T) {
	p := NewParser("", nil, nil)

	tx := createTicketCreateTx(testManagedAccount, []float64{100})
	msg, err := p.parseTicketCreateTransaction(tx)
	assert.NoError(t, err)
	assert.Nil(t, msg)
}

func TestParseTicketCreateTransaction_DispatchedFromParseTransaction(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := createTicketCreateTx(testManagedAccount, []float64{200, 201, 202})
	msg, err := p.parseTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify it was dispatched correctly (XTCF prefix)
	assert.Equal(t, xtcfPrefix[:], msg.Payload[0:4])
}

func TestParseTxResponse_TicketCreateTimestamp(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	txHash := "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233"
	tx := &transactions.TxResponse{
		Hash:        types.Hash256(txHash),
		LedgerIndex: 50000,
		Date:        784111200, // Ripple epoch seconds (2024-11-05T06:00:00Z when converted)
		Validated:   true,
		TxJSON: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
			"TicketCount":     float64(3),
		},
		Meta: transaction.TxMetadataBuilder{
			TransactionIndex:  2,
			TransactionResult: "tesSUCCESS",
			AffectedNodes: []transaction.AffectedNode{
				{
					CreatedNode: &transaction.CreatedNode{
						LedgerEntryType: ledger.TicketEntry,
						NewFields:       ledger.FlatLedgerObject{"TicketSequence": float64(100)},
					},
				},
				{
					CreatedNode: &transaction.CreatedNode{
						LedgerEntryType: ledger.TicketEntry,
						NewFields:       ledger.FlatLedgerObject{"TicketSequence": float64(101)},
					},
				},
				{
					CreatedNode: &transaction.CreatedNode{
						LedgerEntryType: ledger.TicketEntry,
						NewFields:       ledger.FlatLedgerObject{"TicketSequence": float64(102)},
					},
				},
			},
		},
	}

	msg, err := p.ParseTxResponse(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// Verify timestamp: Ripple epoch 784111200 + offset 946684800 = Unix 1730796000
	expectedTime := time.Unix(784111200+rippleEpochOffset, 0)
	assert.Equal(t, expectedTime, msg.Timestamp)
	assert.False(t, msg.Timestamp.IsZero(), "timestamp should not be zero")

	// Verify the timestamp is not the Ripple epoch start (which would indicate Date=0)
	rippleEpochStart := time.Unix(rippleEpochOffset, 0) // 2000-01-01T00:00:00Z
	assert.NotEqual(t, rippleEpochStart, msg.Timestamp, "timestamp should not be the Ripple epoch start (Date=0)")

	// Verify XTCF payload
	assert.Equal(t, xtcfPrefix[:], msg.Payload[0:4])
	ticketStart := binary.BigEndian.Uint64(msg.Payload[4:12])
	assert.Equal(t, uint64(100), ticketStart)
	ticketCount := binary.BigEndian.Uint64(msg.Payload[12:20])
	assert.Equal(t, uint64(3), ticketCount)
}

// TestParseTxResponse_TicketCreateDateFallback verifies that when TxResponse.Date is 0
// (as happens with rippled API v2, which returns `date` inside `tx_json` rather than
// at the top level), ParseTxResponse falls back to reading `date` from TxJSON.
func TestParseTxResponse_TicketCreateDateFallback(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	txHash := "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233"
	rippleDate := float64(773452800) // 2024-06-30 in Ripple epoch seconds
	tx := &transactions.TxResponse{
		Hash:        types.Hash256(txHash),
		LedgerIndex: 50000,
		Date:        0, // Not populated in API v2 — date is inside tx_json instead
		Validated:   true,
		TxJSON: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
			"TicketCount":     float64(1),
			"date":            rippleDate,
		},
		Meta: transaction.TxMetadataBuilder{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			AffectedNodes: []transaction.AffectedNode{
				{
					CreatedNode: &transaction.CreatedNode{
						LedgerEntryType: ledger.TicketEntry,
						NewFields:       ledger.FlatLedgerObject{"TicketSequence": float64(50)},
					},
				},
			},
		},
	}

	msg, err := p.ParseTxResponse(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	// The fallback should read date from TxJSON and produce the correct timestamp
	expectedTimestamp := time.Unix(int64(rippleDate)+rippleEpochOffset, 0)
	assert.Equal(t, expectedTimestamp, msg.Timestamp,
		"should read date from TxJSON when TxResponse.Date is 0")
}

// TestParseTxResponse_DateZeroError verifies that ParseTxResponse returns an error
// when the date is zero in both TxResponse.Date and TxJSON.
func TestParseTxResponse_DateZeroError(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	txHash := "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233"
	tx := &transactions.TxResponse{
		Hash:        types.Hash256(txHash),
		LedgerIndex: 50000,
		Date:        0,
		Validated:   true,
		TxJSON: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
			"TicketCount":     float64(1),
		},
		Meta: transaction.TxMetadataBuilder{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			AffectedNodes: []transaction.AffectedNode{
				{
					CreatedNode: &transaction.CreatedNode{
						LedgerEntryType: ledger.TicketEntry,
						NewFields:       ledger.FlatLedgerObject{"TicketSequence": float64(50)},
					},
				},
			},
		},
	}

	msg, err := p.ParseTxResponse(tx)
	require.Error(t, err)
	require.Nil(t, msg)
	assert.Contains(t, err.Error(), "date is zero")
}

// =============================================================================
// Additional coverage tests
// =============================================================================

// TestParseTxResponse_DateFallbackJsonNumber tests the json.Number date fallback path
func TestParseTxResponse_DateFallbackJsonNumber(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	txHash := "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233"
	tx := &transactions.TxResponse{
		Hash:        types.Hash256(txHash),
		LedgerIndex: 50000,
		Date:        0, // Force fallback
		Validated:   true,
		TxJSON: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
			"TicketCount":     float64(1),
			"date":            json.Number("784111200"),
		},
		Meta: transaction.TxMetadataBuilder{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			AffectedNodes: []transaction.AffectedNode{
				{
					CreatedNode: &transaction.CreatedNode{
						LedgerEntryType: ledger.TicketEntry,
						NewFields:       ledger.FlatLedgerObject{"TicketSequence": float64(50)},
					},
				},
			},
		},
	}

	msg, err := p.ParseTxResponse(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	expectedTimestamp := time.Unix(784111200+rippleEpochOffset, 0)
	assert.Equal(t, expectedTimestamp, msg.Timestamp)
}

// TestParseTxResponse_DateFallbackJsonNumberError tests json.Number that fails to parse
func TestParseTxResponse_DateFallbackJsonNumberError(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	txHash := "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233"
	tx := &transactions.TxResponse{
		Hash:        types.Hash256(txHash),
		LedgerIndex: 50000,
		Date:        0,
		Validated:   true,
		TxJSON: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
			"date":            json.Number("not-a-number"),
		},
		Meta: transaction.TxMetadataBuilder{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
		},
	}

	msg, err := p.ParseTxResponse(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "date is zero")
}

// TestParseNttTransaction_SkipsCoreAccount tests that NTT transactions to the core account are skipped
func TestParseNttTransaction_SkipsCoreAccount(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)

	ftx := createValidNTTTransaction()
	ftx["Destination"] = testCoreAccount

	gtx := GenericTx{
		Transaction:           ftx,
		Timestamp:             time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Hash:                  "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:           12345,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
		MetaDeliveredAmount:   "1000000",
	}

	msg, err := p.parseNttTransaction(gtx)
	require.NoError(t, err)
	assert.Nil(t, msg, "Should skip NTT transactions to the core account")
}

// TestParseCoreTransaction_NoDestination tests parseCoreTransaction when Destination is missing
func TestParseCoreTransaction_NoDestination(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)

	gtx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			// No Destination field
		},
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseCoreTransaction(gtx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "Destination")
}

// TestParseCoreTransaction_MemoParseError tests parseCoreTransaction when memo data is malformed
func TestParseCoreTransaction_MemoParseError(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)

	gtx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     testCoreAccount,
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": testCoreMemoFormat,
						"MemoData":   "ZZZZ", // invalid hex
					},
				},
			},
		},
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseCoreTransaction(gtx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "failed to decode core MemoData")
}

// TestParseCoreTransaction_NoCoreMemo tests parseCoreTransaction when memo is present but not core format
func TestParseCoreTransaction_NoCoreMemo(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)

	gtx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "Payment",
			"Account":         "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh",
			"Destination":     testCoreAccount,
			"Memos": []any{
				map[string]any{
					"Memo": map[string]any{
						"MemoFormat": "746578742F706C61696E", // text/plain
						"MemoData":   "48656C6C6F",
					},
				},
			},
		},
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseCoreTransaction(gtx)
	require.NoError(t, err)
	assert.Nil(t, msg, "Should return nil when no core memo found")
}

// TestParseCoreTransaction_InvalidSender tests parseCoreTransaction with invalid sender address
func TestParseCoreTransaction_InvalidSender(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)

	payload := []byte("test")
	memoData := createCoreMemoData(1, 1, payload)

	ftx := createFlatTransactionWithCoreMemo(memoData, testCoreAccount)
	ftx["Account"] = "invalidAddress"

	gtx := GenericTx{
		Transaction:           ftx,
		Timestamp:             time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Hash:                  "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:           12345,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseCoreTransaction(gtx)
	require.Error(t, err)
	assert.Nil(t, msg)
}

// TestParseCoreTransaction_InvalidTxHash tests parseCoreTransaction with invalid tx hash
func TestParseCoreTransaction_InvalidTxHash(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)

	payload := []byte("test")
	memoData := createCoreMemoData(1, 1, payload)

	gtx := GenericTx{
		Transaction:           createFlatTransactionWithCoreMemo(memoData, testCoreAccount),
		Timestamp:             time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Hash:                  "ZZZZ", // invalid hex
		LedgerIndex:           12345,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseCoreTransaction(gtx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "failed to decode tx hash")
}

// TestParseCoreTransaction_TransactionIndexOverflow tests parseCoreTransaction with overflowing tx index
func TestParseCoreTransaction_TransactionIndexOverflow(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)

	payload := []byte("test")
	memoData := createCoreMemoData(1, 1, payload)

	gtx := GenericTx{
		Transaction:           createFlatTransactionWithCoreMemo(memoData, testCoreAccount),
		Timestamp:             time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Hash:                  "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:           12345,
		MetaTransactionIndex:  math.MaxUint32 + 1,
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseCoreTransaction(gtx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "invalid transaction index")
}

// TestParseCoreMessageMemoData_MalformedStructures tests all the early-return branches
// in parseCoreMessageMemoData for malformed memo structures.
func TestParseCoreMessageMemoData_MalformedStructures(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)

	testCases := []struct {
		name string
		tx   transaction.FlatTransaction
	}{
		{
			name: "Memos not an array",
			tx: transaction.FlatTransaction{
				"Memos": "not an array",
			},
		},
		{
			name: "Empty memos array",
			tx: transaction.FlatTransaction{
				"Memos": []any{},
			},
		},
		{
			name: "Memo wrapper not a map",
			tx: transaction.FlatTransaction{
				"Memos": []any{"not a map"},
			},
		},
		{
			name: "Memo field missing from wrapper",
			tx: transaction.FlatTransaction{
				"Memos": []any{
					map[string]any{
						"NotMemo": map[string]any{},
					},
				},
			},
		},
		{
			name: "Memo not a map",
			tx: transaction.FlatTransaction{
				"Memos": []any{
					map[string]any{
						"Memo": "not a map",
					},
				},
			},
		},
		{
			name: "MemoFormat not a string",
			tx: transaction.FlatTransaction{
				"Memos": []any{
					map[string]any{
						"Memo": map[string]any{
							"MemoFormat": 12345,
						},
					},
				},
			},
		},
		{
			name: "Wrong MemoFormat",
			tx: transaction.FlatTransaction{
				"Memos": []any{
					map[string]any{
						"Memo": map[string]any{
							"MemoFormat": testNTTMemoFormat, // NTT format, not core
							"MemoData":   "01000000010A",
						},
					},
				},
			},
		},
		{
			name: "MemoData not a string",
			tx: transaction.FlatTransaction{
				"Memos": []any{
					map[string]any{
						"Memo": map[string]any{
							"MemoFormat": testCoreMemoFormat,
							"MemoData":   12345,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := p.parseCoreMessageMemoData(tc.tx)
			require.NoError(t, err)
			assert.Nil(t, result)
		})
	}
}

// TestParseCoreMessageMemoData_InvalidHex tests invalid hex in MemoData
func TestParseCoreMessageMemoData_InvalidHex(t *testing.T) {
	p := NewParser(testCoreAccount, nil, nil)

	tx := transaction.FlatTransaction{
		"Memos": []any{
			map[string]any{
				"Memo": map[string]any{
					"MemoFormat": testCoreMemoFormat,
					"MemoData":   "ZZZZ",
				},
			},
		},
	}

	result, err := p.parseCoreMessageMemoData(tx)
	require.Error(t, err)
	assert.Nil(t, result)
	assert.Contains(t, err.Error(), "failed to decode core MemoData")
}

// TestParseTicketCreateTransaction_MissingTransactionType tests early return when TransactionType is missing
func TestParseTicketCreateTransaction_MissingTransactionType(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"Account": testManagedAccount,
		},
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	assert.NoError(t, err)
	assert.Nil(t, msg)
}

// TestParseTicketCreateTransaction_MissingAccount tests early return when Account is missing
func TestParseTicketCreateTransaction_MissingAccount(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
		},
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	assert.NoError(t, err)
	assert.Nil(t, msg)
}

// TestParseTicketCreateTransaction_AccountNotString tests early return when Account is not a string
func TestParseTicketCreateTransaction_AccountNotString(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         12345,
		},
		MetaTransactionResult: "tesSUCCESS",
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	assert.NoError(t, err)
	assert.Nil(t, msg)
}

// TestParseTicketCreateTransaction_AffectedNodeVariants tests different AffectedNode shapes
func TestParseTicketCreateTransaction_AffectedNodeVariants(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	// Mix of: nil CreatedNode, non-Ticket entry, missing TicketSequence, and valid ticket
	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
		},
		Hash:                  "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233",
		LedgerIndex:           50000,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
		MetaAffectedNodes: []transaction.AffectedNode{
			{CreatedNode: nil}, // nil CreatedNode
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: "AccountRoot", // not a Ticket
					NewFields:       ledger.FlatLedgerObject{},
				},
			},
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: ledger.TicketEntry,
					NewFields:       ledger.FlatLedgerObject{}, // missing TicketSequence
				},
			},
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: ledger.TicketEntry,
					NewFields:       ledger.FlatLedgerObject{"TicketSequence": float64(42)},
				},
			},
		},
		Timestamp: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	ticketStart := binary.BigEndian.Uint64(msg.Payload[4:12])
	assert.Equal(t, uint64(42), ticketStart)
	ticketCount := binary.BigEndian.Uint64(msg.Payload[12:20])
	assert.Equal(t, uint64(1), ticketCount)
}

// TestParseTicketCreateTransaction_JsonNumberSequence tests TicketSequence as json.Number
func TestParseTicketCreateTransaction_JsonNumberSequence(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
		},
		Hash:                  "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233",
		LedgerIndex:           50000,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
		MetaAffectedNodes: []transaction.AffectedNode{
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: ledger.TicketEntry,
					NewFields:       ledger.FlatLedgerObject{"TicketSequence": json.Number("100")},
				},
			},
		},
		Timestamp: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	require.NoError(t, err)
	require.NotNil(t, msg)

	ticketStart := binary.BigEndian.Uint64(msg.Payload[4:12])
	assert.Equal(t, uint64(100), ticketStart)
}

// TestParseTicketCreateTransaction_JsonNumberParseError tests json.Number that fails to parse
func TestParseTicketCreateTransaction_JsonNumberParseError(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
		},
		Hash:                  "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233",
		LedgerIndex:           50000,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
		MetaAffectedNodes: []transaction.AffectedNode{
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: ledger.TicketEntry,
					NewFields:       ledger.FlatLedgerObject{"TicketSequence": json.Number("not-a-number")},
				},
			},
		},
		Timestamp: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "failed to parse TicketSequence")
}

// TestParseTicketCreateTransaction_NegativeJsonNumberSequence tests negative TicketSequence
func TestParseTicketCreateTransaction_NegativeJsonNumberSequence(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
		},
		Hash:                  "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233",
		LedgerIndex:           50000,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
		MetaAffectedNodes: []transaction.AffectedNode{
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: ledger.TicketEntry,
					NewFields:       ledger.FlatLedgerObject{"TicketSequence": json.Number("-5")},
				},
			},
		},
		Timestamp: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "negative TicketSequence")
}

// TestParseTicketCreateTransaction_UnexpectedSequenceType tests unexpected TicketSequence type
func TestParseTicketCreateTransaction_UnexpectedSequenceType(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
		},
		Hash:                  "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233",
		LedgerIndex:           50000,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
		MetaAffectedNodes: []transaction.AffectedNode{
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: ledger.TicketEntry,
					NewFields:       ledger.FlatLedgerObject{"TicketSequence": "string-not-number"},
				},
			},
		},
		Timestamp: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "unexpected TicketSequence type")
}

// TestParseTicketCreateTransaction_NoTicketEntries tests when AffectedNodes has no ticket entries
func TestParseTicketCreateTransaction_NoTicketEntries(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
		},
		Hash:                  "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233",
		LedgerIndex:           50000,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
		MetaAffectedNodes: []transaction.AffectedNode{
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: "AccountRoot",
					NewFields:       ledger.FlatLedgerObject{},
				},
			},
		},
		Timestamp: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "no created Ticket entries")
}

// TestParseTicketCreateTransaction_InvalidTxHash tests invalid transaction hash
func TestParseTicketCreateTransaction_InvalidTxHash(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         testManagedAccount,
		},
		Hash:                  "ZZZZ", // invalid hex
		LedgerIndex:           50000,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
		MetaAffectedNodes: []transaction.AffectedNode{
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: ledger.TicketEntry,
					NewFields:       ledger.FlatLedgerObject{"TicketSequence": float64(100)},
				},
			},
		},
		Timestamp: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "failed to decode tx hash")
}

// TestParseTicketCreateTransaction_TransactionIndexOverflow tests tx index > MaxUint32
func TestParseTicketCreateTransaction_TransactionIndexOverflow(t *testing.T) {
	p := NewParser("", []string{testManagedAccount}, nil)

	tx := createTicketCreateTx(testManagedAccount, []float64{100})
	tx.MetaTransactionIndex = math.MaxUint32 + 1

	msg, err := p.parseTicketCreateTransaction(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "invalid transaction index")
}

// TestParseTicketCreateTransaction_InvalidAccount tests when the account address is invalid for emitter conversion
func TestParseTicketCreateTransaction_InvalidAccountAddress(t *testing.T) {
	// Use an account that passes the string check but fails addressToEmitter
	invalidAccount := "rInvalidXRPLAddr"
	p := NewParser("", []string{invalidAccount}, nil)

	tx := GenericTx{
		Transaction: transaction.FlatTransaction{
			"TransactionType": "TicketCreate",
			"Account":         invalidAccount,
		},
		Hash:                  "AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233AABBCCDD00112233",
		LedgerIndex:           50000,
		MetaTransactionIndex:  0,
		MetaTransactionResult: "tesSUCCESS",
		MetaAffectedNodes: []transaction.AffectedNode{
			{
				CreatedNode: &transaction.CreatedNode{
					LedgerEntryType: ledger.TicketEntry,
					NewFields:       ledger.FlatLedgerObject{"TicketSequence": float64(100)},
				},
			},
		},
		Timestamp: time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC),
	}

	msg, err := p.parseTicketCreateTransaction(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "failed to convert account to emitter")
}
