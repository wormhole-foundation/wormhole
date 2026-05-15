package dogecoin

import (
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/txscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Test helper to create a hex-decoded byte slice
func mustDecodeHex(t *testing.T, s string) []byte {
	b, err := hex.DecodeString(s)
	require.NoError(t, err)
	return b
}

func TestBuildUnsignedTransaction(t *testing.T) {
	// Create a test UTXO unlock payload
	var txid [32]byte
	copy(txid[:], mustDecodeHex(t, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"))

	var recipientAddr [32]byte
	copy(recipientAddr[:], mustDecodeHex(t, "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"))

	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []vaa.UTXOInput{
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid,
				Vout:                     0,
			},
		},
		Outputs: []vaa.UTXOOutput{
			{
				Amount:      1000000, // 0.01 DOGE
				AddressType: vaa.UTXOAddressTypeP2PKH,
				Address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			},
		},
	}

	// Create a dummy redeem script
	redeemScript := mustDecodeHex(t, "522103a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af957521036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d52ae")

	tx, err := BuildUnsignedTransaction(payload, redeemScript)
	require.NoError(t, err)
	require.NotNil(t, tx)

	// Verify transaction structure
	assert.Equal(t, 1, tx.InputCount())
	assert.Equal(t, 1, tx.OutputCount())

	// Verify input
	assert.NotNil(t, tx.Tx.TxIn[0].PreviousOutPoint)

	// Verify output
	assert.Equal(t, int64(1000000), tx.Tx.TxOut[0].Value)
}

func TestBuildUnsignedTransactionMultipleInputsOutputs(t *testing.T) {
	var txid1, txid2 [32]byte
	copy(txid1[:], mustDecodeHex(t, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"))
	copy(txid2[:], mustDecodeHex(t, "2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40"))

	var recipientAddr [32]byte
	copy(recipientAddr[:], mustDecodeHex(t, "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"))

	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []vaa.UTXOInput{
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid1,
				Vout:                     0,
			},
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid2,
				Vout:                     1,
			},
		},
		Outputs: []vaa.UTXOOutput{
			{
				Amount:      1000000,
				AddressType: vaa.UTXOAddressTypeP2PKH,
				Address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			},
			{
				Amount:      500000,
				AddressType: vaa.UTXOAddressTypeP2SH,
				Address:     mustDecodeHex(t, "748284390f9e263a4b766a75d0633c50426eb875"),
			},
		},
	}

	redeemScript := mustDecodeHex(t, "522103a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af957521036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d52ae")

	tx, err := BuildUnsignedTransaction(payload, redeemScript)
	require.NoError(t, err)

	assert.Equal(t, 2, tx.InputCount())
	assert.Equal(t, 2, tx.OutputCount())

	// Verify both redeem scripts are set
	assert.Len(t, tx.RedeemScripts, 2)
	for i, rs := range tx.RedeemScripts {
		assert.Equal(t, redeemScript, rs, "redeem script %d should match", i)
	}
}

func TestBuildUnsignedTransactionNoInputs(t *testing.T) {
	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs:                   []vaa.UTXOInput{},
		Outputs: []vaa.UTXOOutput{
			{
				Amount:      1000000,
				AddressType: vaa.UTXOAddressTypeP2PKH,
				Address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			},
		},
	}

	_, err := BuildUnsignedTransaction(payload, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no inputs")
}

func TestBuildUnsignedTransactionNoOutputs(t *testing.T) {
	var txid [32]byte
	var recipientAddr [32]byte

	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []vaa.UTXOInput{
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid,
				Vout:                     0,
			},
		},
		Outputs: []vaa.UTXOOutput{},
	}

	_, err := BuildUnsignedTransaction(payload, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no outputs")
}

func TestComputeSighash(t *testing.T) {
	var txid [32]byte
	copy(txid[:], mustDecodeHex(t, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"))

	var recipientAddr [32]byte
	copy(recipientAddr[:], mustDecodeHex(t, "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"))

	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []vaa.UTXOInput{
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid,
				Vout:                     0,
			},
		},
		Outputs: []vaa.UTXOOutput{
			{
				Amount:      1000000,
				AddressType: vaa.UTXOAddressTypeP2PKH,
				Address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			},
		},
	}

	redeemScript := mustDecodeHex(t, "522103a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af957521036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d52ae")

	tx, err := BuildUnsignedTransaction(payload, redeemScript)
	require.NoError(t, err)

	// Compute sighash for input 0
	sighash, err := tx.ComputeSighash(0, txscript.SigHashAll)
	require.NoError(t, err)
	require.Len(t, sighash, 32)

	// Verify it's not all zeros
	allZeros := true
	for _, b := range sighash {
		if b != 0 {
			allZeros = false
			break
		}
	}
	assert.False(t, allZeros, "sighash should not be all zeros")
}

