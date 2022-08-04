package vaa

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChainIDFromString(t *testing.T) {
	type test struct {
		input  string
		output ChainID
	}

	// Positive Test Cases
	p_tests := []test{
		{input: "solana", output: ChainIDSolana},
		{input: "ethereum", output: ChainIDEthereum},
		{input: "terra", output: ChainIDTerra},
		{input: "bsc", output: ChainIDBSC},
		{input: "polygon", output: ChainIDPolygon},
		{input: "avalanche", output: ChainIDAvalanche},
		{input: "oasis", output: ChainIDOasis},
		{input: "algorand", output: ChainIDAlgorand},
		{input: "aurora", output: ChainIDAurora},
		{input: "fantom", output: ChainIDFantom},
		{input: "karura", output: ChainIDKarura},
		{input: "acala", output: ChainIDAcala},
		{input: "klaytn", output: ChainIDKlaytn},
		{input: "celo", output: ChainIDCelo},
		{input: "moonbeam", output: ChainIDMoonbeam},
		{input: "neon", output: ChainIDNeon},
		{input: "terra2", output: ChainIDTerra2},
		{input: "injective", output: ChainIDInjective},
		{input: "ethereum-ropsten", output: ChainIDEthereumRopsten},

		{input: "Solana", output: ChainIDSolana},
		{input: "Ethereum", output: ChainIDEthereum},
		{input: "Terra", output: ChainIDTerra},
		{input: "Bsc", output: ChainIDBSC},
		{input: "Polygon", output: ChainIDPolygon},
		{input: "Avalanche", output: ChainIDAvalanche},
		{input: "Oasis", output: ChainIDOasis},
		{input: "Algorand", output: ChainIDAlgorand},
		{input: "Aurora", output: ChainIDAurora},
		{input: "Fantom", output: ChainIDFantom},
		{input: "Karura", output: ChainIDKarura},
		{input: "Acala", output: ChainIDAcala},
		{input: "Klaytn", output: ChainIDKlaytn},
		{input: "Celo", output: ChainIDCelo},
		{input: "Moonbeam", output: ChainIDMoonbeam},
		{input: "Neon", output: ChainIDNeon},
		{input: "Terra2", output: ChainIDTerra2},
		{input: "Injective", output: ChainIDInjective},
		{input: "Ethereum-ropsten", output: ChainIDEthereumRopsten},
	}

	// Negative Test Cases
	n_tests := []test{
		{input: "Unknown", output: ChainIDUnset},
	}

	for _, tc := range p_tests {
		t.Run(tc.input, func(t *testing.T) {
			chainId, err := ChainIDFromString(tc.input)
			assert.Equal(t, tc.output, chainId)
			assert.NoError(t, err)
		})
	}

	for _, tc := range n_tests {
		t.Run(tc.input, func(t *testing.T) {
			chainId, err := ChainIDFromString(tc.input)
			assert.Equal(t, tc.output, chainId)
			assert.NotNil(t, err)
		})
	}
}

func TestAddress_MarshalJSON(t *testing.T) {
	addr := Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	expected := "223030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303422"
	marshalJsonAddress, err := addr.MarshalJSON()
	assert.Equal(t, hex.EncodeToString(marshalJsonAddress), expected)
	assert.NoError(t, err)
}

func TestAddress_String(t *testing.T) {
	addr := Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	expected := "0000000000000000000000000000000000000000000000000000000000000004"
	assert.Equal(t, addr.String(), expected)
}

func TestAddress_Bytes(t *testing.T) {
	addr := Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	expected := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4}
	assert.Equal(t, addr.Bytes(), expected)
}

func TestSignatureData_MarshalJSON(t *testing.T) {
	sigData := SignatureData{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0}
	marshalJsonSigData, err := sigData.MarshalJSON()
	require.Nil(t, err)

	expected := "223030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303430303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303034303022"
	assert.Equal(t, hex.EncodeToString(marshalJsonSigData), expected)
}

func TestSignature_DataString(t *testing.T) {
	sigData := SignatureData{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0}
	expected := "0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000400"
	assert.Equal(t, sigData.String(), expected)
}

func TestMinVAALength(t *testing.T) {
	assert.Equal(t, minVAALength, 57)
}

