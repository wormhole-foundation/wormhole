package ibc

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestParseIbcReceivePublishEvent(t *testing.T) {
	logger := zap.NewNop()

	eventJson := `{"type": "wasm","attributes": [` +
		`{"key": "X2NvbnRyYWN0X2FkZHJlc3M=","value": "d29ybWhvbGUxbmM1dGF0YWZ2NmV5cTdsbGtyMmd2NTBmZjllMjJtbmY3MHFnamx2NzM3a3RtdDRlc3dycTBrZGhjag==","index": true},` +
		`{"key": "YWN0aW9u", "value": "cmVjZWl2ZV9wdWJsaXNo", "index": true},` +
		`{"key": "Y2hhbm5lbF9pZA==", "value": "Y2hhbm5lbC0w", "index": true},` +
		`{"key": "bWVzc2FnZS5tZXNzYWdl","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwNA==","index": true},` +
		`{"key": "bWVzc2FnZS5zZW5kZXI=","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMzU3NDMwNzQ5NTZjNzEwODAwZTgzMTk4MDExY2NiZDRkZGYxNTU2ZA==","index": true},` +
		`{ "key": "bWVzc2FnZS5jaGFpbl9pZA==", "value": "MTg=", "index": true },` +
		`{ "key": "bWVzc2FnZS5ub25jZQ==", "value": "MQ==", "index": true },` +
		`{ "key": "bWVzc2FnZS5zZXF1ZW5jZQ==", "value": "Mg==", "index": true },` +
		`{"key": "bWVzc2FnZS5ibG9ja190aW1l","value": "MTY4MDA5OTgxNA==","index": true},` +
		`{"key": "bWVzc2FnZS5ibG9ja19oZWlnaHQ=","value": "MjYxMw==","index": true}` +
		`]}`

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"

	txHash, err := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	require.NoError(t, err)

	evt, err := parseIbcReceivePublishEvent(logger, contractAddress, event, txHash)
	require.NoError(t, err)
	require.NotNil(t, evt)

	expectedSender, err := vaa.StringToAddress("00000000000000000000000035743074956c710800e83198011ccbd4ddf1556d")
	require.NoError(t, err)

	expectedPayload, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000004")
	require.NoError(t, err)

	expectedResult := ibcReceivePublishEvent{
		ChannelID: "channel-0",
		Msg: &common.MessagePublication{
			TxHash:         txHash,
			EmitterAddress: expectedSender,
			EmitterChain:   vaa.ChainIDTerra2,
			Nonce:          1,
			Sequence:       2,
			Timestamp:      time.Unix(1680099814, 0),
			Payload:        expectedPayload,
		},
	}
	// Use DeepEqual() because the response contains pointers.
	assert.True(t, reflect.DeepEqual(expectedResult, *evt))
}

func TestParseEventForWrongContract(t *testing.T) {
	logger := zap.NewNop()

	eventJson := `{"type": "wasm","attributes": [` +
		`{"key": "X2NvbnRyYWN0X2FkZHJlc3M=","value": "d29ybWhvbGUxbmM1dGF0YWZ2NmV5cTdsbGtyMmd2NTBmZjllMjJtbmY3MHFnamx2NzM3a3RtdDRlc3dycTBrZGhjag==","index": true},` +
		`{"key": "YWN0aW9u", "value": "cmVjZWl2ZV9wdWJsaXNo", "index": true},` +
		`{"key": "Y2hhbm5lbF9pZA==", "value": "Y2hhbm5lbC0w", "index": true},` +
		`{"key": "bWVzc2FnZS5tZXNzYWdl","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwNA==","index": true},` +
		`{"key": "bWVzc2FnZS5zZW5kZXI=","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMzU3NDMwNzQ5NTZjNzEwODAwZTgzMTk4MDExY2NiZDRkZGYxNTU2ZA==","index": true},` +
		`{ "key": "bWVzc2FnZS5jaGFpbl9pZA==", "value": "MTg=", "index": true },` +
		`{ "key": "bWVzc2FnZS5ub25jZQ==", "value": "MQ==", "index": true },` +
		`{ "key": "bWVzc2FnZS5zZXF1ZW5jZQ==", "value": "Mg==", "index": true },` +
		`{"key": "bWVzc2FnZS5ibG9ja190aW1l","value": "MTY4MDA5OTgxNA==","index": true},` +
		`{"key": "bWVzc2FnZS5ibG9ja19oZWlnaHQ=","value": "MjYxMw==","index": true}` +
		`]}`

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "someOtherContract"

	txHash, err := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	require.NoError(t, err)

	_, err = parseIbcReceivePublishEvent(logger, contractAddress, event, txHash)
	assert.Error(t, err)
}

