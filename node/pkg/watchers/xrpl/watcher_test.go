package xrpl

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"testing"
	"time"

	streamtypes "github.com/Peersyst/xrpl-go/xrpl/queries/subscription/types"
	"github.com/Peersyst/xrpl-go/xrpl/transaction"
	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// Sample NTT MemoData from real XRPL transaction (hex-encoded, 79 bytes)
// Format: prefix(4) + decimals(1) + amount(8) + sourceToken(32) + recipient(32) + recipientChain(2)
// Prefix: 994E5454 (NTT)
// Decimals: 06
// Amount: 00000000000F4240 (1000000 = 1 XRP in drops)
// SourceToken: 0000000000000000000000000000000000000000000000000000000000000000 (32 bytes)
// Recipient: 000000000000000000000000D8DA6BF26964AF9D7EED9E03E53415D37AA96045 (32 bytes)
// RecipientChain: 0002 (Ethereum)
const sampleNTTMemoData = "994E54540600000000000F42400000000000000000000000000000000000000000000000000000000000000000000000000000000000000000D8DA6BF26964AF9D7EED9E03E53415D37AA960450002"

// nttMemoType hex-encoded: "application/x-ntt-transfer"
const testNTTMemoType = "6170706C69636174696F6E2F782D6E74742D7472616E73666572"

// Helper to create a FlatTransaction with memos
func createFlatTransactionWithMemos(memoType, memoData string) transaction.FlatTransaction {
	return transaction.FlatTransaction{
		"Memos": []interface{}{
			map[string]interface{}{
				"Memo": map[string]interface{}{
					"MemoType": memoType,
					"MemoData": memoData,
				},
			},
		},
	}
}

// =============================================================================
// extractWormholePayload tests
// =============================================================================

func TestExtractWormholePayload_ValidNTTMemo(t *testing.T) {
	w := &Watcher{}
	tx := createFlatTransactionWithMemos(testNTTMemoType, sampleNTTMemoData)

	payload, nonce, err := w.extractWormholePayload(tx)

	require.NoError(t, err)
	require.NotNil(t, payload)
	assert.Equal(t, uint32(0), nonce, "NTT nonce should be 0")

	// Verify payload starts with NTT prefix
	assert.Equal(t, byte(0x99), payload[0])
	assert.Equal(t, byte(0x4E), payload[1])
	assert.Equal(t, byte(0x54), payload[2])
	assert.Equal(t, byte(0x54), payload[3])
}