func TestChainId_String(t *testing.T) {
	type test struct {
		input  ChainID
		output string
	}

	tests := []test{
		{input: 0, output: "unset"},
		{input: 1, output: "solana"},
		{input: 2, output: "ethereum"},
		{input: 3, output: "terra"},
		{input: 4, output: "bsc"},
		{input: 5, output: "polygon"},
		{input: 6, output: "avalanche"},
		{input: 7, output: "oasis"},
		{input: 8, output: "algorand"},
		{input: 9, output: "aurora"},
		{input: 10, output: "fantom"},
		{input: 11, output: "karura"},
		{input: 12, output: "acala"},
		{input: 13, output: "klaytn"},
		{input: 14, output: "celo"},
		{input: 16, output: "moonbeam"},
		{input: 17, output: "neon"},
		{input: 18, output: "terra2"},
		{input: 19, output: "injective"},
		{input: 10001, output: "ethereum-ropsten"},
	}

	for _, tc := range tests {
		t.Run(tc.output, func(t *testing.T) {
			assert.Equal(t, ChainID(tc.input).String(), tc.output)
		})
	}
}

func getVaa() VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var governanceEmitter = Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	return VAA{
		Version:          uint8(1),
		GuardianSetIndex: uint32(1),
		Signatures:       []*Signature{},
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		ConsistencyLevel: uint8(32),
		EmitterChain:     ChainIDSolana,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}
}

func TestAddSignature(t *testing.T) {
	vaa := getVaa()

	// Generate a random private key to sign with
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.Equal(t, []*Signature{}, vaa.Signatures)

	// Add a signature and make sure it's added
	vaa.AddSignature(key, 0)
	assert.Equal(t, len(vaa.Signatures), 1)
}

func TestSerializeBody(t *testing.T) {
	vaa := getVaa()
	expected := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x20, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61}
	assert.Equal(t, vaa.serializeBody(), expected)
}

func TestSigningBody(t *testing.T) {
	vaa := getVaa()
	expected := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x20, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61}
	assert.Equal(t, vaa.signingBody(), expected)
}

func TestSigningMsg(t *testing.T) {
	vaa := getVaa()
	expected := common.HexToHash("4fae136bb1fd782fe1b5180ba735cdc83bcece3f9b7fd0e5e35300a61c8acd8f")
	assert.Equal(t, vaa.SigningMsg(), expected)
}

func TestMessageID(t *testing.T) {
	vaa := getVaa()
	expected := "1/0000000000000000000000000000000000000000000000000000000000000004/1"
	assert.Equal(t, vaa.MessageID(), expected)
}

func TestHexDigest(t *testing.T) {
	vaa := getVaa()
	expected := "4fae136bb1fd782fe1b5180ba735cdc83bcece3f9b7fd0e5e35300a61c8acd8f"
	assert.Equal(t, vaa.HexDigest(), expected)
}

func TestMarshal(t *testing.T) {
	expectedBytes := []byte{0x1, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x20, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61}
	vaa := getVaa()
	marshalBytes, err := vaa.Marshal()
	assert.Nil(t, err)
	assert.Equal(t, expectedBytes, marshalBytes)
}

func TestUnmarshal(t *testing.T) {
	vaaBytes := []byte{0x1, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x20, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61}
	vaa1 := getVaa()
	vaa2, err := Unmarshal(vaaBytes)
	assert.Nil(t, err)
	assert.Equal(t, &vaa1, vaa2)
}

func TestUnmarshalTooBig(t *testing.T) {
	vaa := getVaa()

	// Overwrite an oversized payload for the VAA that we cannot unmarshal
	var payload []byte
	for i := 0; i < 2000; i++ {
		payload = append(payload, 'a')
	}
	vaa.Payload = payload

	// Let's marshal the VAA to bytes to unmarshaled
	marshalBytes, err := vaa.Marshal()
	assert.Nil(t, err)

	// Let's now unmarshal the oversized VAA and cause it to panic
	vaa2, err2 := Unmarshal(marshalBytes)
	assert.Nil(t, err2)

	// Marshal the VAA
	marshalBytes2, err3 := vaa2.Marshal()
	assert.Nil(t, err3)

	// Verify that it's truncated at to 1057 (57 byte header + 1000 byte payload)
	assert.Equal(t, marshalBytes[:1057], marshalBytes2)
}

