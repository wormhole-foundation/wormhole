package xrpl

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Test data
var (
	testCustodyAccountID = [20]byte{
		0xB5, 0xF7, 0x62, 0x79, 0x8A, 0x53, 0xD5, 0x43, 0xA0, 0x14,
		0xCA, 0xF8, 0xB2, 0x97, 0xCF, 0xF8, 0xF2, 0xF9, 0x37, 0xE8,
	}
	testRecipientAccountID = [20]byte{
		0x7A, 0x02, 0xB0, 0x5B, 0x8D, 0x53, 0xC5, 0x4C, 0x16, 0xAA,
		0x7B, 0x55, 0x2D, 0x22, 0x3D, 0xB5, 0x79, 0x67, 0xF7, 0x14,
	}
	testSourceEmitter = [32]byte{
		0x18, 0x07, 0xdd, 0xdb, 0xb4, 0x86, 0x6e, 0x81,
		0xbb, 0x82, 0x51, 0x38, 0x4a, 0xed, 0x02, 0x6d,
		0xe5, 0x49, 0x6f, 0xc8, 0xb3, 0x83, 0xf8, 0x39,
		0x9d, 0x1d, 0xe5, 0xd8, 0x44, 0xb1, 0x42, 0x71,
	}
	testManagerSetM uint8 = 7
)

func TestBuildPaymentTransactionXRP(t *testing.T) {
	payload := &vaa.XRPLReleasePayload{
		TicketID:       42,
		CustodyAccount: testCustodyAccountID,
		Recipient:      testRecipientAccountID,
		Amount:         1000000, // 1 XRP in drops
		TokenDecimals:  6,
		SourceChain:    vaa.ChainIDSolana,
		SourceEmitter:  testSourceEmitter,
		SourceSequence: 100,
		Token: vaa.XRPLTokenID{
			Type: vaa.XRPLTokenTypeXRP,
		},
	}

	flatTx, err := BuildPaymentTransaction(payload, testManagerSetM)
	require.NoError(t, err)

	assert.Equal(t, "Payment", flatTx["TransactionType"])
	assert.Equal(t, uint32(0), flatTx["Sequence"])
	assert.Equal(t, "", flatTx["SigningPubKey"])
	assert.Equal(t, uint32(42), flatTx["TicketSequence"])
	// Fee: (7 + 1) * 12 = 96
	assert.Equal(t, "96", flatTx["Fee"])
	// Amount should be XRP drops as string
	assert.Equal(t, "1000000", flatTx["Amount"])
}

func TestBuildPaymentTransactionIOU(t *testing.T) {
	// USD currency in standard 20-byte format
	currency := [20]byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x55, 0x53, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	issuer := [20]byte{
		0xB5, 0xF7, 0x62, 0x79, 0x8A, 0x53, 0xD5, 0x43, 0xA0, 0x14,
		0xCA, 0xF8, 0xB2, 0x97, 0xCF, 0xF8, 0xF2, 0xF9, 0x37, 0xE8,
	}

	payload := &vaa.XRPLReleasePayload{
		TicketID:       10,
		CustodyAccount: testCustodyAccountID,
		Recipient:      testRecipientAccountID,
		Amount:         12345678, // 123.45678 with 5 decimals
		TokenDecimals:  5,
		SourceChain:    vaa.ChainIDSolana,
		SourceEmitter:  testSourceEmitter,
		SourceSequence: 200,
		Token: vaa.XRPLTokenID{
			Type:     vaa.XRPLTokenTypeIOU,
			Currency: currency,
			Issuer:   issuer,
		},
	}

	flatTx, err := BuildPaymentTransaction(payload, testManagerSetM)
	require.NoError(t, err)

	assert.Equal(t, "Payment", flatTx["TransactionType"])

	// Amount should be an IOU map
	amountMap, ok := flatTx["Amount"].(map[string]interface{})
	require.True(t, ok, "Amount should be a map for IOU")
	assert.Equal(t, "USD", amountMap["currency"])
	assert.Equal(t, "123.45678", amountMap["value"])
}

func TestBuildPaymentTransactionMPT(t *testing.T) {
	mptID := [24]byte{
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	}

	payload := &vaa.XRPLReleasePayload{
		TicketID:       20,
		CustodyAccount: testCustodyAccountID,
		Recipient:      testRecipientAccountID,
		Amount:         999,
		TokenDecimals:  0,
		SourceChain:    vaa.ChainIDSolana,
		SourceEmitter:  testSourceEmitter,
		SourceSequence: 300,
		Token: vaa.XRPLTokenID{
			Type:  vaa.XRPLTokenTypeMPT,
			MPTID: mptID,
		},
	}

	flatTx, err := BuildPaymentTransaction(payload, testManagerSetM)
	require.NoError(t, err)

	assert.Equal(t, "Payment", flatTx["TransactionType"])

	// Amount should be an MPT map
	amountMap, ok := flatTx["Amount"].(map[string]interface{})
	require.True(t, ok, "Amount should be a map for MPT")
	assert.Equal(t, "999", amountMap["value"])
	assert.Equal(t, hex.EncodeToString(mptID[:]), amountMap["mpt_issuance_id"])
}

