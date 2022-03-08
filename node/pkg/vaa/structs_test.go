package vaa

import (
	// "crypto/ecdsa"
	// "crypto/elliptic"
	// "crypto/rand"
	"encoding/hex"
	"github.com/stretchr/testify/assert"
	"testing"
	// "time"
)

func TestChainIDFromString(t *testing.T) {
	type test struct {
		i string
		o ChainID
	}

	// Positive Test Cases
	p_tests := []test{
		{i: "solana", o: ChainIDSolana},
		{i: "ethereum", o: ChainIDEthereum},
		{i: "terra", o: ChainIDTerra},
		{i: "bsc", o: ChainIDBSC},
		{i: "polygon", o: ChainIDPolygon},
		{i: "avalanche", o: ChainIDAvalanche},
		{i: "oasis", o: ChainIDOasis},
		{i: "fantom", o: ChainIDFantom},
		{i: "algorand", o: ChainIDAlgorand},
		{i: "ethereum-ropsten", o: ChainIDEthereumRopsten},
		{i: "Solana", o: ChainIDSolana},
		{i: "Ethereum", o: ChainIDEthereum},
		{i: "Terra", o: ChainIDTerra},
		{i: "Bsc", o: ChainIDBSC},
		{i: "Polygon", o: ChainIDPolygon},
		{i: "Avalanche", o: ChainIDAvalanche},
		{i: "Oasis", o: ChainIDOasis},
		{i: "Fantom", o: ChainIDFantom},
		{i: "Algorand", o: ChainIDAlgorand},
		{i: "Karura", o: ChainIDKarura},
		{i: "Acala", o: ChainIDAcala},
	}

	// Negative Test Cases
	n_tests := []test{
		{i: "Unknown", o: ChainIDUnset},
	}

	for _, tc := range p_tests {
		chainId, err := ChainIDFromString(tc.i)
		assert.Equal(t, tc.o, chainId)
		assert.Nil(t, err)
	}

	for _, tc := range n_tests {
		chainId, err := ChainIDFromString(tc.i)
		assert.Equal(t, tc.o, chainId)
		assert.NotNil(t, err)
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
	expected := "223030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303430303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303030303034303022"
	assert.Equal(t, hex.EncodeToString(marshalJsonSigData), expected)
	assert.Nil(t, err)
}

func TestSignature_DataString(t *testing.T) {
	sigData := SignatureData{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 4, 0}
	expected := "0000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000400"
	assert.Equal(t, sigData.String(), expected)
}

func TestChainId_String(t *testing.T) {
	type test struct {
		i ChainID
		o string
	}

	tests := []test{
		{i: 0, o: "unset"},
		{i: 1, o: "solana"},
		{i: 2, o: "ethereum"},
		{i: 3, o: "terra"},
		{i: 4, o: "bsc"},
		{i: 5, o: "polygon"},
		{i: 6, o: "avalanche"},
		{i: 7, o: "oasis"},
		{i: 10, o: "fantom"},
		{i: 8, o: "algorand"},
		{i: 10001, o: "ethereum-ropsten"},
		{i: 11, o: "karura"},
		{i: 12, o: "acala"},
	}

	for _, tc := range tests {
		assert.Equal(t, ChainID(tc.i).String(), tc.o)
	}
}