func TestVerifySignatures(t *testing.T) {
	// Generate some random private keys to sign with
	privKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey2, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey3, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey4, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	// Give a fixed order of trusted addresses
	addrs := []common.Address{}
	addrs = append(addrs, crypto.PubkeyToAddress(privKey1.PublicKey))
	addrs = append(addrs, crypto.PubkeyToAddress(privKey2.PublicKey))
	addrs = append(addrs, crypto.PubkeyToAddress(privKey3.PublicKey))

	type test struct {
		label      string
		keyOrder   []*ecdsa.PrivateKey
		addrs      []common.Address
		indexOrder []uint8
		result     bool
	}

	tests := []test{
		{label: "NoSignerZero",
			keyOrder:   []*ecdsa.PrivateKey{},
			addrs:      addrs,
			indexOrder: []uint8{0},
			result:     true},
		{label: "NoSignerOne",
			keyOrder:   []*ecdsa.PrivateKey{},
			addrs:      addrs,
			indexOrder: []uint8{1},
			result:     true},
		{label: "SingleZero",
			keyOrder:   []*ecdsa.PrivateKey{privKey1},
			addrs:      addrs,
			indexOrder: []uint8{0},
			result:     true},
		{label: "RogueSingleOne",
			keyOrder:   []*ecdsa.PrivateKey{privKey4},
			addrs:      addrs,
			indexOrder: []uint8{0},
			result:     false},
		{label: "RogueSingleZero",
			keyOrder:   []*ecdsa.PrivateKey{privKey4},
			addrs:      addrs,
			indexOrder: []uint8{0},
			result:     false},
		{label: "SingleOne",
			keyOrder:   []*ecdsa.PrivateKey{privKey1},
			addrs:      addrs,
			indexOrder: []uint8{0},
			result:     true},
		{label: "MultiUniqSignerMonotonicIndex",
			keyOrder:   []*ecdsa.PrivateKey{privKey1, privKey2, privKey3},
			addrs:      addrs,
			indexOrder: []uint8{0, 1, 2},
			result:     true},
		{label: "MultiMisOrderedSignerMonotonicIndex",
			keyOrder:   []*ecdsa.PrivateKey{privKey3, privKey2, privKey1},
			addrs:      addrs,
			indexOrder: []uint8{0, 1, 2}, result: false},
		{label: "MultiUniqSignerNonMonotonic",
			keyOrder:   []*ecdsa.PrivateKey{privKey1, privKey2, privKey3},
			addrs:      addrs,
			indexOrder: []uint8{0, 2, 1},
			result:     false},
		{label: "MultiUniqSignerFullSameIndex0",
			keyOrder:   []*ecdsa.PrivateKey{privKey1, privKey2, privKey3},
			addrs:      addrs,
			indexOrder: []uint8{0, 0, 0},
			result:     false},
		{label: "MultiUniqSignerFullSameIndex1",
			keyOrder:   []*ecdsa.PrivateKey{privKey1, privKey2, privKey3},
			addrs:      addrs,
			indexOrder: []uint8{0, 0, 0},
			result:     false},
		{label: "MultiUniqSignerPartialSameIndex",
			keyOrder:   []*ecdsa.PrivateKey{privKey1, privKey2, privKey3},
			addrs:      addrs,
			indexOrder: []uint8{0, 1, 1},
			result:     false},
		{label: "MultiSameSignerPartialSameIndex",
			keyOrder:   []*ecdsa.PrivateKey{privKey1, privKey2, privKey2},
			addrs:      addrs,
			indexOrder: []uint8{0, 1, 1},
			result:     false},
		{label: "MultiSameSignerNonMonotonic",
			keyOrder:   []*ecdsa.PrivateKey{privKey1, privKey2, privKey2},
			addrs:      addrs,
			indexOrder: []uint8{0, 2, 1},
			result:     false},
		{label: "MultiSameSignerFullSameIndex",
			keyOrder:   []*ecdsa.PrivateKey{privKey1, privKey1, privKey1},
			addrs:      addrs,
			indexOrder: []uint8{0, 0, 0},
			result:     false},
		{label: "MultiSameSignerMonotonic",
			keyOrder:   []*ecdsa.PrivateKey{privKey1, privKey1, privKey1},
			addrs:      addrs,
			indexOrder: []uint8{0, 1, 2},
			result:     false},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			vaa := getVaa()

			for i, key := range tc.keyOrder {
				vaa.AddSignature(key, tc.indexOrder[i])
			}

			assert.Equal(t, tc.result, vaa.VerifySignatures(tc.addrs))
		})
	}
}