func TestExtractWormholePayload_NoMemos(t *testing.T) {
	w := &Watcher{}
	tx := transaction.FlatTransaction{
		"Account": "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	payload, nonce, err := w.extractWormholePayload(tx)

	require.NoError(t, err)
	assert.Nil(t, payload, "Should return nil when no Memos field")
	assert.Equal(t, uint32(0), nonce)
}

func TestExtractWormholePayload_EmptyMemos(t *testing.T) {
	w := &Watcher{}
	tx := transaction.FlatTransaction{
		"Memos": []interface{}{},
	}

	payload, nonce, err := w.extractWormholePayload(tx)

	require.NoError(t, err)
	assert.Nil(t, payload)
	assert.Equal(t, uint32(0), nonce)
}

func TestExtractWormholePayload_WrongMemoType(t *testing.T) {
	w := &Watcher{}
	// Use a different MemoType (hex for "text/plain")
	wrongMemoType := "746578742F706C61696E"
	tx := createFlatTransactionWithMemos(wrongMemoType, sampleNTTMemoData)

	payload, nonce, err := w.extractWormholePayload(tx)

	require.NoError(t, err)
	assert.Nil(t, payload, "Should return nil for wrong MemoType")
	assert.Equal(t, uint32(0), nonce)
}

func TestExtractWormholePayload_InvalidNTTPrefix(t *testing.T) {
	w := &Watcher{}
	// Valid hex but wrong prefix (not 994E5454)
	wrongPrefixData := "DEADBEEF0600000000000F42400000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000000200010002"
	tx := createFlatTransactionWithMemos(testNTTMemoType, wrongPrefixData)

	payload, nonce, err := w.extractWormholePayload(tx)

	require.NoError(t, err)
	assert.Nil(t, payload, "Should return nil for wrong NTT prefix")
	assert.Equal(t, uint32(0), nonce)
}

func TestExtractWormholePayload_InvalidHexMemoData(t *testing.T) {
	w := &Watcher{}
	// Invalid hex string
	tx := createFlatTransactionWithMemos(testNTTMemoType, "NOTVALIDHEX!!!")

	payload, nonce, err := w.extractWormholePayload(tx)

	require.Error(t, err, "Should return error for invalid hex")
	assert.Nil(t, payload)
	assert.Equal(t, uint32(0), nonce)
}

func TestExtractWormholePayload_TooShortPayload(t *testing.T) {
	w := &Watcher{}
	// Payload shorter than 4 bytes (NTT prefix length)
	shortData := "99"
	tx := createFlatTransactionWithMemos(testNTTMemoType, shortData)

	payload, nonce, err := w.extractWormholePayload(tx)

	require.NoError(t, err)
	assert.Nil(t, payload, "Should return nil for payload too short for prefix check")
	assert.Equal(t, uint32(0), nonce)
}

func TestExtractWormholePayload_MultipleMemos_OnlyOneValid(t *testing.T) {
	w := &Watcher{}
	tx := transaction.FlatTransaction{
		"Memos": []interface{}{
			// First memo: wrong type
			map[string]interface{}{
				"Memo": map[string]interface{}{
					"MemoType": "746578742F706C61696E", // "text/plain"
					"MemoData": "48656C6C6F",           // "Hello"
				},
			},
			// Second memo: valid NTT
			map[string]interface{}{
				"Memo": map[string]interface{}{
					"MemoType": testNTTMemoType,
					"MemoData": sampleNTTMemoData,
				},
			},
		},
	}

	payload, nonce, err := w.extractWormholePayload(tx)

	require.NoError(t, err)
	require.NotNil(t, payload, "Should find the valid NTT memo")
	assert.Equal(t, uint32(0), nonce)
	// Verify it's the NTT payload
	assert.Equal(t, byte(0x99), payload[0])
}

func TestExtractWormholePayload_MalformedMemoStructure(t *testing.T) {
	w := &Watcher{}

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
				"Memos": []interface{}{"not a map"},
			},
		},
		{
			name: "Memo field missing",
			tx: transaction.FlatTransaction{
				"Memos": []interface{}{
					map[string]interface{}{
						"NotMemo": map[string]interface{}{},
					},
				},
			},
		},
		{
			name: "Memo not a map",
			tx: transaction.FlatTransaction{
				"Memos": []interface{}{
					map[string]interface{}{
						"Memo": "not a map",
					},
				},
			},
		},
		{
			name: "MemoType not a string",
			tx: transaction.FlatTransaction{
				"Memos": []interface{}{
					map[string]interface{}{
						"Memo": map[string]interface{}{
							"MemoType": 12345,
							"MemoData": sampleNTTMemoData,
						},
					},
				},
			},
		},
		{
			name: "MemoData not a string",
			tx: transaction.FlatTransaction{
				"Memos": []interface{}{
					map[string]interface{}{
						"Memo": map[string]interface{}{
							"MemoType": testNTTMemoType,
							"MemoData": 12345,
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			payload, nonce, err := w.extractWormholePayload(tc.tx)

			// Malformed structures should not cause errors, just return nil
			require.NoError(t, err)
			assert.Nil(t, payload)
			assert.Equal(t, uint32(0), nonce)
		})
	}
}

// =============================================================================
// addressToEmitter tests
// =============================================================================

func TestAddressToEmitter_ValidAddress(t *testing.T) {
	w := &Watcher{}

	// Standard XRPL r-address
	address := "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"

	emitter, err := w.addressToEmitter(address)

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
	w := &Watcher{}

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
			_, err := w.addressToEmitter(tc.address)
			assert.Error(t, err, "Should return error for invalid address")
		})
	}
}

