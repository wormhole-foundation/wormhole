package vaa

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
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
		{input: "near", output: ChainIDNear},
		{input: "moonbeam", output: ChainIDMoonbeam},
		{input: "terra2", output: ChainIDTerra2},
		{input: "injective", output: ChainIDInjective},
		{input: "osmosis", output: ChainIDOsmosis},
		{input: "sui", output: ChainIDSui},
		{input: "aptos", output: ChainIDAptos},
		{input: "arbitrum", output: ChainIDArbitrum},
		{input: "optimism", output: ChainIDOptimism},
		{input: "gnosis", output: ChainIDGnosis},
		{input: "pythnet", output: ChainIDPythNet},
		{input: "xpla", output: ChainIDXpla},
		{input: "btc", output: ChainIDBtc},
		{input: "base", output: ChainIDBase},
		{input: "filecoin", output: ChainIDFileCoin},
		{input: "sei", output: ChainIDSei},
		{input: "rootstock", output: ChainIDRootstock},
		{input: "scroll", output: ChainIDScroll},
		{input: "mantle", output: ChainIDMantle},
		{input: "blast", output: ChainIDBlast},
		{input: "xlayer", output: ChainIDXLayer},
		{input: "linea", output: ChainIDLinea},
		{input: "berachain", output: ChainIDBerachain},
		{input: "eclipse", output: ChainIDEclipse},
		{input: "bob", output: ChainIDBOB},
		{input: "seievm", output: ChainIDSeiEVM},
		{input: "snaxchain", output: ChainIDSnaxchain},
		{input: "unichain", output: ChainIDUnichain},
		{input: "worldchain", output: ChainIDWorldchain},
		{input: "ink", output: ChainIDInk},
		{input: "hyperevm", output: ChainIDHyperEVM},
		{input: "monad", output: ChainIDMonad},
		{input: "movement", output: ChainIDMovement},
		{input: "mezo", output: ChainIDMezo},
		{input: "fogo", output: ChainIDFogo},
		{input: "wormchain", output: ChainIDWormchain},
		{input: "cosmoshub", output: ChainIDCosmoshub},
		{input: "evmos", output: ChainIDEvmos},
		{input: "kujira", output: ChainIDKujira},
		{input: "neutron", output: ChainIDNeutron},
		{input: "celestia", output: ChainIDCelestia},
		{input: "stargaze", output: ChainIDStargaze},
		{input: "seda", output: ChainIDSeda},
		{input: "dymension", output: ChainIDDymension},
		{input: "provenance", output: ChainIDProvenance},
		{input: "noble", output: ChainIDNoble},
		{input: "sepolia", output: ChainIDSepolia},
		{input: "arbitrum_sepolia", output: ChainIDArbitrumSepolia},
		{input: "base_sepolia", output: ChainIDBaseSepolia},
		{input: "optimism_sepolia", output: ChainIDOptimismSepolia},
		{input: "holesky", output: ChainIDHolesky},
		{input: "polygon_sepolia", output: ChainIDPolygonSepolia},

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
		{input: "Near", output: ChainIDNear},
		{input: "Moonbeam", output: ChainIDMoonbeam},
		{input: "Terra2", output: ChainIDTerra2},
		{input: "Injective", output: ChainIDInjective},
		{input: "Osmosis", output: ChainIDOsmosis},
		{input: "Sui", output: ChainIDSui},
		{input: "Aptos", output: ChainIDAptos},
		{input: "Arbitrum", output: ChainIDArbitrum},
		{input: "Optimism", output: ChainIDOptimism},
		{input: "Gnosis", output: ChainIDGnosis},
		{input: "Pythnet", output: ChainIDPythNet},
		{input: "XPLA", output: ChainIDXpla},
		{input: "BTC", output: ChainIDBtc},
		{input: "Base", output: ChainIDBase},
		{input: "filecoin", output: ChainIDFileCoin},
		{input: "Sei", output: ChainIDSei},
		{input: "Rootstock", output: ChainIDRootstock},
		{input: "Scroll", output: ChainIDScroll},
		{input: "Mantle", output: ChainIDMantle},
		{input: "Blast", output: ChainIDBlast},
		{input: "XLayer", output: ChainIDXLayer},
		{input: "Linea", output: ChainIDLinea},
		{input: "Berachain", output: ChainIDBerachain},
		{input: "SeiEVM", output: ChainIDSeiEVM},
		{input: "Eclipse", output: ChainIDEclipse},
		{input: "BOB", output: ChainIDBOB},
		{input: "Snaxchain", output: ChainIDSnaxchain},
		{input: "Unichain", output: ChainIDUnichain},
		{input: "Worldchain", output: ChainIDWorldchain},
		{input: "Ink", output: ChainIDInk},
		{input: "HyperEVM", output: ChainIDHyperEVM},
		{input: "Monad", output: ChainIDMonad},
		{input: "Movement", output: ChainIDMovement},
		{input: "Mezo", output: ChainIDMezo},
		{input: "Fogo", output: ChainIDFogo},
		{input: "Wormchain", output: ChainIDWormchain},
		{input: "Cosmoshub", output: ChainIDCosmoshub},
		{input: "Evmos", output: ChainIDEvmos},
		{input: "Kujira", output: ChainIDKujira},
		{input: "Neutron", output: ChainIDNeutron},
		{input: "Celestia", output: ChainIDCelestia},
		{input: "Stargaze", output: ChainIDStargaze},
		{input: "Seda", output: ChainIDSeda},
		{input: "Dymension", output: ChainIDDymension},
		{input: "Provenance", output: ChainIDProvenance},
		{input: "Noble", output: ChainIDNoble},
		{input: "Sepolia", output: ChainIDSepolia},
		{input: "Arbitrum_Sepolia", output: ChainIDArbitrumSepolia},
		{input: "Base_Sepolia", output: ChainIDBaseSepolia},
		{input: "Optimism_Sepolia", output: ChainIDOptimismSepolia},
		{input: "Holesky", output: ChainIDHolesky},
		{input: "Polygon_Sepolia", output: ChainIDPolygonSepolia},
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