func TestBuildPaymentTransactionWithMemos(t *testing.T) {
	payload := &vaa.XRPLReleasePayload{
		TicketID:       42,
		CustodyAccount: testCustodyAccountID,
		Recipient:      testRecipientAccountID,
		Amount:         1000000,
		TokenDecimals:  6,
		SourceChain:    vaa.ChainIDSolana,
		SourceEmitter:  testSourceEmitter,
		SourceSequence: 100,
		Token: vaa.XRPLTokenID{
			Type: vaa.XRPLTokenTypeXRP,
		},
		Memos: []vaa.XRPLMemo{
			{
				Data:   []byte("hello"),
				Format: []byte("text/plain"),
				Type:   []byte("message"),
			},
			{
				Data:   []byte{0xDE, 0xAD},
				Format: []byte{},
				Type:   []byte("bin"),
			},
		},
	}

	flatTx, err := BuildPaymentTransaction(payload, testManagerSetM)
	require.NoError(t, err)

	assert.Equal(t, "Payment", flatTx["TransactionType"])

	// Verify memos are present in the flat transaction
	memosRaw, ok := flatTx["Memos"]
	require.True(t, ok, "Memos should be present in flat transaction")

	memos, ok := memosRaw.([]interface{})
	require.True(t, ok, "Memos should be a slice")
	assert.Len(t, memos, 2)

	// Verify first memo
	memo0, ok := memos[0].(map[string]interface{})
	require.True(t, ok)
	memoInner0, ok := memo0["Memo"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, hex.EncodeToString([]byte("hello")), memoInner0["MemoData"])
	assert.Equal(t, hex.EncodeToString([]byte("text/plain")), memoInner0["MemoFormat"])
	assert.Equal(t, hex.EncodeToString([]byte("message")), memoInner0["MemoType"])

	// Verify second memo has empty format omitted
	memo1, ok := memos[1].(map[string]interface{})
	require.True(t, ok)
	memoInner1, ok := memo1["Memo"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, hex.EncodeToString([]byte{0xDE, 0xAD}), memoInner1["MemoData"])
	assert.Equal(t, hex.EncodeToString([]byte("bin")), memoInner1["MemoType"])
}

func TestBuildPaymentTransactionNoMemos(t *testing.T) {
	payload := &vaa.XRPLReleasePayload{
		TicketID:       42,
		CustodyAccount: testCustodyAccountID,
		Recipient:      testRecipientAccountID,
		Amount:         1000000,
		TokenDecimals:  6,
		SourceChain:    vaa.ChainIDSolana,
		SourceEmitter:  testSourceEmitter,
		SourceSequence: 100,
		Token: vaa.XRPLTokenID{
			Type: vaa.XRPLTokenTypeXRP,
		},
	}

	flatTx, err := BuildPaymentTransaction(payload, testManagerSetM)
	require.NoError(t, err)

	// Memos should not be in the flat transaction when empty
	_, ok := flatTx["Memos"]
	assert.False(t, ok, "Memos should not be present when empty")
}

func TestComputeMultisignHash(t *testing.T) {
	payload := &vaa.XRPLReleasePayload{
		TicketID:       42,
		CustodyAccount: testCustodyAccountID,
		Recipient:      testRecipientAccountID,
		Amount:         1000000,
		TokenDecimals:  6,
		SourceChain:    vaa.ChainIDSolana,
		SourceEmitter:  testSourceEmitter,
		SourceSequence: 100,
		Token: vaa.XRPLTokenID{
			Type: vaa.XRPLTokenTypeXRP,
		},
	}

	flatTx, err := BuildPaymentTransaction(payload, testManagerSetM)
	require.NoError(t, err)

	// Get the signer address from the custody account
	signerAddr, err := AccountIDToAddress(testCustodyAccountID[:])
	require.NoError(t, err)

	hash, err := ComputeMultisignHash(flatTx, signerAddr)
	require.NoError(t, err)
	assert.Len(t, hash, 32)

	// Different signer address should produce different hash
	otherAddr, err := AccountIDToAddress(testRecipientAccountID[:])
	require.NoError(t, err)

	// Need to rebuild flatTx because EncodeForMultisigning modifies the map
	flatTx2, err := BuildPaymentTransaction(payload, testManagerSetM)
	require.NoError(t, err)

	hash2, err := ComputeMultisignHash(flatTx2, otherAddr)
	require.NoError(t, err)
	assert.Len(t, hash2, 32)

	// Hashes should differ for different signers
	assert.NotEqual(t, hash, hash2)
}