func TestAddressToEmitter_ConsistentResults(t *testing.T) {
	w := &Watcher{}
	address := "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"

	// Call twice and verify same result
	emitter1, err1 := w.addressToEmitter(address)
	emitter2, err2 := w.addressToEmitter(address)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.Equal(t, emitter1, emitter2, "Same address should produce same emitter")
}

func TestAddressToEmitter_DifferentAddresses(t *testing.T) {
	w := &Watcher{}

	addr1 := "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9"
	addr2 := "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh" // Genesis account

	emitter1, err1 := w.addressToEmitter(addr1)
	emitter2, err2 := w.addressToEmitter(addr2)

	require.NoError(t, err1)
	require.NoError(t, err2)
	assert.NotEqual(t, emitter1, emitter2, "Different addresses should produce different emitters")
}

// =============================================================================
// parseTransactionStream tests
// =============================================================================

func TestParseTransactionStream_ValidTransaction(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	// Create a mock TransactionStream matching real XRPL transaction structure
	txHash := "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D"
	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  createFlatTransactionWithMemos(testNTTMemoType, sampleNTTMemoData),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  7,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000", // XRP drops as string (matches NTT payload amount)
		},
	}

	msg, err := w.parseTransactionStream(tx)

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

	// Verify payload is present
	assert.NotNil(t, msg.Payload)
	assert.True(t, len(msg.Payload) > 0)
}

func TestParseTransactionStream_NoWormholePayload(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"Account": "rSomeOtherAccount",
		},
	}

	msg, err := w.parseTransactionStream(tx)

	require.NoError(t, err)
	assert.Nil(t, msg, "Should return nil for transaction without Wormhole payload")
}

func TestParseTransactionStream_InvalidTimestamp(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "not-a-valid-timestamp",
		Validated:    true,
		Transaction:  createFlatTransactionWithMemos(testNTTMemoType, sampleNTTMemoData),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := w.parseTransactionStream(tx)

	assert.Error(t, err, "Should return error for invalid timestamp")
}

func TestParseTransactionStream_InvalidTxHash(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	tx := &streamtypes.TransactionStream{
		Hash:         "NOT_VALID_HEX!!!",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction:  createFlatTransactionWithMemos(testNTTMemoType, sampleNTTMemoData),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  0,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000",
		},
	}

	_, err := w.parseTransactionStream(tx)

	assert.Error(t, err, "Should return error for invalid tx hash")
}

// =============================================================================
// processTransaction tests
// =============================================================================

func TestProcessTransaction_SkipsUnvalidated(t *testing.T) {
	msgChan := make(chan *common.MessagePublication, 1)
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
		msgChan:  msgChan,
	}

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    false, // Not validated
		Transaction:  createFlatTransactionWithMemos(testNTTMemoType, sampleNTTMemoData),
	}

	err := w.processTransaction(context.Background(), zap.NewNop(), tx)

	require.NoError(t, err)
	assert.Empty(t, msgChan, "No message should be sent for unvalidated transaction")
}

func TestProcessTransaction_SendsValidatedMessage(t *testing.T) {
	msgChan := make(chan *common.MessagePublication, 1)
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
		msgChan:  msgChan,
	}

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true, // Validated
		Transaction:  createFlatTransactionWithMemos(testNTTMemoType, sampleNTTMemoData),
		Meta: transaction.TxObjMeta{
			TransactionIndex:  3,
			TransactionResult: "tesSUCCESS",
			DeliveredAmount:   "1000000", // XRP drops as string
		},
	}

	err := w.processTransaction(context.Background(), zap.NewNop(), tx)

	require.NoError(t, err)
	require.Len(t, msgChan, 1, "Message should be sent for validated transaction")

	// Verify the message contents
	msg := <-msgChan
	assert.Equal(t, vaa.ChainIDXRPL, msg.EmitterChain)
	// Sequence = (ledgerIndex << 32) | txIndex = (12345 << 32) | 3
	assert.Equal(t, (uint64(12345)<<32)|3, msg.Sequence)
	assert.False(t, msg.IsReobservation)
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
// validateTransactionResult tests
// =============================================================================

func TestValidateTransactionResult_Success(t *testing.T) {
	w := &Watcher{}
	tx := transaction.FlatTransaction{
		"meta": map[string]interface{}{
			"TransactionResult": "tesSUCCESS",
		},
	}

	err := w.validateTransactionResult(tx)
	require.NoError(t, err)
}

