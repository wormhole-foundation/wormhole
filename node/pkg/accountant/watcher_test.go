package accountant

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmAbci "github.com/cometbft/cometbft/abci/types"

	"go.uber.org/zap"
)

func TestParseWasmObservationFromTestTool(t *testing.T) {
	logger := zap.NewNop()

	eventJson := []byte(`{"type":"wasm-Observation","attributes":[
		{"key":"_contract_address","value":"wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh","index":true},
		{"key":"tx_hash","value":"\"guolNsXRZxgwy0kSD5RHnjS1RZao3TafvCZmZnp2X0s=\"","index":true},
		{"key":"timestamp","value":"1672932998","index":true},
		{"key":"nonce","value":"0","index":true},
		{"key":"emitter_chain","value":"2","index":true},
		{"key":"emitter_address","value":"\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\"","index":true},
		{"key":"sequence","value":"1672932998","index":true},
		{"key":"consistency_level","value":"15","index":true},
		{"key":"test_field","value":"15","index":true},
		{"key":"payload","value":"\"AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA3gtrOnZAAAAAAAAAAAAAAAAAAALYvmvwuqdOCpBwFmecrpGQ6A3QoAAgAAAAAAAAAAAAAAAMEIIJg/M0Vs576zoEb1qD+jTwJ9DCAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\"","index":true}
	]}`)

	event := tmAbci.Event{}
	err := json.Unmarshal(eventJson, &event)
	require.NoError(t, err)

	xfer, err := parseEvent[WasmObservation](logger, event, "wasm-Observation", "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh")
	require.NoError(t, err)
	require.NotNil(t, xfer)

	expectedTxHash, err := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	require.NoError(t, err)

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedPayload, err := hex.DecodeString("010000000000000000000000000000000000000000000000000de0b6b3a76400000000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a0002000000000000000000000000c10820983f33456ce7beb3a046f5a83fa34f027d0c200000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	expectedResult := WasmObservation{
		TxHash:           expectedTxHash.Bytes(),
		Timestamp:        1672932998,
		Nonce:            0,
		EmitterChain:     uint16(vaa.ChainIDEthereum),
		EmitterAddress:   expectedEmitterAddress,
		Sequence:         1672932998,
		ConsistencyLevel: 15,
		Payload:          expectedPayload,
	}
	assert.Equal(t, expectedResult, *xfer)
}

func TestParseWasmObservationFromPortalBridge(t *testing.T) {
	logger := zap.NewNop()

	eventJson := []byte(`{"type":"wasm-Observation","attributes":[
		{"key":"_contract_address","value":"wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh","index":true},
		{"key":"tx_hash","value":"\"Z/3LunIR+AZZ7QvYjKGGH0MeOx3ZHfTuHvzL01voSr4=\"","index":true},
		{"key":"timestamp","value":"9507","index":true},
		{"key":"nonce","value":"554303744","index":true},
		{"key":"emitter_chain","value":"2","index":true},
		{"key":"emitter_address","value":"\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\"","index":true},
		{"key":"sequence","value":"1","index":true},
		{"key":"consistency_level","value":"1","index":true},
		{"key":"payload","value":"\"AQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJUC+QAAAAAAAAAAAAAAAAA3bZP5GqR1G7ilCBTn8Jf0Hxf6j4AAgAAAAAAAAAAAAAAAJD4v2pHnzIOrQdEEaSw55ROqMnBAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\"","index":true}]}
	`)

	event := tmAbci.Event{}
	err := json.Unmarshal(eventJson, &event)
	require.NoError(t, err)

	xfer, err := parseEvent[WasmObservation](logger, event, "wasm-Observation", "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh")
	require.NoError(t, err)
	require.NotNil(t, xfer)

	expectedTxHash, err := vaa.StringToHash("67fdcbba7211f80659ed0bd88ca1861f431e3b1dd91df4ee1efccbd35be84abe")
	require.NoError(t, err)

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedPayload, err := hex.DecodeString("0100000000000000000000000000000000000000000000000000000002540be400000000000000000000000000ddb64fe46a91d46ee29420539fc25fd07c5fea3e000200000000000000000000000090f8bf6a479f320ead074411a4b0e7944ea8c9c100040000000000000000000000000000000000000000000000000000000000000000")
	require.NoError(t, err)

	expectedResult := WasmObservation{
		TxHash:           expectedTxHash.Bytes(),
		Timestamp:        9507,
		Nonce:            554303744,
		EmitterChain:     uint16(vaa.ChainIDEthereum),
		EmitterAddress:   expectedEmitterAddress,
		Sequence:         1,
		ConsistencyLevel: 1,
		Payload:          expectedPayload,
	}

	assert.Equal(t, expectedResult, *xfer)
}

func TestParseWasmObservationError(t *testing.T) {
	logger := zap.NewNop()

	eventJson := []byte(`{"type":"wasm-ObservationError","attributes":[
		{"key":"_contract_address","value":"wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh","index":true},
		{"key":"key","value":"{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1674144545}","index":true},
		{"key":"error","value":"\"digest mismatch for processed message\"","index":true}]}
	`)

	event := tmAbci.Event{}
	err := json.Unmarshal(eventJson, &event)
	require.NoError(t, err)

	evt, err := parseEvent[WasmObservationError](logger, event, "wasm-ObservationError", "wormhole1466nf3zuxpya8q9emxukd7vftaf6h4psr0a07srl5zw74zh84yjq4lyjmh")
	require.NoError(t, err)
	require.NotNil(t, evt)

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedResult := WasmObservationError{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674144545,
		},
		Error: "digest mismatch for processed message",
	}

	assert.Equal(t, expectedResult, *evt)
}