func TestEncodeDERSignature(t *testing.T) {
	// Test with known r and s values
	r := make([]byte, 32)
	s := make([]byte, 32)
	for i := range r {
		r[i] = byte(i + 1)
		s[i] = byte(i + 33)
	}

	derSig := EncodeDERSignature(r, s)

	// Verify DER structure
	assert.Equal(t, byte(0x30), derSig[0]) // Sequence tag
	assert.Equal(t, byte(0x02), derSig[2]) // Integer tag for r

	// Should NOT have sighash type byte at the end (unlike Dogecoin)
	// The last byte should be part of the s value, not a sighash type
	totalLen := derSig[1]
	assert.Equal(t, int(totalLen)+2, len(derSig)) // 0x30 + len + content = total
}

func TestEncodeDERSignatureWithHighBit(t *testing.T) {
	// Test with high bit set on r (should prepend 0x00)
	r := []byte{0x80, 0x01, 0x02}
	s := []byte{0x01, 0x02, 0x03}

	derSig := EncodeDERSignature(r, s)

	// r should have 0x00 prepended
	assert.Equal(t, byte(0x30), derSig[0])
	assert.Equal(t, byte(0x02), derSig[2])
	rLen := derSig[3]
	assert.Equal(t, byte(4), rLen) // 3 bytes + 1 padding
	assert.Equal(t, byte(0x00), derSig[4])
	assert.Equal(t, byte(0x80), derSig[5])
}

func TestEncodeDERSignatureStripLeadingZeros(t *testing.T) {
	// Test stripping leading zeros
	r := []byte{0x00, 0x00, 0x01, 0x02}
	s := []byte{0x00, 0x03, 0x04}

	derSig := EncodeDERSignature(r, s)

	assert.Equal(t, byte(0x30), derSig[0])
	// r should be [0x01, 0x02]
	rLen := derSig[3]
	assert.Equal(t, byte(2), rLen)
	assert.Equal(t, byte(0x01), derSig[4])
	assert.Equal(t, byte(0x02), derSig[5])
}

func TestAccountIDToAddress(t *testing.T) {
	// Test with a known account ID
	addr, err := AccountIDToAddress(testCustodyAccountID[:])
	require.NoError(t, err)
	assert.NotEmpty(t, addr)
	// Should start with 'r' (XRPL classic address prefix)
	assert.Equal(t, byte('r'), addr[0])
}

func TestAccountIDToAddressInvalidLength(t *testing.T) {
	_, err := AccountIDToAddress([]byte{0x01, 0x02, 0x03})
	require.ErrorContains(t, err, "invalid account ID length")
}

func TestCompressedPubKeyToAddress(t *testing.T) {
	// Use a known compressed public key
	pubKey, err := hex.DecodeString("0279BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798")
	require.NoError(t, err)

	addr, err := CompressedPubKeyToAddress(pubKey)
	require.NoError(t, err)
	assert.NotEmpty(t, addr)
	assert.Equal(t, byte('r'), addr[0])
}

func TestCompressedPubKeyToAddressInvalidLength(t *testing.T) {
	_, err := CompressedPubKeyToAddress([]byte{0x02, 0x03})
	require.ErrorContains(t, err, "invalid compressed public key length")
}

func TestSHA512Half(t *testing.T) {
	data := []byte("test data")
	hash := SHA512Half(data)
	assert.Len(t, hash, 32)

	// Same input should produce same output
	hash2 := SHA512Half(data)
	assert.Equal(t, hash, hash2)

	// Different input should produce different output
	hash3 := SHA512Half([]byte("different data"))
	assert.NotEqual(t, hash, hash3)
}