func TestValidateTransactionResult_Failed(t *testing.T) {
	w := &Watcher{}

	failureCodes := []string{
		"tecUNFUNDED_PAYMENT",
		"tecNO_DST",
		"tecNO_DST_INSUF_XRP",
		"tecPATH_DRY",
		"tefPAST_SEQ",
	}

	for _, code := range failureCodes {
		t.Run(code, func(t *testing.T) {
			tx := transaction.FlatTransaction{
				"meta": map[string]interface{}{
					"TransactionResult": code,
				},
			}

			err := w.validateTransactionResult(tx)
			require.Error(t, err)
			assert.Contains(t, err.Error(), code)
		})
	}
}

func TestValidateTransactionResult_NoMeta(t *testing.T) {
	w := &Watcher{}
	tx := transaction.FlatTransaction{
		"Account": "rSomeAccount",
	}

	// No meta field - should allow processing to continue
	err := w.validateTransactionResult(tx)
	require.NoError(t, err)
}

func TestValidateTransactionResult_MalformedMeta(t *testing.T) {
	w := &Watcher{}

	testCases := []struct {
		name string
		tx   transaction.FlatTransaction
	}{
		{
			name: "meta not a map",
			tx: transaction.FlatTransaction{
				"meta": "not a map",
			},
		},
		{
			name: "TransactionResult missing",
			tx: transaction.FlatTransaction{
				"meta": map[string]interface{}{
					"other_field": "value",
				},
			},
		},
		{
			name: "TransactionResult not a string",
			tx: transaction.FlatTransaction{
				"meta": map[string]interface{}{
					"TransactionResult": 12345,
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Malformed structures should not cause errors, allow processing to continue
			err := w.validateTransactionResult(tc.tx)
			require.NoError(t, err)
		})
	}
}

// =============================================================================
// validateDestination tests
// =============================================================================

func TestValidateDestination_Matches(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}
	tx := transaction.FlatTransaction{
		"Destination": "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	err := w.validateDestination(tx)
	require.NoError(t, err)
}

func TestValidateDestination_DoesNotMatch(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}
	tx := transaction.FlatTransaction{
		"Destination": "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", // Different address
	}

	err := w.validateDestination(tx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match custody account")
}

func TestValidateDestination_NoDestination(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}
	tx := transaction.FlatTransaction{
		"Account": "rSomeAccount",
	}

	// No Destination field - should allow processing to continue
	err := w.validateDestination(tx)
	require.NoError(t, err)
}

func TestValidateDestination_MalformedDestination(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}
	tx := transaction.FlatTransaction{
		"Destination": 12345, // Not a string
	}

	// Malformed destination should allow processing to continue
	err := w.validateDestination(tx)
	require.NoError(t, err)
}

// =============================================================================
// validateRecipientChain tests
// =============================================================================

func TestValidateRecipientChain_ValidChain(t *testing.T) {
	w := &Watcher{}

	// Sample NTT payload has recipient chain at bytes 77-78
	// sampleNTTMemoData ends with "0002" which is chain ID 2 (Ethereum)
	payload, err := hex.DecodeString(sampleNTTMemoData)
	require.NoError(t, err)

	err = w.validateRecipientChain(payload)
	require.NoError(t, err)
}