func TestVerifySignaturesFuzz(t *testing.T) {
	// Generate some random trusted private keys to sign with
	privKey1, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey2, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey3, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	// Generate some random untrusted private keys to sign with
	privKey4, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey5, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)
	privKey6, _ := ecdsa.GenerateKey(crypto.S256(), rand.Reader)

	// Give a fixed order of trusted addresses (we intentionally omit privKey4, privKey5, privKey6)
	addrs := []common.Address{}
	addrs = append(addrs, crypto.PubkeyToAddress(privKey1.PublicKey))
	addrs = append(addrs, crypto.PubkeyToAddress(privKey2.PublicKey))
	addrs = append(addrs, crypto.PubkeyToAddress(privKey3.PublicKey))

	// key space for fuzz tests
	keys := []*ecdsa.PrivateKey{}
	keys = append(keys, privKey1)
	keys = append(keys, privKey2)
	keys = append(keys, privKey3)
	keys = append(keys, privKey4)
	keys = append(keys, privKey5)
	keys = append(keys, privKey6)

	// index space for fuzz tests
	indexes := []uint8{0, 1, 2, 3, 4, 5}

	type test struct {
		label      string
		keyOrder   []*ecdsa.PrivateKey
		addrs      []common.Address
		indexOrder []uint8
		result     bool
	}

	type allow struct {
		keyPair   []*ecdsa.PrivateKey
		indexPair []uint8
	}

	// Known good cases where we should have a verified result for
	allows := []allow{
		{keyPair: []*ecdsa.PrivateKey{}, indexPair: []uint8{}},
		{keyPair: []*ecdsa.PrivateKey{privKey1}, indexPair: []uint8{0}},
		{keyPair: []*ecdsa.PrivateKey{privKey2}, indexPair: []uint8{1}},
		{keyPair: []*ecdsa.PrivateKey{privKey3}, indexPair: []uint8{2}},
		{keyPair: []*ecdsa.PrivateKey{privKey1, privKey2}, indexPair: []uint8{0, 1}},
		{keyPair: []*ecdsa.PrivateKey{privKey1, privKey3}, indexPair: []uint8{0, 2}},
		{keyPair: []*ecdsa.PrivateKey{privKey2, privKey3}, indexPair: []uint8{1, 2}},
		{keyPair: []*ecdsa.PrivateKey{privKey1, privKey2, privKey3}, indexPair: []uint8{0, 1, 2}},
	}

	tests := []test{}
	keyPairs := [][]*ecdsa.PrivateKey{}
	indexPairs := [][]uint8{}

	// Build empty keyPair
	keyPairs = append(keyPairs, []*ecdsa.PrivateKey{})

	// Build single keyPairs
	for _, key := range keys {
		keyPairs = append(keyPairs, []*ecdsa.PrivateKey{key})
	}

	// Build double keyPairs
	for _, key_i := range keys {
		for _, key_j := range keys {
			keyPairs = append(keyPairs, []*ecdsa.PrivateKey{key_i, key_j})
		}
	}

	// Build triple keyPairs
	for _, key_i := range keys {
		for _, key_j := range keys {
			for _, key_k := range keys {
				keyPairs = append(keyPairs, []*ecdsa.PrivateKey{key_i, key_j, key_k})
			}
		}
	}

	// Build empty indexPairs
	indexPairs = append(indexPairs, []uint8{})

	// Build single indexPairs
	for _, ind := range indexes {
		indexPairs = append(indexPairs, []uint8{ind})
	}

	// Build double indexPairs
	for _, ind_i := range indexes {
		for _, ind_j := range indexes {
			indexPairs = append(indexPairs, []uint8{ind_i, ind_j})
		}
	}

	// Build triple keyPairs
	for _, ind_i := range indexes {
		for _, ind_j := range indexes {
			for _, ind_k := range indexes {
				indexPairs = append(indexPairs, []uint8{ind_i, ind_j, ind_k})
			}
		}
	}

	// Build out the fuzzTest cases
	for _, keyPair := range keyPairs {
		for _, indexPair := range indexPairs {
			if len(keyPair) == len(indexPair) {
				result := false

				for _, allow := range allows {
					if reflect.DeepEqual(allow.indexPair, indexPair) && reflect.DeepEqual(allow.keyPair, keyPair) {
						result = true
						break
					}
				}

				test := test{label: "A", keyOrder: keyPair, addrs: addrs, indexOrder: indexPair, result: result}
				tests = append(tests, test)
			}
		}
	}

	// Run the fuzzTest cases
	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			vaa := getVaa()

			for i, key := range tc.keyOrder {
				vaa.AddSignature(key, tc.indexOrder[i])
			}

			/* Fuzz Debugging
			 * Tell us what keys and indexes were used (for debug when/if we have a failure case)
			 */
			if vaa.VerifySignatures(tc.addrs) != tc.result {
				if len(tc.keyOrder) == 0 {
					t.Logf("Key Order %v\n", tc.keyOrder)
				} else {
					keyIndex := []uint8{}
					for i, key_i := range keys {
						for _, key_k := range tc.keyOrder {
							if key_i == key_k {
								keyIndex = append(keyIndex, uint8(i))
							}
						}
					}
					t.Logf("Key Order %v\n", keyIndex)
				}
				t.Logf("Index Order %v\n", tc.indexOrder)

			}

			assert.Equal(t, tc.result, vaa.VerifySignatures(tc.addrs))
		})
	}
}