func TestAddress_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		address     Address
		addressJSON string
		err         error
	}{
		{
			name:        "working",
			addressJSON: "0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16",
			address:     Address{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x2, 0x90, 0xfb, 0x16, 0x72, 0x8, 0xaf, 0x45, 0x5b, 0xb1, 0x37, 0x78, 0x1, 0x63, 0xb7, 0xb7, 0xa9, 0xa1, 0xc, 0x16},
			err:         nil,
		},
		{
			name:        "failure",
			addressJSON: "derp",
			address:     Address{},
			err:         hex.InvalidByteError(0x72),
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			var unmarshalAddr Address
			err := unmarshalAddr.UnmarshalJSON([]byte(testCase.addressJSON))
			require.Equal(t, testCase.err, err)
			assert.Equal(t, testCase.address, unmarshalAddr)
		})
	}
}

func TestAddress_Unmarshal(t *testing.T) {
	addr, _ := StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	b, err := json.Marshal(addr)
	require.NoError(t, err)

	var unmarshalAddr Address
	err = json.Unmarshal(b, &unmarshalAddr)
	require.NoError(t, err)

	assert.Equal(t, addr, unmarshalAddr)
}

func TestAddress_UnmarshalEmptyBuffer(t *testing.T) {
	b := []byte{}

	var unmarshalAddr Address
	err := json.Unmarshal(b, &unmarshalAddr)
	require.Error(t, err)
}