func TestComputeSighashOutOfRange(t *testing.T) {
	var txid [32]byte
	var recipientAddr [32]byte

	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []vaa.UTXOInput{
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid,
				Vout:                     0,
			},
		},
		Outputs: []vaa.UTXOOutput{
			{
				Amount:      1000000,
				AddressType: vaa.UTXOAddressTypeP2PKH,
				Address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			},
		},
	}

	redeemScript := mustDecodeHex(t, "522103a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af957521036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d52ae")

	tx, err := BuildUnsignedTransaction(payload, redeemScript)
	require.NoError(t, err)

	// Try to compute sighash for invalid index
	_, err = tx.ComputeSighash(1, txscript.SigHashAll)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")

	_, err = tx.ComputeSighash(-1, txscript.SigHashAll)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}

func TestComputeAllSighashes(t *testing.T) {
	var txid1, txid2 [32]byte
	copy(txid1[:], mustDecodeHex(t, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"))
	copy(txid2[:], mustDecodeHex(t, "2122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d3e3f40"))

	var recipientAddr [32]byte

	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []vaa.UTXOInput{
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid1,
				Vout:                     0,
			},
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid2,
				Vout:                     1,
			},
		},
		Outputs: []vaa.UTXOOutput{
			{
				Amount:      1000000,
				AddressType: vaa.UTXOAddressTypeP2PKH,
				Address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			},
		},
	}

	redeemScript := mustDecodeHex(t, "522103a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af957521036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d52ae")

	tx, err := BuildUnsignedTransaction(payload, redeemScript)
	require.NoError(t, err)

	sighashes, err := tx.ComputeAllSighashes(txscript.SigHashAll)
	require.NoError(t, err)
	require.Len(t, sighashes, 2)

	// Each sighash should be different (different inputs)
	assert.NotEqual(t, sighashes[0], sighashes[1])
}

func TestSerializeForBroadcast(t *testing.T) {
	var txid [32]byte
	var recipientAddr [32]byte

	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []vaa.UTXOInput{
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid,
				Vout:                     0,
			},
		},
		Outputs: []vaa.UTXOOutput{
			{
				Amount:      1000000,
				AddressType: vaa.UTXOAddressTypeP2PKH,
				Address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			},
		},
	}

	redeemScript := mustDecodeHex(t, "522103a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af957521036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d52ae")

	tx, err := BuildUnsignedTransaction(payload, redeemScript)
	require.NoError(t, err)

	serialized, err := tx.SerializeForBroadcast()
	require.NoError(t, err)
	assert.NotEmpty(t, serialized)

	// Basic transaction structure check:
	// Version (4 bytes) + input count (1 byte varint) + ... + lock time (4 bytes)
	assert.GreaterOrEqual(t, len(serialized), 10)
}

func TestDoubleSha256(t *testing.T) {
	input := []byte("hello")
	result := DoubleSha256(input)

	// Known double SHA256 of "hello"
	expected := mustDecodeHex(t, "9595c9df90075148eb06860365df33584b75bff782a510c6cd4883a419833d50")
	assert.Equal(t, expected, result)
}

func TestHash160(t *testing.T) {
	input := []byte("hello")
	result := Hash160(input)

	assert.Len(t, result, 20)
	// Hash160 = RIPEMD160(SHA256(data))
	// SHA256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	// RIPEMD160 of that = b6a9c8c230722b7c748331a8b450f05566dc7d0f
	expected := mustDecodeHex(t, "b6a9c8c230722b7c748331a8b450f05566dc7d0f")
	assert.Equal(t, expected, result)
}

func TestBuildP2SHAddress(t *testing.T) {
	// A simple 2-of-2 multisig redeem script
	redeemScript := mustDecodeHex(t, "522103a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af957521036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d52ae")

	address := BuildP2SHAddress(redeemScript)
	assert.Len(t, address, 20)
}

func TestEncodeDERSignature(t *testing.T) {
	// Test case with known r and s values
	r := mustDecodeHex(t, "6e7a6e6f7d8e8f909192939495969798999a9b9c9d9e9fa0a1a2a3a4a5a6a7a8")
	s := mustDecodeHex(t, "1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d")

	sig := EncodeDERSignature(r, s, txscript.SigHashAll)

	// DER format: 0x30 [len] 0x02 [r-len] [r] 0x02 [s-len] [s] [sighash]
	assert.Equal(t, byte(0x30), sig[0], "should start with DER sequence tag")
	assert.Equal(t, byte(0x02), sig[2], "r should be tagged as integer")
	assert.Equal(t, byte(txscript.SigHashAll), sig[len(sig)-1], "should end with sighash type")
}

