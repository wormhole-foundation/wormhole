package dogecoin

import (
	"testing"

	"github.com/btcsuite/btcd/txscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestBuildRedeemScript(t *testing.T) {
	emitterChain := vaa.ChainIDSolana
	var emitterContract vaa.Address
	copy(emitterContract[:], mustDecodeHex(t, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"))

	var recipientAddress [32]byte
	copy(recipientAddress[:], mustDecodeHex(t, "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"))

	pubkey1 := mustDecodeHex(t, "02a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af9575")
	pubkey2 := mustDecodeHex(t, "036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d")

	script, err := BuildRedeemScript(
		emitterChain,
		emitterContract,
		recipientAddress,
		2,
		[][]byte{pubkey1, pubkey2},
	)
	require.NoError(t, err)
	require.NotEmpty(t, script)

	// Verify script length is reasonable (within P2SH limits of 520 bytes)
	assert.LessOrEqual(t, len(script), 520, "redeem script should be within P2SH limits")

	// Disassemble the script to verify structure
	disasm, err := txscript.DisasmString(script)
	require.NoError(t, err)

	// Should contain the key components
	assert.Contains(t, disasm, "OP_2DROP", "should have OP_2DROP for emitter info")
	assert.Contains(t, disasm, "OP_DROP", "should have OP_DROP for recipient address")
	assert.Contains(t, disasm, "OP_CHECKMULTISIG", "should end with OP_CHECKMULTISIG")
}

func TestBuildRedeemScriptMOfN(t *testing.T) {
	emitterChain := vaa.ChainIDSolana
	var emitterContract vaa.Address
	var recipientAddress [32]byte

	pubkeys := make([][]byte, 7)
	for i := range pubkeys {
		pubkeys[i] = make([]byte, 33)
		pubkeys[i][0] = 0x02 // Compressed pubkey prefix
		pubkeys[i][1] = byte(i + 1)
	}

	// Test 5-of-7 multisig
	script, err := BuildRedeemScript(
		emitterChain,
		emitterContract,
		recipientAddress,
		5,
		pubkeys,
	)
	require.NoError(t, err)

	// Should be within limits
	assert.LessOrEqual(t, len(script), 520)

	disasm, err := txscript.DisasmString(script)
	require.NoError(t, err)

	// Should have M (5) and N (7) in the script
	assert.Contains(t, disasm, "5")
	assert.Contains(t, disasm, "7")
}

func TestBuildRedeemScriptInvalidM(t *testing.T) {
	emitterChain := vaa.ChainIDSolana
	var emitterContract vaa.Address
	var recipientAddress [32]byte

	pubkey := mustDecodeHex(t, "02a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af9575")

	// M = 0 is invalid
	_, err := BuildRedeemScript(
		emitterChain,
		emitterContract,
		recipientAddress,
		0,
		[][]byte{pubkey},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid m-of-n")

	// M > N is invalid
	_, err = BuildRedeemScript(
		emitterChain,
		emitterContract,
		recipientAddress,
		2,
		[][]byte{pubkey},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid m-of-n")
}

func TestBuildRedeemScriptTooManyPubkeys(t *testing.T) {
	emitterChain := vaa.ChainIDSolana
	var emitterContract vaa.Address
	var recipientAddress [32]byte

	// Create 16 pubkeys (more than the 15 limit for standard multisig)
	pubkeys := make([][]byte, 16)
	for i := range pubkeys {
		pubkeys[i] = make([]byte, 33)
		pubkeys[i][0] = 0x02
		pubkeys[i][1] = byte(i + 1)
	}

	_, err := BuildRedeemScript(
		emitterChain,
		emitterContract,
		recipientAddress,
		10,
		pubkeys,
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "too many pubkeys")
}

func TestBuildRedeemScriptInvalidPubkeyLength(t *testing.T) {
	emitterChain := vaa.ChainIDSolana
	var emitterContract vaa.Address
	var recipientAddress [32]byte

	// Pubkey with wrong length (should be 33 for compressed)
	invalidPubkey := make([]byte, 32)

	_, err := BuildRedeemScript(
		emitterChain,
		emitterContract,
		recipientAddress,
		1,
		[][]byte{invalidPubkey},
	)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid length")
}

func TestBuildP2PKHScriptPubKey(t *testing.T) {
	pubkeyHash := mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105")

	script, err := BuildP2PKHScriptPubKey(pubkeyHash)
	require.NoError(t, err)

	// P2PKH: OP_DUP OP_HASH160 <20-byte hash> OP_EQUALVERIFY OP_CHECKSIG
	disasm, err := txscript.DisasmString(script)
	require.NoError(t, err)

	assert.Contains(t, disasm, "OP_DUP")
	assert.Contains(t, disasm, "OP_HASH160")
	assert.Contains(t, disasm, "OP_EQUALVERIFY")
	assert.Contains(t, disasm, "OP_CHECKSIG")
}

func TestBuildP2PKHScriptPubKeyInvalidLength(t *testing.T) {
	// Wrong length (should be 20)
	pubkeyHash := mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb6410511")

	_, err := BuildP2PKHScriptPubKey(pubkeyHash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be 20 bytes")
}

func TestBuildP2SHScriptPubKey(t *testing.T) {
	scriptHash := mustDecodeHex(t, "748284390f9e263a4b766a75d0633c50426eb875")

	script, err := BuildP2SHScriptPubKey(scriptHash)
	require.NoError(t, err)

	// P2SH: OP_HASH160 <20-byte hash> OP_EQUAL
	disasm, err := txscript.DisasmString(script)
	require.NoError(t, err)

	assert.Contains(t, disasm, "OP_HASH160")
	assert.Contains(t, disasm, "OP_EQUAL")
}

func TestBuildP2SHScriptPubKeyInvalidLength(t *testing.T) {
	// Wrong length
	scriptHash := mustDecodeHex(t, "748284390f9e263a4b766a75d0633c50426eb8")

	_, err := BuildP2SHScriptPubKey(scriptHash)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "must be 20 bytes")
}

func TestBuildScriptPubKey(t *testing.T) {
	tests := []struct {
		name        string
		addrType    vaa.UTXOAddressType
		address     []byte
		shouldError bool
		checkDisasm func(t *testing.T, disasm string)
	}{
		{
			name:        "P2PKH",
			addrType:    vaa.UTXOAddressTypeP2PKH,
			address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			shouldError: false,
			checkDisasm: func(t *testing.T, disasm string) {
				assert.Contains(t, disasm, "OP_DUP")
				assert.Contains(t, disasm, "OP_CHECKSIG")
			},
		},
		{
			name:        "P2SH",
			addrType:    vaa.UTXOAddressTypeP2SH,
			address:     mustDecodeHex(t, "748284390f9e263a4b766a75d0633c50426eb875"),
			shouldError: false,
			checkDisasm: func(t *testing.T, disasm string) {
				assert.Contains(t, disasm, "OP_HASH160")
				assert.Contains(t, disasm, "OP_EQUAL")
			},
		},
		{
			name:        "unsupported type",
			addrType:    99, // Invalid type
			address:     mustDecodeHex(t, "55ae51684c43435da751ac8d2173b2652eb64105"),
			shouldError: true,
			checkDisasm: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			script, err := BuildScriptPubKey(tc.addrType, tc.address)

			if tc.shouldError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			disasm, err := txscript.DisasmString(script)
			require.NoError(t, err)
			tc.checkDisasm(t, disasm)
		})
	}
}

func TestRedeemScriptDeterministic(t *testing.T) {
	emitterChain := vaa.ChainIDSolana
	var emitterContract vaa.Address
	copy(emitterContract[:], mustDecodeHex(t, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20"))

	var recipientAddress [32]byte
	copy(recipientAddress[:], mustDecodeHex(t, "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"))

	pubkey1 := mustDecodeHex(t, "02a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af9575")
	pubkey2 := mustDecodeHex(t, "036ce31db9bdd543e72fe3039a1f1c047dab87037c36a669ff90e28da1848f640d")

	// Build the same script twice
	script1, err := BuildRedeemScript(emitterChain, emitterContract, recipientAddress, 2, [][]byte{pubkey1, pubkey2})
	require.NoError(t, err)

	script2, err := BuildRedeemScript(emitterChain, emitterContract, recipientAddress, 2, [][]byte{pubkey1, pubkey2})
	require.NoError(t, err)

	// Should be exactly the same (deterministic)
	assert.Equal(t, script1, script2, "redeem script must be deterministic")
}

func TestRedeemScriptEmitterChainEncoding(t *testing.T) {
	// Test that emitter chain is encoded as big-endian
	emitterChain := vaa.ChainIDSolana // Chain ID 1
	var emitterContract vaa.Address
	var recipientAddress [32]byte

	pubkey := mustDecodeHex(t, "02a882d414e478039cd5b52a92ffb13dd5e6bd4515497439dffd691a0f12af9575")

	script, err := BuildRedeemScript(emitterChain, emitterContract, recipientAddress, 1, [][]byte{pubkey})
	require.NoError(t, err)

	// The emitter chain should be at the beginning after the push opcode
	// Script starts with: OP_PUSHDATA <chain bytes>
	// For chain ID 1: 0x00 0x01 (big-endian)
	assert.Equal(t, byte(0x02), script[0], "first byte should be OP_PUSHBYTES_2")
	assert.Equal(t, byte(0x00), script[1], "high byte of chain ID 1")
	assert.Equal(t, byte(0x01), script[2], "low byte of chain ID 1")
}