func TestStringToAddress(t *testing.T) {

	type Test struct {
		label     string
		rawAddr   string
		addr      Address
		errString string
	}

	tests := []Test{
		{label: "simple",
			rawAddr:   "0000000000000000000000000000000000000000000000000000000000000004",
			addr:      Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4},
			errString: ""},
		{label: "zero-padding",
			rawAddr:   "04",
			addr:      Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4},
			errString: ""},
		{label: "trim-0x", rawAddr: "0x04",
			addr:      Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4},
			errString: ""},
		{label: "20byte eth-style address", rawAddr: "0x0290FB167208Af455bB137780163b7B7a9a10C16",
			addr:      Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x90, 0xfb, 0x16, 0x72, 0x8, 0xaf, 0x45, 0x5b, 0xb1, 0x37, 0x78, 0x1, 0x63, 0xb7, 0xb7, 0xa9, 0xa1, 0xc, 0x16},
			errString: ""},
		{label: "too long",
			rawAddr:   "0x0000000000000000000000000000000000000000000000000000000000000000000004",
			errString: "value must be no more than 32 bytes"},
		{label: "too short",
			rawAddr:   "4",
			errString: "value must be at least 1 byte"},
		{label: "empty string",
			rawAddr:   "",
			errString: "value must be at least 1 byte"},
	}

	for _, tc := range tests {
		t.Run(string(tc.label), func(t *testing.T) {
			addr, err := StringToAddress(tc.rawAddr)
			if len(tc.errString) == 0 {
				assert.NoError(t, err)
				assert.Equal(t, tc.addr, addr)
			} else {
				assert.Equal(t, tc.errString, err.Error())
			}
		})
	}
}

func TestBytesToAddress(t *testing.T) {
	addrStr := "0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"
	expectedAddr, err := StringToAddress(addrStr)
	assert.NoError(t, err)

	addrBytes, err := hex.DecodeString(addrStr)
	assert.NoError(t, err)

	addr, err := BytesToAddress(addrBytes)
	assert.NoError(t, err)
	assert.Equal(t, expectedAddr, addr)

	// More than 32 bytes should generate an error.
	tooLongAddrBytes, err := hex.DecodeString("0000" + addrStr)
	assert.NoError(t, err)

	_, err = BytesToAddress(tooLongAddrBytes)
	assert.NotNil(t, err)
	assert.Equal(t, "value must be no more than 32 bytes", err.Error())

	// Less than 32 bytes should get left padded with zeros.
	shortAddr, err := BytesToAddress(addrBytes[4:])
	assert.NoError(t, err)
	assert.Equal(t, expectedAddr, shortAddr)
}

