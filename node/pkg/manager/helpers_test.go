package manager

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/txscript"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// makeEthSig builds a 65-byte Ethereum-style signature (r || s || v=0x1b).
// r and s are 32-byte big-endian scalars; v is a recovery id that the
// functions under test do not inspect.
func makeEthSig(r, s []byte) []byte {
	out := make([]byte, 65)
	copy(out[0:32], r)
	copy(out[32:64], s)
	out[64] = 0x1b
	return out
}

// be32 left-pads to 32 bytes.
func be32(b []byte) []byte {
	out := make([]byte, 32)
	copy(out[32-len(b):], b)
	return out
}

// =============================================================================
// parseSequencer / parseEmitters
// =============================================================================

func TestParseSequencer_Empty(t *testing.T) {
	got := parseSequencer(struct {
		ChainId vaa.ChainID
		Addr    string
	}{ChainId: vaa.ChainIDXRPL, Addr: ""})
	assert.Nil(t, got)
}

func TestParseSequencer_Valid(t *testing.T) {
	addrHex := "00112233445566778899aabbccddeeff00112233"
	got := parseSequencer(struct {
		ChainId vaa.ChainID
		Addr    string
	}{ChainId: vaa.ChainIDXRPL, Addr: addrHex})
	require.NotNil(t, got)
	assert.Equal(t, vaa.ChainIDXRPL, got.chainId)
	expected, _ := hex.DecodeString(addrHex)
	assert.Equal(t, expected, got.addr)
}

func TestParseSequencer_InvalidHexPanics(t *testing.T) {
	assert.Panics(t, func() {
		parseSequencer(struct {
			ChainId vaa.ChainID
			Addr    string
		}{ChainId: vaa.ChainIDXRPL, Addr: "not-hex"})
	})
}

func TestParseEmitters_Empty(t *testing.T) {
	got := parseEmitters(nil)
	assert.Empty(t, got)
}

func TestParseEmitters_MultipleValid(t *testing.T) {
	in := []struct {
		ChainId vaa.ChainID
		Addr    string
	}{
		{vaa.ChainIDEthereum, "00000000000000000000000000000000000000000000000000000000000000aa"},
		{vaa.ChainIDSolana, "00000000000000000000000000000000000000000000000000000000000000bb"},
	}
	got := parseEmitters(in)
	require.Len(t, got, 2)
	assert.Equal(t, vaa.ChainIDEthereum, got[0].chainId)
	assert.Equal(t, vaa.ChainIDSolana, got[1].chainId)
}

func TestParseEmitters_InvalidHexPanics(t *testing.T) {
	assert.Panics(t, func() {
		parseEmitters([]struct {
			ChainId vaa.ChainID
			Addr    string
		}{{ChainId: vaa.ChainIDEthereum, Addr: "zz"}})
	})
}

// =============================================================================
// isXRPLSequencer / isKnownEmitter / validateEmitter
// =============================================================================

func TestIsXRPLSequencer(t *testing.T) {
	// Sequencer addresses are 32-byte emitter addresses (see sdk.KnownXRPLSequencer).
	seqAddr := bytes.Repeat([]byte{0xab}, 32)
	c := &ManagerService{
		xrplSequencer: &emitterEntry{chainId: vaa.ChainIDSolana, addr: seqAddr},
	}

	var addr vaa.Address
	copy(addr[:], seqAddr)

	assert.True(t, c.isXRPLSequencer(vaa.ChainIDSolana, addr))
	assert.False(t, c.isXRPLSequencer(vaa.ChainIDEthereum, addr), "wrong chain")

	var wrongAddr vaa.Address
	wrongAddr[31] = 0x01
	assert.False(t, c.isXRPLSequencer(vaa.ChainIDSolana, wrongAddr), "wrong addr")

	// nil sequencer
	c2 := &ManagerService{}
	assert.False(t, c2.isXRPLSequencer(vaa.ChainIDSolana, addr))
}

func TestIsKnownEmitter(t *testing.T) {
	knownAddr := bytes.Repeat([]byte{0x42}, 32)
	c := &ManagerService{
		emitters: []emitterEntry{
			{chainId: vaa.ChainIDEthereum, addr: knownAddr},
		},
	}

	var addr vaa.Address
	copy(addr[:], knownAddr)

	assert.True(t, c.isKnownEmitter(vaa.ChainIDEthereum, addr))
	assert.False(t, c.isKnownEmitter(vaa.ChainIDSolana, addr), "wrong chain")

	var wrongAddr vaa.Address
	wrongAddr[0] = 0x01
	assert.False(t, c.isKnownEmitter(vaa.ChainIDEthereum, wrongAddr), "wrong addr")
}