func TestParseEventForWrongAction(t *testing.T) {
	logger := zap.NewNop()

	eventJson := `{"type": "wasm","attributes": [` +
		`{"key": "X2NvbnRyYWN0X2FkZHJlc3M=","value": "d29ybWhvbGUxbmM1dGF0YWZ2NmV5cTdsbGtyMmd2NTBmZjllMjJtbmY3MHFnamx2NzM3a3RtdDRlc3dycTBrZGhjag==","index": true},` +
		`{"key": "YWN0aW9u", "value": "cmVjZWl2ZV9wa3Q=", "index": true},` + // Changed action value to "receive_pkt"
		`{"key": "Y2hhbm5lbF9pZA==", "value": "Y2hhbm5lbC0w", "index": true},` +
		`{"key": "bWVzc2FnZS5tZXNzYWdl","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwNA==","index": true},` +
		`{"key": "bWVzc2FnZS5zZW5kZXI=","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMzU3NDMwNzQ5NTZjNzEwODAwZTgzMTk4MDExY2NiZDRkZGYxNTU2ZA==","index": true},` +
		`{ "key": "bWVzc2FnZS5jaGFpbl9pZA==", "value": "MTg=", "index": true },` +
		`{ "key": "bWVzc2FnZS5ub25jZQ==", "value": "MQ==", "index": true },` +
		`{ "key": "bWVzc2FnZS5zZXF1ZW5jZQ==", "value": "Mg==", "index": true },` +
		`{"key": "bWVzc2FnZS5ibG9ja190aW1l","value": "MTY4MDA5OTgxNA==","index": true},` +
		`{"key": "bWVzc2FnZS5ibG9ja19oZWlnaHQ=","value": "MjYxMw==","index": true}` +
		`]}`

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"

	txHash, err := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	require.NoError(t, err)

	evt, err := parseIbcReceivePublishEvent(logger, contractAddress, event, txHash)
	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestParseEventForNoContractSpecified(t *testing.T) {
	logger := zap.NewNop()

	eventJson := `{"type": "wasm","attributes": [` +
		// No contract specified
		`{"key": "YWN0aW9u", "value": "cmVjZWl2ZV9wdWJsaXNo", "index": true},` +
		`{"key": "Y2hhbm5lbF9pZA==", "value": "Y2hhbm5lbC0w", "index": true},` +
		`{"key": "bWVzc2FnZS5tZXNzYWdl","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwNA==","index": true},` +
		`{"key": "bWVzc2FnZS5zZW5kZXI=","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMzU3NDMwNzQ5NTZjNzEwODAwZTgzMTk4MDExY2NiZDRkZGYxNTU2ZA==","index": true},` +
		`{ "key": "bWVzc2FnZS5jaGFpbl9pZA==", "value": "MTg=", "index": true },` +
		`{ "key": "bWVzc2FnZS5ub25jZQ==", "value": "MQ==", "index": true },` +
		`{ "key": "bWVzc2FnZS5zZXF1ZW5jZQ==", "value": "Mg==", "index": true },` +
		`{"key": "bWVzc2FnZS5ibG9ja190aW1l","value": "MTY4MDA5OTgxNA==","index": true},` +
		`{"key": "bWVzc2FnZS5ibG9ja19oZWlnaHQ=","value": "MjYxMw==","index": true}` +
		`]}`

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"

	txHash, err := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	require.NoError(t, err)

	_, err = parseIbcReceivePublishEvent(logger, contractAddress, event, txHash)
	assert.Error(t, err)
}

func TestParseEventForNoActionSpecified(t *testing.T) {
	logger := zap.NewNop()

	eventJson := `{"type": "wasm","attributes": [` +
		`{"key": "X2NvbnRyYWN0X2FkZHJlc3M=","value": "d29ybWhvbGUxbmM1dGF0YWZ2NmV5cTdsbGtyMmd2NTBmZjllMjJtbmY3MHFnamx2NzM3a3RtdDRlc3dycTBrZGhjag==","index": true},` +
		// No action specified
		`{"key": "Y2hhbm5lbF9pZA==", "value": "Y2hhbm5lbC0w", "index": true},` +
		`{"key": "bWVzc2FnZS5tZXNzYWdl","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwNA==","index": true},` +
		`{"key": "bWVzc2FnZS5zZW5kZXI=","value": "MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMzU3NDMwNzQ5NTZjNzEwODAwZTgzMTk4MDExY2NiZDRkZGYxNTU2ZA==","index": true},` +
		`{ "key": "bWVzc2FnZS5jaGFpbl9pZA==", "value": "MTg=", "index": true },` +
		`{ "key": "bWVzc2FnZS5ub25jZQ==", "value": "MQ==", "index": true },` +
		`{ "key": "bWVzc2FnZS5zZXF1ZW5jZQ==", "value": "Mg==", "index": true },` +
		`{"key": "bWVzc2FnZS5ibG9ja190aW1l","value": "MTY4MDA5OTgxNA==","index": true},` +
		`{"key": "bWVzc2FnZS5ibG9ja19oZWlnaHQ=","value": "MjYxMw==","index": true}` +
		`]}`

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"

	txHash, err := vaa.StringToHash("82ea2536c5d1671830cb49120f94479e34b54596a8dd369fbc2666667a765f4b")
	require.NoError(t, err)

	evt, err := parseIbcReceivePublishEvent(logger, contractAddress, event, txHash)
	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestParseIbcAllChannelChainsQueryResults(t *testing.T) {
	respJson := []byte(`
	{
		"data": {
			"channels_chains": [
				[
					"Y2hhbm5lbC0w",
					18
				],
				[
					"Y2hhbm5lbC00Mg==",
					22
				]
			]
		}
	}
	`)

	var result ibcAllChannelChainsQueryResults
	err := json.Unmarshal(respJson, &result)
	require.NoError(t, err)

	expectedChannStr1 := base64.StdEncoding.EncodeToString([]byte("channel-0"))
	expectedChannStr2 := base64.StdEncoding.EncodeToString([]byte("channel-42"))

	require.Equal(t, 2, len(result.Data.ChannelChains))
	require.Equal(t, 2, len(result.Data.ChannelChains[0]))
	assert.Equal(t, expectedChannStr1, result.Data.ChannelChains[0][0].(string))   //nolint:forcetypeassert
	assert.Equal(t, uint16(18), uint16(result.Data.ChannelChains[0][1].(float64))) //nolint:forcetypeassert
	assert.Equal(t, expectedChannStr2, result.Data.ChannelChains[1][0].(string))   //nolint:forcetypeassert
	assert.Equal(t, uint16(22), uint16(result.Data.ChannelChains[1][1].(float64))) //nolint:forcetypeassert
}