func TestDecodeTransferPayloadHdr(t *testing.T) {
	type Test struct {
		label          string
		vaa            string
		payloadType    uint8
		emitterChainId ChainID
		emitterAddr    string
		originChain    ChainID
		originAddress  string
		targetChain    ChainID
		targetAddress  string
		amount         int64
		errString      string
	}

	tests := []Test{
		{label: "valid vaa",
			vaa:            "01000000000100e424aef95296cb0f2185f351086c7c0b9cd031d1288f0537d04ab20d5fc709416224b2bd9a8010a81988aa9cb38b378eb915f88b67e32a765928d948dc02077e00000102584a8d000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000f0f01000000000000000000000000000000000000000000000000000000002b369f40000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000221c175fcd8e3a19fe2e0deae96534f0f4e6a896f4df0e3ec5345fe27ac3f63f000010000000000000000000000000000000000000000000000000000000000000000",
			payloadType:    1,
			emitterChainId: ChainIDEthereum,
			emitterAddr:    "0000000000000000000000000290FB167208Af455bB137780163b7B7a9a10C16",
			originChain:    ChainIDEthereum,
			originAddress:  "000000000000000000000000DDb64fE46a91D46ee29420539FC25FD07c5FEa3E",
			targetChain:    ChainIDSolana,
			targetAddress:  "21c175fcd8e3a19fe2e0deae96534f0f4e6a896f4df0e3ec5345fe27ac3f63f0",
			amount:         725000000,
			errString:      "",
		},
		{label: "unsupported payload type",
			vaa:       "01000000000100e424aef95296cb0f2185f351086c7c0b9cd031d1288f0537d04ab20d5fc709416224b2bd9a8010a81988aa9cb38b378eb915f88b67e32a765928d948dc02077e00000102584a8d000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000f0f02000000000000000000000000000000000000000000000000000000002b369f40000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000221c175fcd8e3a19fe2e0deae96534f0f4e6a896f4df0e3ec5345fe27ac3f63f000010000000000000000000000000000000000000000000000000000000000000000",
			errString: "unsupported payload type",
		},
		{label: "buffer too short",
			vaa:       "01000000000100e424aef95296cb0f2185f351086c7c0b9cd031d1288f0537d04ab20d5fc709416224b2bd9a8010a81988aa9cb38b378eb915f88b67e32a765928d948dc02077e00000102584a8d000000020000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16000000000000000f0f01",
			errString: "buffer too short",
		},
		{label: "empty string",
			vaa:       "",
			errString: "VAA is too short",
		},
	}

	for _, tc := range tests {
		t.Run(string(tc.label), func(t *testing.T) {
			data, err := hex.DecodeString(tc.vaa)
			assert.NoError(t, err)

			vaa, err := Unmarshal(data)
			if err != nil {
				assert.Equal(t, tc.errString, err.Error())
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, vaa)

				if len(tc.errString) == 0 {
					expectedEmitterAddr, err := StringToAddress(tc.emitterAddr)
					assert.NoError(t, err)

					expectedOriginAddress, err := StringToAddress(tc.originAddress)
					assert.NoError(t, err)

					expectedTargetAddress, err := StringToAddress(tc.targetAddress)
					assert.NoError(t, err)

					expectedAmount := big.NewInt(tc.amount)

					assert.Equal(t, tc.emitterChainId, vaa.EmitterChain)
					assert.Equal(t, expectedEmitterAddr, vaa.EmitterAddress)
					assert.Equal(t, 133, len(vaa.Payload))

					payload, err := DecodeTransferPayloadHdr(vaa.Payload)
					assert.NoError(t, err)
					assert.Equal(t, tc.payloadType, payload.Type)
					assert.Equal(t, tc.originChain, payload.OriginChain)
					assert.Equal(t, expectedOriginAddress, payload.OriginAddress)
					assert.Equal(t, tc.targetChain, payload.TargetChain)
					assert.Equal(t, expectedTargetAddress, payload.TargetAddress)
					assert.Equal(t, expectedAmount.Cmp(payload.Amount), 0)
				} else {
					_, err = DecodeTransferPayloadHdr(vaa.Payload)
					assert.NotNil(t, err)
					assert.Equal(t, tc.errString, err.Error())
				}
			}

		})
	}
}

func TestIsTransfer(t *testing.T) {
	type Test struct {
		label   string
		payload []byte
		result  bool
	}

	tests := []Test{
		{label: "empty", payload: []byte{}, result: false},
		{label: "non-valid payload", payload: []byte{0x0}, result: false},
		{label: "payload 1", payload: []byte{0x1}, result: true},
		{label: "payload 2", payload: []byte{0x2}, result: false},
		{label: "payload 3", payload: []byte{0x3}, result: true},
		{label: "payload 4", payload: []byte{0x4}, result: false},
	}

	for _, tc := range tests {
		t.Run(string(tc.label), func(t *testing.T) {
			assert.Equal(t, tc.result, IsTransfer(tc.payload))
		})
	}
}