func TestValidateEmitter(t *testing.T) {
	xrplSeqAddr := bytes.Repeat([]byte{0xab}, 32)
	emitterAddr := bytes.Repeat([]byte{0x42}, 32)

	c := &ManagerService{
		xrplSequencer: &emitterEntry{chainId: vaa.ChainIDSolana, addr: xrplSeqAddr},
		emitters:      []emitterEntry{{chainId: vaa.ChainIDEthereum, addr: emitterAddr}},
	}

	var seqAddr vaa.Address
	copy(seqAddr[:], xrplSeqAddr)
	var ethAddr vaa.Address
	copy(ethAddr[:], emitterAddr)

	makeVAA := func(prefix [4]byte, chain vaa.ChainID, addr vaa.Address) *vaa.VAA {
		payload := make([]byte, 4)
		copy(payload, prefix[:])
		return &vaa.VAA{
			EmitterChain:   chain,
			EmitterAddress: addr,
			Payload:        payload,
		}
	}

	t.Run("XREL valid", func(t *testing.T) {
		assert.True(t, c.validateEmitter(makeVAA(vaa.XRPLPayloadPrefix, vaa.ChainIDSolana, seqAddr)))
	})
	t.Run("XRFL valid", func(t *testing.T) {
		assert.True(t, c.validateEmitter(makeVAA(vaa.XRPLTicketRefillPrefix, vaa.ChainIDSolana, seqAddr)))
	})
	t.Run("XBRN valid", func(t *testing.T) {
		assert.True(t, c.validateEmitter(makeVAA(vaa.XRPLBurnTicketPrefix, vaa.ChainIDSolana, seqAddr)))
	})
	t.Run("UTX0 valid", func(t *testing.T) {
		assert.True(t, c.validateEmitter(makeVAA(vaa.UTXOPayloadPrefix, vaa.ChainIDEthereum, ethAddr)))
	})
	t.Run("XREL wrong emitter", func(t *testing.T) {
		assert.False(t, c.validateEmitter(makeVAA(vaa.XRPLPayloadPrefix, vaa.ChainIDSolana, ethAddr)))
	})
	t.Run("UTX0 unknown emitter", func(t *testing.T) {
		assert.False(t, c.validateEmitter(makeVAA(vaa.UTXOPayloadPrefix, vaa.ChainIDSolana, ethAddr)))
	})
	t.Run("unknown prefix", func(t *testing.T) {
		assert.False(t, c.validateEmitter(makeVAA([4]byte{'Z', 'Z', 'Z', 'Z'}, vaa.ChainIDSolana, seqAddr)))
	})
}

// =============================================================================
// normalizeEthSig / convertEthSigToDER / convertEthSigToXRPLDER
// =============================================================================

// halfOrder is N/2 for secp256k1, computed via btcec.
func halfOrder(t *testing.T) []byte {
	t.Helper()
	// N/2 = 0x7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0
	out, err := hex.DecodeString("7fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a0")
	require.NoError(t, err)
	return out
}

func TestNormalizeEthSig_BadLength(t *testing.T) {
	for _, n := range []int{0, 1, 64, 66, 128} {
		_, _, err := normalizeEthSig(make([]byte, n))
		require.Error(t, err, "len=%d should error", n)
		assert.Contains(t, err.Error(), "invalid Ethereum signature length")
	}
}

func TestNormalizeEthSig_LowS_Unchanged(t *testing.T) {
	r := be32([]byte{0x01})
	s := be32([]byte{0x02}) // small s, well below N/2
	sig := makeEthSig(r, s)

	rOut, sOut, err := normalizeEthSig(sig)
	require.NoError(t, err)
	assert.Equal(t, r, rOut)
	assert.Equal(t, s, sOut)
}

func TestNormalizeEthSig_HighS_Negated(t *testing.T) {
	// s = N - 1 (max valid scalar) should be negated to 1.
	// We don't compute N explicitly; instead use halfOrder + 1 and verify the
	// returned s is strictly <= halfOrder.
	half := halfOrder(t)
	highS := make([]byte, 32)
	copy(highS, half)
	highS[31]++ // half + 1, just over half order

	sig := makeEthSig(be32([]byte{0x01}), highS)

	_, sOut, err := normalizeEthSig(sig)
	require.NoError(t, err)
	// After negation s must be <= N/2
	assert.NotEqual(t, highS, sOut, "high s should be negated")
	// And |sOut| <= half + 1 byte-wise (sOut is N - highS so it equals half - 1)
	// We assert byte-prefix: sOut must be lexicographically <= half
	assert.LessOrEqual(t, bytes.Compare(sOut, half), 0)
}

func TestNormalizeEthSig_OverflowR(t *testing.T) {
	// r = all-0xff is > N, so SetByteSlice reports overflow.
	r := bytes.Repeat([]byte{0xff}, 32)
	s := be32([]byte{0x01})
	sig := makeEthSig(r, s)

	_, _, err := normalizeEthSig(sig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "r value overflows")
}

