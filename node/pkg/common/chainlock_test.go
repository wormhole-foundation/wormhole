package common

import (
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/vaa"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func encodePayloadBytes(payload *vaa.TransferPayloadHdr) []byte {
	bytes := make([]byte, 101)
	bytes[0] = payload.Type

	amtBytes := payload.Amount.Bytes()
	if len(amtBytes) > 32 {
		panic("amount will not fit in 32 bytes!")
	}
	copy(bytes[33-len(amtBytes):33], amtBytes)

	copy(bytes[33:65], payload.OriginAddress.Bytes())
	binary.BigEndian.PutUint16(bytes[65:67], uint16(payload.OriginChain))
	copy(bytes[67:99], payload.TargetAddress.Bytes())
	binary.BigEndian.PutUint16(bytes[99:101], uint16(payload.TargetChain))
	return bytes
}

func TestSerializeAndDeserializeOfMessagePublication(t *testing.T) {
	originAddress, err := vaa.StringToAddress("0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E") //nolint:gosec
	require.NoError(t, err)

	targetAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	payload1 := &vaa.TransferPayloadHdr{
		Type:          0x01,
		Amount:        big.NewInt(27000000000),
		OriginAddress: originAddress,
		OriginChain:   vaa.ChainIDEthereum,
		TargetAddress: targetAddress,
		TargetChain:   vaa.ChainIDPolygon,
	}

	payloadBytes1 := encodePayloadBytes(payload1)

	msg1 := &MessagePublication{
		TxHash:           eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes1,
		ConsistencyLevel: 32,
	}

	bytes, err := msg1.Marshal()
	require.NoError(t, err)

	msg2, err := UnmarshalMessagePublication(bytes)
	require.NoError(t, err)
	assert.Equal(t, msg1, msg2)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	require.NoError(t, err)

	assert.Equal(t, payload1, payload2)
}

func TestMessageIDString(t *testing.T) {
	addr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	type test struct {
		label  string
		input  MessagePublication
		output string
	}

	tests := []test{
		{label: "simple",
			input:  MessagePublication{Sequence: 1, EmitterChain: vaa.ChainIDEthereum, EmitterAddress: addr},
			output: "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/1"},
		{label: "missing sequence",
			input:  MessagePublication{EmitterChain: vaa.ChainIDEthereum, EmitterAddress: addr},
			output: "2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0"},
		{label: "missing chain id",
			input:  MessagePublication{Sequence: 1, EmitterAddress: addr},
			output: "0/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/1"},
		{label: "missing emitter address",
			input:  MessagePublication{Sequence: 1, EmitterChain: vaa.ChainIDEthereum},
			output: "2/0000000000000000000000000000000000000000000000000000000000000000/1"},
		{label: "empty message",
			input:  MessagePublication{},
			output: "0/0000000000000000000000000000000000000000000000000000000000000000/0"},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			assert.Equal(t, tc.output, tc.input.MessageIDString())
		})
	}
}

func TestMessageID(t *testing.T) {
	addr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	type test struct {
		label  string
		input  MessagePublication
		output []byte
	}

	tests := []test{
		{label: "simple",
			input:  MessagePublication{Sequence: 1, EmitterChain: vaa.ChainIDEthereum, EmitterAddress: addr},
			output: []byte("2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/1")},
		{label: "missing sequence",
			input:  MessagePublication{EmitterChain: vaa.ChainIDEthereum, EmitterAddress: addr},
			output: []byte("2/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/0")},
		{label: "missing chain id",
			input:  MessagePublication{Sequence: 1, EmitterAddress: addr},
			output: []byte("0/0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16/1")},
		{label: "missing emitter address",
			input:  MessagePublication{Sequence: 1, EmitterChain: vaa.ChainIDEthereum},
			output: []byte("2/0000000000000000000000000000000000000000000000000000000000000000/1")},
		{label: "empty message",
			input:  MessagePublication{},
			output: []byte("0/0000000000000000000000000000000000000000000000000000000000000000/0")},
	}

	for _, tc := range tests {
		t.Run(tc.label, func(t *testing.T) {
			assert.Equal(t, tc.output, tc.input.MessageID())
		})
	}
}
