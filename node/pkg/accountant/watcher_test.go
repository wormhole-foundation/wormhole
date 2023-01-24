package accountant

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmAbci "github.com/tendermint/tendermint/abci/types"

	"go.uber.org/zap"
)

func TestParseWasmObservationFromTestTool(t *testing.T) {
	logger := zap.NewNop()

	eventJson := []byte("{\"type\":\"wasm-Observation\",\"attributes\":[{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxNDY2bmYzenV4cHlhOHE5ZW14dWtkN3ZmdGFmNmg0cHNyMGEwN3NybDV6dzc0emg4NHlqcTRseWptaA==\",\"index\":true},{\"key\":\"dHhfaGFzaA==\",\"value\":\"Imd1b2xOc1hSWnhnd3kwa1NENVJIbmpTMVJaYW8zVGFmdkNabVpucDJYMHM9Ig==\",\"index\":true},{\"key\":\"dGltZXN0YW1w\",\"value\":\"MTY3MjkzMjk5OA==\",\"index\":true},{\"key\":\"bm9uY2U=\",\"value\":\"MA==\",\"index\":true},{\"key\":\"ZW1pdHRlcl9jaGFpbg==\",\"value\":\"Mg==\",\"index\":true},{\"key\":\"ZW1pdHRlcl9hZGRyZXNz\",\"value\":\"IjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAyOTBmYjE2NzIwOGFmNDU1YmIxMzc3ODAxNjNiN2I3YTlhMTBjMTYi\",\"index\":true},{\"key\":\"c2VxdWVuY2U=\",\"value\":\"MTY3MjkzMjk5OA==\",\"index\":true},{\"key\":\"Y29uc2lzdGVuY3lfbGV2ZWw=\",\"value\":\"MTU=\",\"index\":true},{\"key\":\"dGVzdF9maWVsZA==\",\"value\":\"MTU=\",\"index\":true},{\"key\":\"cGF5bG9hZA==\",\"value\":\"IkFRQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUEzZ3RyT25aQUFBQUFBQUFBQUFBQUFBQUFBQUxZdm12d3VxZE9DcEJ3Rm1lY3JwR1E2QTNRb0FBZ0FBQUFBQUFBQUFBQUFBQU1FSUlKZy9NMFZzNTc2em9FYjFxRCtqVHdKOURDQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUE9PSI=\",\"index\":true}]}")
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

	eventJson := []byte("{\"type\":\"wasm-Observation\",\"attributes\":[{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxNDY2bmYzenV4cHlhOHE5ZW14dWtkN3ZmdGFmNmg0cHNyMGEwN3NybDV6dzc0emg4NHlqcTRseWptaA==\",\"index\":true},{\"key\":\"dHhfaGFzaA==\",\"value\":\"IlovM0x1bklSK0FaWjdRdllqS0dHSDBNZU94M1pIZlR1SHZ6TDAxdm9TcjQ9Ig==\",\"index\":true},{\"key\":\"dGltZXN0YW1w\",\"value\":\"OTUwNw==\",\"index\":true},{\"key\":\"bm9uY2U=\",\"value\":\"NTU0MzAzNzQ0\",\"index\":true},{\"key\":\"ZW1pdHRlcl9jaGFpbg==\",\"value\":\"Mg==\",\"index\":true},{\"key\":\"ZW1pdHRlcl9hZGRyZXNz\",\"value\":\"IjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAyOTBmYjE2NzIwOGFmNDU1YmIxMzc3ODAxNjNiN2I3YTlhMTBjMTYi\",\"index\":true},{\"key\":\"c2VxdWVuY2U=\",\"value\":\"MQ==\",\"index\":true},{\"key\":\"Y29uc2lzdGVuY3lfbGV2ZWw=\",\"value\":\"MQ==\",\"index\":true},{\"key\":\"cGF5bG9hZA==\",\"value\":\"IkFRQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBSlVDK1FBQUFBQUFBQUFBQUFBQUFBQTNiWlA1R3FSMUc3aWxDQlRuOEpmMEh4ZjZqNEFBZ0FBQUFBQUFBQUFBQUFBQUpENHYycEhueklPclFkRUVhU3c1NVJPcU1uQkFBUUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUE9PSI=\",\"index\":true}]}")
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

	eventJson := []byte("{\"type\":\"wasm-ObservationError\",\"attributes\":[{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxNDY2bmYzenV4cHlhOHE5ZW14dWtkN3ZmdGFmNmg0cHNyMGEwN3NybDV6dzc0emg4NHlqcTRseWptaA==\",\"index\":true},{\"key\":\"a2V5\",\"value\":\"eyJlbWl0dGVyX2NoYWluIjoyLCJlbWl0dGVyX2FkZHJlc3MiOiIwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMjkwZmIxNjcyMDhhZjQ1NWJiMTM3NzgwMTYzYjdiN2E5YTEwYzE2Iiwic2VxdWVuY2UiOjE2NzQxNDQ1NDV9\",\"index\":true},{\"key\":\"ZXJyb3I=\",\"value\":\"ImRpZ2VzdCBtaXNtYXRjaCBmb3IgcHJvY2Vzc2VkIG1lc3NhZ2Ui\",\"index\":true}]}")
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