func TestNormalizeEthSig_OverflowS(t *testing.T) {
	r := be32([]byte{0x01})
	s := bytes.Repeat([]byte{0xff}, 32)
	sig := makeEthSig(r, s)

	_, _, err := normalizeEthSig(sig)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "s value overflows")
}

func TestConvertEthSigToDER(t *testing.T) {
	sig := makeEthSig(be32([]byte{0x05}), be32([]byte{0x06}))
	der, err := convertEthSigToDER(sig, txscript.SigHashAll)
	require.NoError(t, err)
	// Last byte must be the sighash type appended.
	assert.Equal(t, byte(txscript.SigHashAll), der[len(der)-1])
	// DER signature must start with 0x30.
	assert.Equal(t, byte(0x30), der[0])
}

func TestConvertEthSigToDER_Error(t *testing.T) {
	_, err := convertEthSigToDER(make([]byte, 10), txscript.SigHashAll)
	require.Error(t, err)
}

func TestConvertEthSigToXRPLDER(t *testing.T) {
	sig := makeEthSig(be32([]byte{0x05}), be32([]byte{0x06}))
	der, err := convertEthSigToXRPLDER(sig)
	require.NoError(t, err)
	// DER signature must start with 0x30, no sighash byte for XRPL.
	assert.Equal(t, byte(0x30), der[0])
}

func TestConvertEthSigToXRPLDER_Error(t *testing.T) {
	_, err := convertEthSigToXRPLDER(make([]byte, 10))
	require.Error(t, err)
}

// =============================================================================
// GetFeatureString
// =============================================================================

func TestGetFeatureString_Empty(t *testing.T) {
	c := &ManagerService{}
	assert.Equal(t, "", c.GetFeatureString())
}

func TestGetFeatureString_SingleChain(t *testing.T) {
	pubKey := bytes.Repeat([]byte{0x01}, 33)
	c := &ManagerService{
		signerPubKeys: map[vaa.ChainID][][]byte{
			vaa.ChainIDXRPL: {pubKey},
		},
	}
	expected := "manager:" + chainIDFeaturePart(vaa.ChainIDXRPL, pubKey)
	assert.Equal(t, expected, c.GetFeatureString())
}

func TestGetFeatureString_MultipleChainsSorted(t *testing.T) {
	xrplKey := bytes.Repeat([]byte{0xaa}, 33)
	dogeKey := bytes.Repeat([]byte{0xbb}, 33)
	c := &ManagerService{
		signerPubKeys: map[vaa.ChainID][][]byte{
			vaa.ChainIDXRPL:     {xrplKey},
			vaa.ChainIDDogecoin: {dogeKey},
		},
	}

	got := c.GetFeatureString()
	require.Contains(t, got, "manager:")

	// The chain with the lower numeric ID must come first.
	low, high := vaa.ChainIDXRPL, vaa.ChainIDDogecoin
	lowKey, highKey := xrplKey, dogeKey
	if uint16(vaa.ChainIDDogecoin) < uint16(vaa.ChainIDXRPL) {
		low, high = vaa.ChainIDDogecoin, vaa.ChainIDXRPL
		lowKey, highKey = dogeKey, xrplKey
	}
	expected := "manager:" + chainIDFeaturePart(low, lowKey) + "|" + chainIDFeaturePart(high, highKey)
	assert.Equal(t, expected, got)
}

func chainIDFeaturePart(chain vaa.ChainID, pubKey []byte) string {
	return formatChainPubKey(uint16(chain), pubKey)
}

func formatChainPubKey(chainID uint16, pubKey []byte) string {
	return sprintfChainPubKey(chainID, pubKey)
}

func sprintfChainPubKey(chainID uint16, pubKey []byte) string {
	return itoa(int(chainID)) + "/" + hex.EncodeToString(pubKey)
}

// itoa is a small local helper to avoid pulling strconv in the test for one call.
func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var buf [20]byte
	n := len(buf)
	for i > 0 {
		n--
		buf[n] = byte('0' + i%10)
		i /= 10
	}
	if neg {
		n--
		buf[n] = '-'
	}
	return string(buf[n:])
}

// =============================================================================
// compressPublicKey (nil path)
// =============================================================================

func TestCompressPublicKey_Roundtrip(t *testing.T) {
	// Use a generator point so we get a known compressed form.
	pk, _ := btcec.PrivKeyFromBytes(be32([]byte{0x01}))
	pub := pk.PubKey().ToECDSA()

	out := compressPublicKey(pub)
	require.NotNil(t, out)
	assert.Len(t, out, 33)
	assert.Equal(t, pk.PubKey().SerializeCompressed(), out)
}