func TestValidateRecipientChain_ChainIDZero(t *testing.T) {
	w := &Watcher{}

	// Create a 79-byte payload with chain ID 0 (invalid)
	// NTT format: prefix(4) + decimals(1) + amount(8) + sourceToken(32) + recipient(32) + recipientChain(2)
	payload := make([]byte, 79)
	copy(payload[0:4], nttPayloadPrefix) // NTT prefix
	payload[4] = 0x06                    // decimals
	// amount at offset 5-12 (8 bytes) - leave as zeros
	// sourceToken at offset 13-44 (32 bytes) - leave as zeros
	// recipient at offset 45-76 (32 bytes) - leave as zeros
	// recipientChain at offset 77-78 = 0x0000 (chain ID 0)
	payload[77] = 0x00
	payload[78] = 0x00

	err := w.validateRecipientChain(payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid recipient chain ID: 0")
}

func TestValidateRecipientChain_PayloadTooShort(t *testing.T) {
	w := &Watcher{}

	// Create a payload that's too short (less than 79 bytes)
	shortPayload := make([]byte, 50)
	copy(shortPayload, nttPayloadPrefix)

	err := w.validateRecipientChain(shortPayload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NTT payload too short")
}

func TestValidateRecipientChain_UnknownChainID(t *testing.T) {
	w := &Watcher{}

	// Create a 79-byte payload with an unknown chain ID (e.g., 9999)
	payload := make([]byte, 79)
	copy(payload[0:4], nttPayloadPrefix) // NTT prefix
	payload[4] = 0x06                    // decimals
	// Set recipient chain ID to 9999 (0x270F) - not a known Wormhole chain
	payload[77] = 0x27
	payload[78] = 0x0F

	err := w.validateRecipientChain(payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown recipient chain ID: 9999")
}

func TestValidateRecipientChain_VariousValidChains(t *testing.T) {
	w := &Watcher{}

	// Test various valid chain IDs
	validChains := []struct {
		id   uint16
		name string
	}{
		{1, "Solana"},
		{2, "Ethereum"},
		{4, "BSC"},
		{6, "Avalanche"},
		{21, "Sui"},
		{22, "Aptos"},
		{66, "XRPL"},
	}

	for _, chain := range validChains {
		t.Run("ChainID_"+chain.name, func(t *testing.T) {
			// Create a 79-byte payload with the chain ID
			payload := make([]byte, 79)
			copy(payload[0:4], nttPayloadPrefix) // NTT prefix
			payload[4] = 0x06                    // decimals
			// Set recipient chain ID (big-endian)
			payload[77] = byte(chain.id >> 8)
			payload[78] = byte(chain.id)

			err := w.validateRecipientChain(payload)
			require.NoError(t, err)
		})
	}
}

// =============================================================================
// Integration test: parseTransactionStream with validations
// =============================================================================

func TestParseTransactionStream_FailsOnNonSuccessResult(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"Destination": "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]interface{}{
				"TransactionResult": "tecUNFUNDED_PAYMENT",
			},
			"Memos": []interface{}{
				map[string]interface{}{
					"Memo": map[string]interface{}{
						"MemoType": testNTTMemoType,
						"MemoData": sampleNTTMemoData,
					},
				},
			},
		},
	}

	msg, err := w.parseTransactionStream(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "tecUNFUNDED_PAYMENT")
}

func TestParseTransactionStream_FailsOnWrongDestination(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"Destination": "rHb9CJAWyB4rj91VRWn96DkukG4bwdtyTh", // Wrong destination
			"meta": map[string]interface{}{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []interface{}{
				map[string]interface{}{
					"Memo": map[string]interface{}{
						"MemoType": testNTTMemoType,
						"MemoData": sampleNTTMemoData,
					},
				},
			},
		},
	}

	msg, err := w.parseTransactionStream(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "does not match custody account")
}

func TestParseTransactionStream_FailsOnInvalidRecipientChain(t *testing.T) {
	w := &Watcher{
		contract: "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
	}

	// Create a valid 79-byte NTT payload with chain ID 0 (invalid)
	// NTT format: prefix(4) + decimals(1) + amount(8) + sourceToken(32) + recipient(32) + recipientChain(2) = 79 bytes
	payload := make([]byte, 79)
	copy(payload[0:4], nttPayloadPrefix) // NTT prefix 0x994E5454
	payload[4] = 0x06                    // decimals
	// amount at offset 5-12 (8 bytes), sourceToken at 13-44 (32 bytes), recipient at 45-76 (32 bytes)
	// all zeros is fine for test
	// recipientChain at offset 77-78 = 0x0000 (chain ID 0)
	payload[77] = 0x00
	payload[78] = 0x00
	invalidChainPayload := hex.EncodeToString(payload)

	tx := &streamtypes.TransactionStream{
		Hash:         "8A9ABA7F403A49F8AF8ADE4E54BE2BD5901FBD2E426C2844207D287A090AF55D",
		LedgerIndex:  12345,
		CloseTimeISO: "2024-01-15T10:30:00Z",
		Validated:    true,
		Transaction: transaction.FlatTransaction{
			"Destination": "rN7n3473SaZBCG4dFL83w7a1RXtXtbk2D9",
			"meta": map[string]interface{}{
				"TransactionResult": "tesSUCCESS",
			},
			"Memos": []interface{}{
				map[string]interface{}{
					"Memo": map[string]interface{}{
						"MemoType": testNTTMemoType,
						"MemoData": invalidChainPayload,
					},
				},
			},
		},
	}

	msg, err := w.parseTransactionStream(tx)
	require.Error(t, err)
	assert.Nil(t, msg)
	assert.Contains(t, err.Error(), "invalid recipient chain ID: 0")
}

