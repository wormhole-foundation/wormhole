package common

import (
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
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
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes1,
		ConsistencyLevel: 32,
		// NOTE: these fields are not marshalled or unmarshalled. They are set to non-default values
		// here to prove that they will be unmarshalled into their defaults.
		Unreliable:        true,
		verificationState: Anomalous,
	}

	bytes, err := msg1.Marshal()
	require.NoError(t, err)

	msg2, err := UnmarshalMessagePublication(bytes)
	require.NoError(t, err)

	require.Equal(t, msg1.TxID, msg2.TxID)
	require.Equal(t, msg1.Timestamp, msg2.Timestamp)
	require.Equal(t, msg1.Nonce, msg2.Nonce)
	require.Equal(t, msg1.Sequence, msg2.Sequence)
	require.Equal(t, msg1.EmitterChain, msg2.EmitterChain)
	require.Equal(t, msg1.EmitterAddress, msg2.EmitterAddress)
	require.Equal(t, msg1.ConsistencyLevel, msg2.ConsistencyLevel)
	// These fields are not currently marshalled or unmarshalled. Ensure that the unmarshalled values are equal
	// to the defaults for the types, even if the original struct instance had non-default values.
	require.Equal(t, NotVerified, msg2.verificationState)
	require.Equal(t, false, msg2.Unreliable)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	require.NoError(t, err)

	assert.Equal(t, payload1, payload2)
}

func TestSerializeAndDeserializeOfMessagePublicationWithEmptyTxID(t *testing.T) {
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
		TxID:             []byte{},
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

func TestSerializeAndDeserializeOfMessagePublicationWithArbitraryTxID(t *testing.T) {
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
		TxID:             []byte("This is some arbitrary string with just some random junk in it. This is to prove that the TxID does not have to be a ethCommon.Hash"),
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

func TestTxIDStringTooLongShouldFail(t *testing.T) {
	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	// This is limited to 255. Make it 256 and the marshal should fail.
	txID := []byte("0123456789012345678901234567890123456789012345678901234567890123012345678901234567890123456789012345678901234567890123456789012301234567890123456789012345678901234567890123456789012345678901230123456789012345678901234567890123456789012345678901234567890123")

	msg := &MessagePublication{
		TxID:             txID,
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          []byte("Hello, World!"),
		ConsistencyLevel: 32,
	}

	_, err = msg.Marshal()
	assert.ErrorContains(t, err, "TxID too long")
}

func TestSerializeAndDeserializeOfMessagePublicationWithBigPayload(t *testing.T) {
	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	// Create a payload of more than 1000 bytes.
	var payload1 []byte
	for i := 0; i < 2000; i++ {
		ch := i % 255
		payload1 = append(payload1, byte(ch))
	}

	msg1 := &MessagePublication{
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payload1,
		ConsistencyLevel: 32,
	}

	bytes, err := msg1.Marshal()
	require.NoError(t, err)

	msg2, err := UnmarshalMessagePublication(bytes)
	require.NoError(t, err)

	assert.Equal(t, msg1, msg2)
}

func TestMarshalUnmarshalJSONOfMessagePublication(t *testing.T) {
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
		TxID:             eth_common.HexToHash("0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063").Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes1,
		ConsistencyLevel: 32,
	}

	bytes, err := msg1.MarshalJSON()
	require.NoError(t, err)

	var msg2 MessagePublication
	err = msg2.UnmarshalJSON(bytes)
	require.NoError(t, err)
	assert.Equal(t, *msg1, msg2)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	require.NoError(t, err)

	assert.Equal(t, *payload1, *payload2)
}

func TestMarshalUnmarshalJSONOfMessagePublicationWithArbitraryTxID(t *testing.T) {
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
		TxID:             []byte("This is some arbitrary string with just some random junk in it. This is to prove that the TxID does not have to be a ethCommon.Hash"),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          payloadBytes1,
		ConsistencyLevel: 32,
	}

	bytes, err := msg1.MarshalJSON()
	require.NoError(t, err)

	var msg2 MessagePublication
	err = msg2.UnmarshalJSON(bytes)
	require.NoError(t, err)
	assert.Equal(t, *msg1, msg2)

	payload2, err := vaa.DecodeTransferPayloadHdr(msg2.Payload)
	require.NoError(t, err)

	assert.Equal(t, *payload1, *payload2)
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

func TestTxIDStringMatchesHashToString(t *testing.T) {
	tokenBridgeAddress, err := vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
	require.NoError(t, err)

	expectedHashID := "0x06f541f5ecfc43407c31587aa6ac3a689e8960f36dc23c332db5510dfc6a4063"

	msg := &MessagePublication{
		TxID:             eth_common.HexToHash(expectedHashID).Bytes(),
		Timestamp:        time.Unix(int64(1654516425), 0),
		Nonce:            123456,
		Sequence:         789101112131415,
		EmitterChain:     vaa.ChainIDEthereum,
		EmitterAddress:   tokenBridgeAddress,
		Payload:          []byte("Hello, World!"),
		ConsistencyLevel: 32,
	}

	assert.Equal(t, expectedHashID, msg.TxIDString())
}

func TestMessagePublication_SetVerificationState(t *testing.T) {
	tests := []struct {
		name    string
		initial VerificationState
		arg     VerificationState
		wantErr bool
	}{
		{
			"can't overwrite existing status with default value",
			Valid,
			NotVerified,
			true,
		},
		{
			"can't overwrite with the same value",
			Valid,
			Valid,
			true,
		},
		{
			"happy path: default status to non-default",
			NotVerified,
			Valid,
			false,
		},
		{
			"happy path: non-default status to non-default",
			Rejected,
			Valid,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &MessagePublication{
				verificationState: tt.initial,
			}
			if err := msg.SetVerificationState(tt.arg); (err != nil) != tt.wantErr {
				t.Errorf("MessagePublication.SetVerificationState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