func TestFormatDecimalAmount(t *testing.T) {
	tests := []struct {
		name     string
		amount   uint64
		decimals uint8
		expected string
	}{
		{"zero decimals", 12345, 0, "12345"},
		{"6 decimals", 1000000, 6, "1.000000"},
		{"6 decimals fractional", 1234567, 6, "1.234567"},
		{"8 decimals", 100000000, 8, "1.00000000"},
		{"small amount", 1, 6, "0.000001"},
		{"zero amount", 0, 6, "0.000000"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := formatDecimalAmount(tc.amount, tc.decimals)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestEncodeCurrency(t *testing.T) {
	tests := []struct {
		name     string
		currency [20]byte
		expected string
	}{
		{
			name: "standard USD",
			currency: [20]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x55, 0x53, 0x44, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			expected: "USD",
		},
		{
			name: "standard EUR",
			currency: [20]byte{
				0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
				0x00, 0x00, 0x45, 0x55, 0x52, 0x00, 0x00, 0x00, 0x00, 0x00,
			},
			expected: "EUR",
		},
		{
			name: "non-standard hex currency",
			currency: [20]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a,
				0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14,
			},
			expected: "0102030405060708090a0b0c0d0e0f1011121314",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := encodeCurrency(tc.currency)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestBuildPaymentTransactionFee(t *testing.T) {
	// Test with different N values
	tests := []struct {
		name        string
		n           uint8
		expectedFee string
	}{
		{"N=7", 7, "96"},    // (7+1)*12 = 96
		{"N=13", 13, "168"}, // (13+1)*12 = 168
		{"N=1", 1, "24"},    // (1+1)*12 = 24
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := &vaa.XRPLReleasePayload{
				TicketID:       1,
				CustodyAccount: testCustodyAccountID,
				Recipient:      testRecipientAccountID,
				Amount:         1000000,
				TokenDecimals:  6,
				SourceChain:    vaa.ChainIDSolana,
				SourceEmitter:  testSourceEmitter,
				SourceSequence: 1,
				Token:          vaa.XRPLTokenID{Type: vaa.XRPLTokenTypeXRP},
			}

			flatTx, err := BuildPaymentTransaction(payload, tc.n)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedFee, flatTx["Fee"])
		})
	}
}

func TestBuildTicketCreateTransaction(t *testing.T) {
	payload := &vaa.XRPLTicketRefillPayload{
		Account:      testCustodyAccountID,
		UseTicket:    42,
		RequestCount: 10,
	}

	flatTx, err := BuildTicketCreateTransaction(payload, testManagerSetM)
	require.NoError(t, err)

	assert.Equal(t, "TicketCreate", flatTx["TransactionType"])
	assert.Equal(t, uint32(0), flatTx["Sequence"])
	assert.Equal(t, "", flatTx["SigningPubKey"])
	assert.Equal(t, uint32(42), flatTx["TicketSequence"])
	assert.Equal(t, uint32(10), flatTx["TicketCount"])

	// Fee = 12 * (M+1) = 12 * 8 = 96
	assert.Equal(t, "96", flatTx["Fee"])

	// Verify the account address
	expectedAddr, err := AccountIDToAddress(testCustodyAccountID[:])
	require.NoError(t, err)
	assert.Equal(t, expectedAddr, flatTx["Account"])
}

func TestBuildTicketCreateTransactionFee(t *testing.T) {
	tests := []struct {
		name        string
		m           uint8
		expectedFee string
	}{
		{"m=1", 1, "24"},    // 12 * (1+1) = 24
		{"m=2", 2, "36"},    // 12 * (2+1) = 36
		{"m=7", 7, "96"},    // 12 * (7+1) = 96
		{"m=10", 10, "132"}, // 12 * (10+1) = 132
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := &vaa.XRPLTicketRefillPayload{
				Account:      testCustodyAccountID,
				UseTicket:    1,
				RequestCount: 5,
			}

			flatTx, err := BuildTicketCreateTransaction(payload, tc.m)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedFee, flatTx["Fee"])
		})
	}
}

func TestBuildBurnTicketTransaction(t *testing.T) {
	payload := &vaa.XRPLBurnTicketPayload{
		Account:  testCustodyAccountID,
		TicketID: 42,
	}

	flatTx, err := BuildBurnTicketTransaction(payload, testManagerSetM)
	require.NoError(t, err)

	assert.Equal(t, "AccountSet", flatTx["TransactionType"])
	assert.Equal(t, uint32(0), flatTx["Sequence"])
	assert.Equal(t, "", flatTx["SigningPubKey"])
	assert.Equal(t, uint32(42), flatTx["TicketSequence"])

	// Fee = 15 * (M+1) = 15 * 8 = 120
	assert.Equal(t, "120", flatTx["Fee"])

	// Verify the account address
	expectedAddr, err := AccountIDToAddress(testCustodyAccountID[:])
	require.NoError(t, err)
	assert.Equal(t, expectedAddr, flatTx["Account"])
}

func TestBuildBurnTicketTransactionFee(t *testing.T) {
	tests := []struct {
		name        string
		m           uint8
		expectedFee string
	}{
		{"m=1", 1, "30"},    // 15 * (1+1) = 30
		{"m=2", 2, "45"},    // 15 * (2+1) = 45
		{"m=7", 7, "120"},   // 15 * (7+1) = 120
		{"m=10", 10, "165"}, // 15 * (10+1) = 165
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			payload := &vaa.XRPLBurnTicketPayload{
				Account:  testCustodyAccountID,
				TicketID: 1,
			}

			flatTx, err := BuildBurnTicketTransaction(payload, tc.m)
			require.NoError(t, err)
			assert.Equal(t, tc.expectedFee, flatTx["Fee"])
		})
	}
}