// =============================================================================
// validateAmount tests
// =============================================================================

func TestValidateAmount_ValidAmount(t *testing.T) {
	w := &Watcher{}

	// Create a 79-byte NTT payload with amount = 1000000 (1 XRP in drops)
	payload := make([]byte, 79)
	copy(payload[0:4], nttPayloadPrefix)
	payload[4] = 0x06 // decimals
	// Amount at offset 5-12 (8 bytes, big-endian) = 1000000
	binary.BigEndian.PutUint64(payload[5:13], 1000000)
	// Set a valid recipient chain
	payload[77] = 0x00
	payload[78] = 0x02 // Ethereum

	// delivered_amount = "2000000" (2 XRP in drops) - greater than NTT amount
	err := w.validateAmount("2000000", payload)
	require.NoError(t, err)
}

func TestValidateAmount_ExactMatch(t *testing.T) {
	w := &Watcher{}

	payload := make([]byte, 79)
	copy(payload[0:4], nttPayloadPrefix)
	payload[4] = 0x06
	binary.BigEndian.PutUint64(payload[5:13], 1000000)
	payload[77] = 0x00
	payload[78] = 0x02

	// delivered_amount exactly matches NTT amount
	err := w.validateAmount("1000000", payload)
	require.NoError(t, err)
}

func TestValidateAmount_ExceedsDelivered(t *testing.T) {
	w := &Watcher{}

	payload := make([]byte, 79)
	copy(payload[0:4], nttPayloadPrefix)
	payload[4] = 0x06
	// NTT amount = 2000000
	binary.BigEndian.PutUint64(payload[5:13], 2000000)
	payload[77] = 0x00
	payload[78] = 0x02

	// delivered_amount = "1000000" - less than NTT amount (invalid)
	err := w.validateAmount("1000000", payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NTT amount 2000000 exceeds delivered amount 1000000")
}

func TestValidateAmount_NotAString(t *testing.T) {
	w := &Watcher{}

	payload := make([]byte, 79)
	copy(payload[0:4], nttPayloadPrefix)
	payload[4] = 0x06
	binary.BigEndian.PutUint64(payload[5:13], 1000000)
	payload[77] = 0x00
	payload[78] = 0x02

	// delivered_amount is nil - NTT requires XRP, so this should error
	err := w.validateAmount(nil, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delivered_amount is not a string")

	// delivered_amount is an integer - should also error
	err = w.validateAmount(12345, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "delivered_amount is not a string")
}

func TestValidateAmount_InvalidString(t *testing.T) {
	w := &Watcher{}

	payload := make([]byte, 79)
	copy(payload[0:4], nttPayloadPrefix)
	payload[4] = 0x06
	binary.BigEndian.PutUint64(payload[5:13], 1000000)
	payload[77] = 0x00
	payload[78] = 0x02

	// delivered_amount is not a valid number
	err := w.validateAmount("not_a_number", payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse delivered_amount")
}

func TestValidateAmount_PayloadTooShort(t *testing.T) {
	w := &Watcher{}

	// Payload too short to contain amount
	shortPayload := make([]byte, 10)
	copy(shortPayload[0:4], nttPayloadPrefix)

	err := w.validateAmount("1000000", shortPayload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "NTT payload too short")
}