func TestAddress_UnmarshalBufferTooShort(t *testing.T) {
	addr, _ := StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")

	b, err := json.Marshal(addr)
	require.NoError(t, err)

	var unmarshalAddr Address

	// Lop off the first byte, and it should fail.
	b1 := b[1:]
	err = json.Unmarshal(b1, &unmarshalAddr)
	assert.Error(t, err)

	// Lop off the last byte, and it should fail.
	b2 := b[0 : len(b)-1]
	err = json.Unmarshal(b2, &unmarshalAddr)
	assert.Error(t, err)
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
		{input: 15, output: "near"},
		{input: 16, output: "moonbeam"},
		// 17 (Neon) is obsolete.
		{input: 18, output: "terra2"},
		{input: 19, output: "injective"},
		{input: 20, output: "osmosis"},
		{input: 21, output: "sui"},
		{input: 22, output: "aptos"},
		{input: 23, output: "arbitrum"},
		{input: 24, output: "optimism"},
		{input: 25, output: "gnosis"},
		{input: 26, output: "pythnet"},
		// NOTE: 27 belongs to a chain that was never deployed.
		{input: 28, output: "xpla"},
		{input: 29, output: "btc"},
		{input: 30, output: "base"},
		{input: 31, output: "filecoin"},
		{input: 32, output: "sei"},
		{input: 33, output: "rootstock"},
		{input: 34, output: "scroll"},
		{input: 35, output: "mantle"},
		{input: 36, output: "blast"},
		{input: 37, output: "xlayer"},
		{input: 38, output: "linea"},
		{input: 39, output: "berachain"},
		{input: 40, output: "seievm"},
		{input: 41, output: "eclipse"},
		{input: 42, output: "bob"},
		{input: 43, output: "snaxchain"},
		{input: 44, output: "unichain"},
		{input: 45, output: "worldchain"},
		{input: 46, output: "ink"},
		{input: 47, output: "hyperevm"},
		{input: 48, output: "monad"},
		{input: 49, output: "movement"},
		{input: 3104, output: "wormchain"},
		{input: 4000, output: "cosmoshub"},
		{input: 4001, output: "evmos"},
		{input: 4002, output: "kujira"},
		{input: 4003, output: "neutron"},
		{input: 4004, output: "celestia"},
		{input: 4005, output: "stargaze"},
		{input: 4006, output: "seda"},
		{input: 4007, output: "dymension"},
		{input: 4008, output: "provenance"},
		{input: 4009, output: "noble"},
		{input: 10002, output: "sepolia"},
		{input: 10003, output: "arbitrum_sepolia"},
		{input: 10004, output: "base_sepolia"},
		{input: 10005, output: "optimism_sepolia"},
		{input: 10006, output: "holesky"},
		{input: 10007, output: "polygon_sepolia"},
		{input: 10000, output: "unknown chain ID: 10000"},
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

func getEmptyPayloadVaa() VAA {
	var payload = []byte{}
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
	assert.Equal(t, vaa.SigningDigest(), expected)
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

func TestUnmarshalNoPayload(t *testing.T) {
	vaaBytes := []byte{0x1, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x20}
	vaa1 := getEmptyPayloadVaa()
	vaa2, err := Unmarshal(vaaBytes)
	assert.Nil(t, err)
	assert.Equal(t, &vaa1, vaa2)
}

func TestUnmarshalBigPayload(t *testing.T) {
	vaa := getVaa()

	// Create a payload of more than 1000 bytes.
	var payload []byte
	for i := 0; i < 2000; i++ {
		ch := i % 255
		payload = append(payload, byte(ch))
	}
	vaa.Payload = payload

	marshalBytes, err := vaa.Marshal()
	require.NoError(t, err)

	vaa2, err := Unmarshal(marshalBytes)
	require.NoError(t, err)

	assert.Equal(t, vaa, *vaa2)
}

func FuzzUnmarshalBigPayload(f *testing.F) {
	f.Add([]byte{})
	f.Add([]byte{0x1, 0x0, 0x0, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x0, 0x1, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x4, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x1, 0x20})
	f.Fuzz(func(t *testing.T, payload []byte) {
		vaa := getVaa()
		vaa.Payload = payload

		// It should always marshal
		marshalBytes, err := vaa.Marshal()
		require.NoError(t, err)

		// It should aways unmarshal
		vaa2, err := Unmarshal(marshalBytes)
		require.NoError(t, err)

		// The payload should not be lossy
		assert.Equal(t, vaa.Payload, payload)
		assert.Equal(t, vaa2.Payload, payload)

		// The marshal and unmarshal should always be the same
		assert.Equal(t, vaa, *vaa2)
	})
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
								keyIndex = append(keyIndex, uint8(i)) // #nosec G115 -- We're using 6 keys in this test case
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

func TestUnmarshalBody(t *testing.T) {
	addr, _ := StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	testPayload := []byte("Hi")
	tests := []struct {
		name        string
		data        []byte
		vaa         *VAA
		err         error
		expectedVAA *VAA
		dataFunc    func() []byte
	}{
		{
			name: "invalid_timestamp",
			dataFunc: func() []byte {
				return []byte("Hi")
			},
			err: fmt.Errorf("failed to read timestamp: %w", errors.New("unexpected EOF")),
		},
		{
			name: "invalid_nonce",
			err:  fmt.Errorf("failed to read nonce: %w", errors.New("EOF")),
			vaa:  &VAA{},
			dataFunc: func() []byte {
				buf := new(bytes.Buffer)
				MustWrite(buf, binary.BigEndian, uint32(time.Now().Unix())) // #nosec G115 -- This conversion is safe until year 2106
				return buf.Bytes()
			},
		},
		{
			name: "invalid_emmitter_chain",
			err:  fmt.Errorf("failed to read emitter chain: %w", errors.New("EOF")),
			vaa:  &VAA{},
			dataFunc: func() []byte {
				buf := new(bytes.Buffer)
				MustWrite(buf, binary.BigEndian, uint32(time.Now().Unix())) // #nosec G115 -- This conversion is safe until year 2106
				MustWrite(buf, binary.BigEndian, uint32(123))
				return buf.Bytes()
			},
		},
		{
			name: "invalid_emmitter_address",
			err:  fmt.Errorf("failed to read emitter address [0]: %w", errors.New("EOF")),
			vaa:  &VAA{},
			dataFunc: func() []byte {
				buf := new(bytes.Buffer)
				MustWrite(buf, binary.BigEndian, uint32(time.Now().Unix())) // #nosec G115 -- This conversion is safe until year 2106
				MustWrite(buf, binary.BigEndian, uint32(123))
				MustWrite(buf, binary.BigEndian, ChainIDPythNet)
				return buf.Bytes()
			},
		},
		{
			name: "invalid_sequence_number",
			err:  fmt.Errorf("failed to read sequence: %w", errors.New("EOF")),
			vaa:  &VAA{},
			dataFunc: func() []byte {
				buf := new(bytes.Buffer)
				MustWrite(buf, binary.BigEndian, uint32(time.Now().Unix())) // #nosec G115 -- This conversion is safe until year 2106
				MustWrite(buf, binary.BigEndian, uint32(123))
				MustWrite(buf, binary.BigEndian, ChainIDBSC)
				buf.Write(addr[:])
				return buf.Bytes()
			},
		},
		{
			name: "invalid_consistency_level",
			err:  fmt.Errorf("failed to read commitment: %w", errors.New("EOF")),
			vaa:  &VAA{},
			dataFunc: func() []byte {
				buf := new(bytes.Buffer)
				MustWrite(buf, binary.BigEndian, uint32(time.Now().Unix())) // #nosec G115 -- This conversion is safe until year 2106
				MustWrite(buf, binary.BigEndian, uint32(123))
				MustWrite(buf, binary.BigEndian, ChainIDBSC)
				buf.Write(addr[:])
				MustWrite(buf, binary.BigEndian, uint64(42))
				return buf.Bytes()
			},
		},
		{
			name: "has_payload",
			err:  nil,
			vaa:  &VAA{},
			expectedVAA: &VAA{
				Nonce:            uint32(123),
				Sequence:         uint64(42),
				ConsistencyLevel: uint8(1),
				EmitterChain:     ChainIDBSC,
				Timestamp:        time.Unix(0, 0),
				EmitterAddress:   addr,
				Payload:          testPayload,
			},
			dataFunc: func() []byte {
				buf := new(bytes.Buffer)
				MustWrite(buf, binary.BigEndian, uint32(0))
				MustWrite(buf, binary.BigEndian, uint32(123))
				MustWrite(buf, binary.BigEndian, ChainIDBSC)
				buf.Write(addr[:])
				MustWrite(buf, binary.BigEndian, uint64(42))
				MustWrite(buf, binary.BigEndian, uint8(1))
				buf.Write(testPayload)
				return buf.Bytes()
			},
		},
		{
			name: "has_empty_payload",
			err:  nil,
			vaa:  &VAA{},
			expectedVAA: &VAA{
				Nonce:            uint32(123),
				Sequence:         uint64(42),
				ConsistencyLevel: uint8(1),
				EmitterChain:     ChainIDBSC,
				Timestamp:        time.Unix(0, 0),
				EmitterAddress:   addr,
				Payload:          []byte{},
			},
			dataFunc: func() []byte {
				buf := new(bytes.Buffer)
				MustWrite(buf, binary.BigEndian, uint32(0))
				MustWrite(buf, binary.BigEndian, uint32(123))
				MustWrite(buf, binary.BigEndian, ChainIDBSC)
				buf.Write(addr[:])
				MustWrite(buf, binary.BigEndian, uint64(42))
				MustWrite(buf, binary.BigEndian, uint8(1))
				buf.Write([]byte{})
				return buf.Bytes()
			},
		},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			testBytes := testCase.dataFunc()
			body, err := UnmarshalBody(testCase.data, bytes.NewReader(testBytes), testCase.vaa)
			require.Equal(t, testCase.err, err)
			if err == nil {
				assert.Equal(t, testCase.expectedVAA, body)
			}
		})
	}
}

func TestChainIDFromNumber(t *testing.T) {
	// Define test case struct that works with any Number type
	type testCase[N number] struct {
		name      string
		input     N
		expected  ChainID
		wantErr   bool
		errMsg    string
		wantKnown bool
	}
	// Using the int64 type here because it can be representative of the error conditions (overflow, negative)
	// NOTE: more test cases could be added with different concrete types.
	tests := []testCase[int64]{
		{
			name:      "valid",
			input:     int64(1),
			expected:  ChainIDSolana,
			wantErr:   false,
			wantKnown: true,
		},
		{
			name:      "valid but unknown",
			input:     int64(math.MaxUint16),
			expected:  ChainID(math.MaxUint16),
			wantErr:   false,
			wantKnown: false,
		},
		{
			name:      "overflow",
			input:     math.MaxUint16 + 1,
			expected:  ChainIDUnset,
			wantErr:   true,
			wantKnown: false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			got, err := ChainIDFromNumber(testCase.input)
			require.Equal(t, testCase.expected, got)
			if testCase.wantErr {
				require.ErrorContains(t, err, testCase.errMsg)
				require.Equal(t, ChainIDUnset, got)
			}

			got, err = KnownChainIDFromNumber(testCase.input)
			if testCase.wantKnown {
				require.NoError(t, err)
				require.Equal(t, testCase.expected, got)
			} else {
				require.Error(t, err)
				require.Equal(t, ChainIDUnset, got)
			}
		})
	}
}

