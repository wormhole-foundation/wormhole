package ibc

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strconv"
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

	eventJson := `{"type": "wasm", "attributes": [
			{"key": "_contract_address", "value": "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj", "index": true},
			{"key": "action", "value": "receive_publish", "index": true},
			{"key": "channel_id", "value": "channel-0", "index": true},
			{"key": "message.message", "value": "0000000000000000000000000000000000000000000000000000000000000004", "index": true},
			{"key": "message.sender", "value": "00000000000000000000000035743074956c710800e83198011ccbd4ddf1556d", "index": true},
			{"key": "message.chain_id", "value": "18", "index": true},
			{"key": "message.nonce", "value": "1", "index": true},
			{"key": "message.sequence", "value": "2", "index": true},
			{"key": "message.block_time", "value": "1680099814", "index": true},
			{"key": "message.block_height", "value": "2613", "index": true}
		]}`

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
			TxID:           txHash.Bytes(),
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

	eventJson := `{"type":"wasm","attributes":[
		{"key":"_contract_address","value":"wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj","index":true},
		{"key":"action","value":"receive_publish","index":true},{"key":"channel_id","value":"channel-0","index":true},
		{"key":"message.message","value":"0000000000000000000000000000000000000000000000000000000000000004","index":true},
		{"key":"message.sender","value":"00000000000000000000000035743074956c710800e83198011ccbd4ddf1556d","index":true},
		{"key":"message.chain_id","value":"18","index":true},{"key":"message.nonce","value":"1","index":true},
		{"key":"message.sequence","value":"2","index":true},{"key":"message.block_time","value":"1680099814","index":true},
		{"key":"message.block_height","value":"2613","index":true}
	]}`

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

	eventJson := `{"type":"wasm","attributes":[
		{"key":"_contract_address","value":"wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj","index":true},
		{"key":"action","value":"receive_pkt","index":true},{"key":"channel_id","value":"channel-0","index":true},
		{"key":"message.message","value":"0000000000000000000000000000000000000000000000000000000000000004","index":true},
		{"key":"message.sender","value":"00000000000000000000000035743074956c710800e83198011ccbd4ddf1556d","index":true},
		{"key":"message.chain_id","value":"18","index":true},{"key":"message.nonce","value":"1","index":true},
		{"key":"message.sequence","value":"2","index":true},{"key":"message.block_time","value":"1680099814","index":true},
		{"key":"message.block_height","value":"2613","index":true}
	]}`

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

	eventJson := `{"type":"wasm","attributes":[
		{"key":"action","value":"receive_publish","index":true},
		{"key":"channel_id","value":"channel-0","index":true},
		{"key":"message.message","value":"0000000000000000000000000000000000000000000000000000000000000004","index":true},
		{"key":"message.sender","value":"00000000000000000000000035743074956c710800e83198011ccbd4ddf1556d","index":true},
		{"key":"message.chain_id","value":"18","index":true},{"key":"message.nonce","value":"1","index":true},
		{"key":"message.sequence","value":"2","index":true},{"key":"message.block_time","value":"1680099814","index":true},
		{"key":"message.block_height","value":"2613","index":true}
	]}`

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

	eventJson := `{"type":"wasm","attributes":[
		{"key":"_contract_address","value":"wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj","index":true},
		{"key":"channel_id","value":"channel-0","index":true},{"key":"message.message","value":"0000000000000000000000000000000000000000000000000000000000000004","index":true},
		{"key":"message.sender","value":"00000000000000000000000035743074956c710800e83198011ccbd4ddf1556d","index":true},
		{"key":"message.chain_id","value":"18","index":true},
		{"key":"message.nonce","value":"1","index":true},
		{"key":"message.sequence","value":"2","index":true},
		{"key":"message.block_time","value":"1680099814","index":true},
		{"key":"message.block_height","value":"2613","index":true}
	]}`

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

func TestConvertingWsUrlToHttpUrl(t *testing.T) {
	assert.Equal(t, "http://wormchain:26657", convertWsUrlToHttpUrl("ws://wormchain:26657/websocket"))
	assert.Equal(t, "http://wormchain:26657", convertWsUrlToHttpUrl("ws://wormchain:26657"))
	assert.Equal(t, "http://wormchain:26657", convertWsUrlToHttpUrl("wss://wormchain:26657/websocket"))
	assert.Equal(t, "http://wormchain:26657", convertWsUrlToHttpUrl("wss://wormchain:26657"))
	assert.Equal(t, "http://wormchain:26657", convertWsUrlToHttpUrl("wormchain:26657"))
}

func TestParseAbciInfoResults(t *testing.T) {
	// This came from the following query: http://localhost:26659/abci_info
	respJson := []byte(`
{
  "jsonrpc": "2.0",
  "id": -1,
  "result": {
    "response": {
      "data": "wormchain",
      "version": "v0.0.1",
      "last_block_height": "2037",
      "last_block_app_hash": "7lVJBWOpP+owbc0Gohn4htF6s2J2DrbjhdL9m79lAjU="
    }
  }
}
	`)

	var resp abciInfoResults
	err := json.Unmarshal(respJson, &resp)
	require.NoError(t, err)

	assert.Equal(t, "v0.0.1", resp.Result.Response.Version)
	assert.Equal(t, "2037", resp.Result.Response.LastBlockHeight)

	blockHeight, err := strconv.ParseInt(resp.Result.Response.LastBlockHeight, 10, 64)
	require.NoError(t, err)
	assert.Equal(t, int64(2037), blockHeight)
	assert.Equal(t, float64(2037), float64(blockHeight)) // We need it as a float to post it to Prometheus.
}
