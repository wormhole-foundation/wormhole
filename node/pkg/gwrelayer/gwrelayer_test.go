package gwrelayer

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func Test_convertBech32AddressToWormhole(t *testing.T) {
	expectedAddress, err := hex.DecodeString("ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b")
	require.NoError(t, err)

	// Basic success case.
	targetAddress, err := convertBech32AddressToWormhole("wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465")
	require.NoError(t, err)
	assert.Equal(t, true, bytes.Equal(expectedAddress, targetAddress.Bytes()))

	// Garbage in should generate an error.
	_, err = convertBech32AddressToWormhole("hello world!")
	assert.Error(t, err)

	// Empty input should generate an error.
	_, err = convertBech32AddressToWormhole("")
	assert.Error(t, err)
}

func Test_shouldPublish(t *testing.T) {
	type Test struct {
		label   string
		payload []byte
		result  bool
		err     bool
	}

	tests := []Test{
		{label: "empty payload", payload: []byte{}, result: false, err: false},
		{label: "non-transfer", payload: []byte{0x0}, result: false, err: false},
		{label: "payload type 1", payload: []byte{0x1}, result: false, err: false},
		{label: "payload too short", payload: []byte{0x3, 0x00, 0x00}, result: false, err: true},
		{label: "wrong target chain", payload: decodeBytes("0300000000000000000000000000000000000000000000000000000000000000640000000000000000000000005425890298aed601595a70ab815c96711a31bc6500066d9ae6b2d333c1d65301a59da3eed388ca5dc60cb12496584b75cbe6b15fdbed0020000000000000000000000000e6990c7e206d418d62b9e50c8e61f59dc360183b7b2262617369635f726563697069656e74223a7b22726563697069656e74223a22633256704d57786c656d3179636d31336348687865575679626e6c344d33706a595768735a4756715958686e4f485a364f484e774d32526f227d7d"), result: false, err: false},
		{label: "wrong target address", payload: decodeBytes("0300000000000000000000000000000000000000000000000000000000000000640000000000000000000000005425890298aed601595a70ab815c96711a31bc6500066d9ae6b2d333c1d65301a59da3eed388ca5dc60cb12496584b75cbe6b15fdbed0C20000000000000000000000000e6990c7e206d418d62b9e50c8e61f59dc360183b7b2262617369635f726563697069656e74223a7b22726563697069656e74223a22633256704d57786c656d3179636d31336348687865575679626e6c344d33706a595768735a4756715958686e4f485a364f484e774d32526f227d7d"), result: false, err: false},
		{label: "should publish", payload: decodeBytes("0300000000000000000000000000000000000000000000000000000000000000640000000000000000000000005425890298aed601595a70ab815c96711a31bc650006ade4a5f5803a439835c636395a8d648dee57b2fc90d98dc17fa887159b69638b0C20000000000000000000000000e6990c7e206d418d62b9e50c8e61f59dc360183b7b2262617369635f726563697069656e74223a7b22726563697069656e74223a22633256704d57786c656d3179636d31336348687865575679626e6c344d33706a595768735a4756715958686e4f485a364f484e774d32526f227d7d"), result: true, err: false},
	}

	targetAddress, err := convertBech32AddressToWormhole("wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465")
	require.NoError(t, err)

	for _, tc := range tests {
		t.Run(string(tc.label), func(t *testing.T) {
			result, err := shouldPublish(tc.payload, vaa.ChainIDWormchain, targetAddress)
			assert.Equal(t, tc.err, err != nil)
			assert.Equal(t, tc.result, result)
		})
	}
}

func decodeBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}