func TestStringToKnownChainID(t *testing.T) {

	happy := []struct {
		name     string
		input    string
		expected ChainID
	}{
		{
			name:     "simple int 1",
			input:    "1",
			expected: ChainIDSolana,
		},
		{
			name:     "simple int 2",
			input:    "3104",
			expected: ChainIDWormchain,
		},
		{
			name:     "chain name 1",
			input:    "solana",
			expected: ChainIDSolana,
		},
	}
	for _, tt := range happy {
		// Avoid "loop variable capture".
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := StringToKnownChainID(tt.input)
			require.Equal(t, tt.expected, actual)
			require.NoError(t, err)
		})
	}

	// Check error cases
	sad := []struct {
		name  string
		input string
	}{
		{
			name:  "zero is not a valid ChainID",
			input: "0",
		},
		{
			name:  "negative value",
			input: "-1",
		},
		{
			name:  "NaN",
			input: "garbage",
		},
		{
			name:  "overflow",
			input: "65536",
		},
		{
			name:  "not a real chain",
			input: "12345",
		},
		{
			name:  "empty string",
			input: "",
		},
		{
			name:  "no hex inputs",
			input: "0x10",
		},
	}
	for _, tt := range sad {
		// Avoid "loop variable capture".
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			actual, err := StringToKnownChainID(tt.input)
			require.Equal(t, ChainIDUnset, actual)
			require.Error(t, err)
		})
	}
}
