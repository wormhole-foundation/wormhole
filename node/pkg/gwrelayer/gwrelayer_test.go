package gwrelayer

import (
	"bytes"
	"encoding/hex"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	sdktypes "github.com/cosmos/cosmos-sdk/types"
	"go.uber.org/zap"
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

func Test_shouldPublishToIbcTranslator(t *testing.T) {
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
			result, err := shouldPublishToIbcTranslator(tc.payload, vaa.ChainIDWormchain, targetAddress)
			assert.Equal(t, tc.err, err != nil)
			assert.Equal(t, tc.result, result)
		})
	}
}

func Test_shouldPublishToTokenBridge(t *testing.T) {
	type Test struct {
		label   string
		chain   vaa.ChainID
		address vaa.Address
		payload []byte
		result  bool
	}

	logger := zap.NewNop()

	tokenBridges, tokenBridgeAddress, err := buildTokenBridgeMap(logger, common.MainNet)
	require.NoError(t, err)
	require.NotNil(t, tokenBridges)
	require.Equal(t, tokenBridgeAddress, "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh")

	tests := []Test{
		{label: "unknown chain", chain: vaa.ChainIDUnset, address: addr("0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"), payload: []byte{}, result: false},
		{label: "unknown emitter", chain: vaa.ChainIDEthereum, address: addr("0000000000000000000000000000000000000000000000000000000000000000"), payload: []byte{}, result: false},
		{label: "wormchain", chain: vaa.ChainIDWormchain, address: addr("aeb534c45c3049d380b9d9b966f9895f53abd4301bfaff407fa09dea8ae7a924"), payload: []byte{}, result: false},
		{label: "empty payload", chain: vaa.ChainIDEthereum, address: addr("0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"), payload: []byte{}, result: false},
		{label: "not an attest", chain: vaa.ChainIDEthereum, address: addr("0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"), payload: []byte{0x1}, result: false},
		{label: "should publish", chain: vaa.ChainIDEthereum, address: addr("0000000000000000000000003ee18b2214aff97000d974cf647e7c347e8fa585"), payload: []byte{0x2}, result: true},
	}

	for _, tc := range tests {
		t.Run(string(tc.label), func(t *testing.T) {
			v := &vaa.VAA{
				Version:          uint8(1),
				GuardianSetIndex: uint32(1),
				Signatures:       []*vaa.Signature{},
				Timestamp:        time.Unix(0, 0),
				Nonce:            uint32(1),
				Sequence:         uint64(1),
				ConsistencyLevel: uint8(32),
				EmitterChain:     tc.chain,
				EmitterAddress:   tc.address,
				Payload:          tc.payload,
			}

			result := shouldPublishToTokenBridge(tokenBridges, v)
			assert.Equal(t, tc.result, result)
		})
	}

	_, err = sdktypes.Bech32ifyAddressBytes("wormhole", decodeBytes("aeb534c45c3049d380b9d9b966f9895f53abd4301bfaff407fa09dea8ae7a924"))
	require.NoError(t, err)
}

func decodeBytes(s string) []byte {
	b, err := hex.DecodeString(s)
	if err != nil {
		panic(err)
	}
	return b
}

func addr(str string) vaa.Address {
	a, err := vaa.StringToAddress(str)
	if err != nil {
		panic("failed to convert address")
	}
	return a
}

func Test_verifyDevnetTokenBridgeAddress(t *testing.T) {
	tokenBridgeAddressInTilt := "wormhole1eyfccmjm6732k7wp4p6gdjwhxjwsvje44j0hfx8nkgrm8fs7vqfssvpdkx" //nolint:gosec
	targetAddress, err := convertBech32AddressToWormhole(tokenBridgeAddressInTilt)
	require.NoError(t, err)

	expectedAddress, exists := sdk.KnownDevnetTokenbridgeEmitters[vaa.ChainIDWormchain]
	require.True(t, exists)
	assert.True(t, bytes.Equal(expectedAddress[:], targetAddress[:]))
}