func TestEncodeDERSignatureWithLeadingZeros(t *testing.T) {
	// Test with r that has leading zeros (should be stripped)
	r := mustDecodeHex(t, "00006e7a6e6f7d8e8f909192939495969798999a9b9c9d9e9fa0a1a2a3a4a5a6")
	s := mustDecodeHex(t, "1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d")

	sig := EncodeDERSignature(r, s, txscript.SigHashAll)

	// Verify it's a valid DER signature
	assert.Equal(t, byte(0x30), sig[0])
	assert.Equal(t, byte(0x02), sig[2])
}

func TestEncodeDERSignatureHighBit(t *testing.T) {
	// Test with r that has high bit set (should add 0x00 prefix)
	r := mustDecodeHex(t, "8e7a6e6f7d8e8f909192939495969798999a9b9c9d9e9fa0a1a2a3a4a5a6a7a8")
	s := mustDecodeHex(t, "1e1f202122232425262728292a2b2c2d2e2f303132333435363738393a3b3c3d")

	sig := EncodeDERSignature(r, s, txscript.SigHashAll)

	// Find r in the signature
	rLen := int(sig[3])
	rValue := sig[4 : 4+rLen]

	// Should have 0x00 prefix to prevent negative interpretation
	assert.Equal(t, byte(0x00), rValue[0], "high-bit r should have 0x00 prefix")
}

func TestCanonicalizeInt(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "no change needed",
			input:    []byte{0x45, 0x67},
			expected: []byte{0x45, 0x67},
		},
		{
			name:     "strip leading zeros",
			input:    []byte{0x00, 0x00, 0x45, 0x67},
			expected: []byte{0x45, 0x67},
		},
		{
			name:     "preserve zero needed for high bit",
			input:    []byte{0x00, 0x85, 0x67},
			expected: []byte{0x00, 0x85, 0x67},
		},
		{
			name:     "add zero for high bit",
			input:    []byte{0x85, 0x67},
			expected: []byte{0x00, 0x85, 0x67},
		},
		{
			name:     "single byte with high bit",
			input:    []byte{0x80},
			expected: []byte{0x00, 0x80},
		},
		{
			name:     "single zero byte",
			input:    []byte{0x00},
			expected: []byte{0x00},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := canonicalizeInt(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestVoutToBytes(t *testing.T) {
	tests := []struct {
		vout     uint32
		expected []byte
	}{
		{0, []byte{0x00, 0x00, 0x00, 0x00}},
		{1, []byte{0x01, 0x00, 0x00, 0x00}},
		{256, []byte{0x00, 0x01, 0x00, 0x00}},
		{0xFFFFFFFF, []byte{0xFF, 0xFF, 0xFF, 0xFF}},
	}

	for _, tc := range tests {
		result := VoutToBytes(tc.vout)
		assert.Equal(t, tc.expected, result, "vout %d", tc.vout)
	}
}

func TestApplySignatureToInput(t *testing.T) {
	var txid [32]byte
	var recipientAddr [32]byte

	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []vaa.UTXOInput{
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid,
				Vout:                     0,
			},
		},
		Outputs: []vaa.UTXOOutput{
			{
				Amount:      1000000,
				AddressType: vaa.UTXOAddressTypeP2PKH,
				Address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			},
		},
	}

	redeemScript := mustDecodeHex(t, "522103a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af957521036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d52ae")

	tx, err := BuildUnsignedTransaction(payload, redeemScript)
	require.NoError(t, err)

	// Create dummy signatures
	sig1 := mustDecodeHex(t, "304402201234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef02201234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef01")
	sig2 := mustDecodeHex(t, "304502201234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef022100fedcba0987654321fedcba0987654321fedcba0987654321fedcba098765432101")

	err = ApplySignatureToInput(tx.Tx, 0, [][]byte{sig1, sig2}, redeemScript)
	require.NoError(t, err)

	// Verify scriptSig was set
	assert.NotEmpty(t, tx.Tx.TxIn[0].SignatureScript)

	// Verify scriptSig starts with OP_0 (for CHECKMULTISIG bug)
	assert.Equal(t, byte(txscript.OP_0), tx.Tx.TxIn[0].SignatureScript[0])
}

func TestApplySignatureToInputOutOfRange(t *testing.T) {
	var txid [32]byte
	var recipientAddr [32]byte

	payload := &vaa.UTXOUnlockPayload{
		DestinationChain:         vaa.ChainIDDogecoin,
		DelegatedManagerSetIndex: 1,
		Inputs: []vaa.UTXOInput{
			{
				OriginalRecipientAddress: recipientAddr,
				TransactionID:            txid,
				Vout:                     0,
			},
		},
		Outputs: []vaa.UTXOOutput{
			{
				Amount:      1000000,
				AddressType: vaa.UTXOAddressTypeP2PKH,
				Address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			},
		},
	}

	redeemScript := mustDecodeHex(t, "522103a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af957521036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d52ae")

	tx, err := BuildUnsignedTransaction(payload, redeemScript)
	require.NoError(t, err)

	err = ApplySignatureToInput(tx.Tx, 5, [][]byte{}, redeemScript)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "out of range")
}
