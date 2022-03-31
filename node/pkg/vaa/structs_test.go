package vaa

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/hex"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func getVAA() *VAA {
	var payload = []byte{97, 97, 97, 97, 97, 97}
	var governanceEmitter = Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}

	vaa := &VAA{
		Version:          uint8(1),
		GuardianSetIndex: uint32(1),
		Signatures:       nil,
		Timestamp:        time.Unix(0, 0),
		Nonce:            uint32(1),
		Sequence:         uint64(1),
		ConsistencyLevel: uint8(32),
		EmitterChain:     ChainIDSolana,
		EmitterAddress:   governanceEmitter,
		Payload:          payload,
	}

	return vaa
}

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
			assert.Nil(t, err)
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
	assert.Nil(t, err)
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
		{input: 10001, output: "ethereum-ropsten"},
	}

	for _, tc := range tests {
		t.Run(tc.output, func(t *testing.T) {
			assert.Equal(t, ChainID(tc.input).String(), tc.output)
		})
	}
}

func TestAddSignature(t *testing.T) {
	vaa := getVAA()

	// Generate a random private key to sign with
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.Nil(t, vaa.Signatures)

	// Add a signature and make sure it's added
	vaa.AddSignature(key, 0)
	assert.Equal(t, len(vaa.Signatures), 1)
}

func TestSerializeBody(t *testing.T) {
	vaa := getVAA()
	expected := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x20, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61}
	assert.Equal(t, vaa.serializeBody(), expected)
}

func TestSigningBody(t *testing.T) {
	vaa := getVAA()
	expected := []byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x20, 0x61, 0x61, 0x61, 0x61, 0x61, 0x61}
	assert.Equal(t, vaa.signingBody(), expected)
}

func TestSigningMsg(t *testing.T) {
	vaa := getVAA()
	expected := common.HexToHash("4fae136bb1fd782fe1b5180ba735cdc83bcece3f9b7fd0e5e35300a61c8acd8f")
	assert.Equal(t, vaa.SigningMsg(), expected)
}

func TestMessageID(t *testing.T) {
	vaa := getVAA()
	expected := "1/0000000000000000000000000000000000000000000000000000000000000004/1"
	assert.Equal(t, vaa.MessageID(), expected)
}

func TestHexDigest(t *testing.T) {
	vaa := getVAA()
	expected := "4fae136bb1fd782fe1b5180ba735cdc83bcece3f9b7fd0e5e35300a61c8acd8f"
	assert.Equal(t, vaa.HexDigest(), expected)
}

func TestVerifySignatures(t *testing.T) {
	vaa := getVAA()

	// Generate a random private key to sign with
	privKey, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.Nil(t, vaa.Signatures)

	// Add a signature and make sure it's added
	vaa.AddSignature(privKey, 0)
	assert.Equal(t, len(vaa.Signatures), 1)

	// Generate a public key to compare to from our private key
	h := vaa.SigningMsg()
	pubKey, err := crypto.Ecrecover(h.Bytes(), vaa.Signatures[0].Signature[:])
	assert.Nil(t, err)
	assert.NotNil(t, pubKey)

	// Translate that public key back to an address
	addr := common.BytesToAddress(crypto.Keccak256(pubKey[1:])[12:])

	// Make sure that it verifies
	assert.True(t, vaa.VerifySignatures([]common.Address{addr}))
}

func TestStringToAddress(t *testing.T) {
	expected := Address{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4}
	addr, err := StringToAddress("0000000000000000000000000000000000000000000000000000000000000004")
	assert.Nil(t, err)
	assert.Equal(t, expected, addr)
}
