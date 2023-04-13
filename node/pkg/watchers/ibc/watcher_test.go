package ibc

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/tidwall/gjson"

	"go.uber.org/zap"
)

func TestParseIbcReceivePublishEvent(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	eventJson := "{\"type\":\"wasm\",\"attributes\":[" +
		"{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxbmM1dGF0YWZ2NmV5cTdsbGtyMmd2NTBmZjllMjJtbmY3MHFnamx2NzM3a3RtdDRlc3dycTBrZGhjag==\",\"index\":true}," +
		"{\"key\":\"YWN0aW9u\",\"value\":\"cmVjZWl2ZV9wdWJsaXNo\",\"index\":true}," +
		"{\"key\":\"Y2hhbm5lbF9pZA==\",\"value\":\"ImNoYW5uZWwtMCI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5tZXNzYWdl\",\"value\":\"IkFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQVE9Ig==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZW5kZXI=\",\"value\":\"IjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDM1NzQzMDc0OTU2YzcxMDgwMGU4MzE5ODAxMWNjYmQ0ZGRmMTU1NmQi\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5jaGFpbl9pZA==\",\"value\":\"MTg=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ub25jZQ==\",\"value\":\"NDI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZXF1ZW5jZQ==\",\"value\":\"MTIzNDU2\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja190aW1l\",\"value\":\"MTY3NzY5MTA1Mw==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja19oZWlnaHQ=\",\"value\":\"NDM0\",\"index\":true}" +
		"]}"

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"

	evt, err := parseWasmEvent[ibcReceivePublishEvent](logger, contractAddress, "receive_publish", event)
	require.NoError(t, err)
	require.NotNil(t, evt)

	expectedSender, err := vaa.StringToAddress("00000000000000000000000035743074956c710800e83198011ccbd4ddf1556d")
	require.NoError(t, err)

	expectedPayload, err := hex.DecodeString("0000000000000000000000000000000000000000000000000000000000000004")
	require.NoError(t, err)

	expectedResult := ibcReceivePublishEvent{
		ChannelID:      "channel-0",
		EmitterAddress: expectedSender,
		EmitterChain:   vaa.ChainIDTerra2,
		Nonce:          42,
		Sequence:       123456,
		Timestamp:      1677691053,
		Payload:        expectedPayload,
	}
	assert.Equal(t, expectedResult, *evt)
}

func TestParseEventOfWrongType(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	eventJson := "{\"type\":\"hello\",\"attributes\":[" +
		"{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxbmM1dGF0YWZ2NmV5cTdsbGtyMmd2NTBmZjllMjJtbmY3MHFnamx2NzM3a3RtdDRlc3dycTBrZGhjag==\",\"index\":true}," +
		"{\"key\":\"YWN0aW9u\",\"value\":\"cmVjZWl2ZV9wdWJsaXNo\",\"index\":true}," +
		"{\"key\":\"Y2hhbm5lbF9pZA==\",\"value\":\"ImNoYW5uZWwtMCI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5tZXNzYWdl\",\"value\":\"IkFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQVE9Ig==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZW5kZXI=\",\"value\":\"IjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDM1NzQzMDc0OTU2YzcxMDgwMGU4MzE5ODAxMWNjYmQ0ZGRmMTU1NmQi\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5jaGFpbl9pZA==\",\"value\":\"MTg=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ub25jZQ==\",\"value\":\"NDI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZXF1ZW5jZQ==\",\"value\":\"MTIzNDU2\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja190aW1l\",\"value\":\"MTY3NzY5MTA1Mw==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja19oZWlnaHQ=\",\"value\":\"NDM0\",\"index\":true}" +
		"]}"

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"

	evt, err := parseWasmEvent[ibcReceivePublishEvent](logger, contractAddress, "receive_publish", event)
	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestParseEventForWrongContract(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	eventJson := "{\"type\":\"wasm\",\"attributes\":[" +
		"{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxbmM1dGF0YWZ2NmV5cTdsbGtyMmd2NTBmZjllMjJtbmY3MHFnamx2NzM3a3RtdDRlc3dycTBrZGhjag==\",\"index\":true}," +
		"{\"key\":\"YWN0aW9u\",\"value\":\"cmVjZWl2ZV9wdWJsaXNo\",\"index\":true}," +
		"{\"key\":\"Y2hhbm5lbF9pZA==\",\"value\":\"ImNoYW5uZWwtMCI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5tZXNzYWdl\",\"value\":\"IkFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQVE9Ig==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZW5kZXI=\",\"value\":\"IjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDM1NzQzMDc0OTU2YzcxMDgwMGU4MzE5ODAxMWNjYmQ0ZGRmMTU1NmQi\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5jaGFpbl9pZA==\",\"value\":\"MTg=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ub25jZQ==\",\"value\":\"NDI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZXF1ZW5jZQ==\",\"value\":\"MTIzNDU2\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja190aW1l\",\"value\":\"MTY3NzY5MTA1Mw==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja19oZWlnaHQ=\",\"value\":\"NDM0\",\"index\":true}" +
		"]}"

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "someOtherContract"

	evt, err := parseWasmEvent[ibcReceivePublishEvent](logger, contractAddress, "receive_publish", event)
	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestParseEventForWrongAction(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	eventJson := "{\"type\":\"wasm\",\"attributes\":[" +
		"{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxbmM1dGF0YWZ2NmV5cTdsbGtyMmd2NTBmZjllMjJtbmY3MHFnamx2NzM3a3RtdDRlc3dycTBrZGhjag==\",\"index\":true}," +
		"{\"key\":\"YWN0aW9u\",\"value\":\"cmVjZWl2ZV9wa3Q=\",\"index\":true}," + // Changed action value to "receive_pkt"
		"{\"key\":\"Y2hhbm5lbF9pZA==\",\"value\":\"ImNoYW5uZWwtMCI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5tZXNzYWdl\",\"value\":\"IkFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQVE9Ig==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZW5kZXI=\",\"value\":\"IjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDM1NzQzMDc0OTU2YzcxMDgwMGU4MzE5ODAxMWNjYmQ0ZGRmMTU1NmQi\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5jaGFpbl9pZA==\",\"value\":\"MTg=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ub25jZQ==\",\"value\":\"NDI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZXF1ZW5jZQ==\",\"value\":\"MTIzNDU2\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja190aW1l\",\"value\":\"MTY3NzY5MTA1Mw==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja19oZWlnaHQ=\",\"value\":\"NDM0\",\"index\":true}" +
		"]}"

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"

	evt, err := parseWasmEvent[ibcReceivePublishEvent](logger, contractAddress, "receive_publish", event)
	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestParseEventForNoContractSpecified(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	eventJson := "{\"type\":\"wasm\",\"attributes\":[" +
		// Not specifying a contract address.
		"{\"key\":\"Y2hhbm5lbF9pZA==\",\"value\":\"ImNoYW5uZWwtMCI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5tZXNzYWdl\",\"value\":\"IkFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQVE9Ig==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZW5kZXI=\",\"value\":\"IjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDM1NzQzMDc0OTU2YzcxMDgwMGU4MzE5ODAxMWNjYmQ0ZGRmMTU1NmQi\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5jaGFpbl9pZA==\",\"value\":\"MTg=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ub25jZQ==\",\"value\":\"NDI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZXF1ZW5jZQ==\",\"value\":\"MTIzNDU2\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja190aW1l\",\"value\":\"MTY3NzY5MTA1Mw==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja19oZWlnaHQ=\",\"value\":\"NDM0\",\"index\":true}" +
		"]}"

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"

	evt, err := parseWasmEvent[ibcReceivePublishEvent](logger, contractAddress, "receive_publish", event)
	require.NoError(t, err)
	assert.Nil(t, evt)
}

func TestParseEventForNoActionSpecified(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	eventJson := "{\"type\":\"wasm\",\"attributes\":[" +
		"{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxbmM1dGF0YWZ2NmV5cTdsbGtyMmd2NTBmZjllMjJtbmY3MHFnamx2NzM3a3RtdDRlc3dycTBrZGhjag==\",\"index\":true}," +
		// Not specifying the action.
		"{\"key\":\"Y2hhbm5lbF9pZA==\",\"value\":\"ImNoYW5uZWwtMCI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5tZXNzYWdl\",\"value\":\"IkFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQVE9Ig==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZW5kZXI=\",\"value\":\"IjAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDM1NzQzMDc0OTU2YzcxMDgwMGU4MzE5ODAxMWNjYmQ0ZGRmMTU1NmQi\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5jaGFpbl9pZA==\",\"value\":\"MTg=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ub25jZQ==\",\"value\":\"NDI=\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5zZXF1ZW5jZQ==\",\"value\":\"MTIzNDU2\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja190aW1l\",\"value\":\"MTY3NzY5MTA1Mw==\",\"index\":true}," +
		"{\"key\":\"bWVzc2FnZS5ibG9ja19oZWlnaHQ=\",\"value\":\"NDM0\",\"index\":true}" +
		"]}"

	require.Equal(t, true, gjson.Valid(eventJson))
	event := gjson.Parse(eventJson)

	contractAddress := "wormhole1nc5tatafv6eyq7llkr2gv50ff9e22mnf70qgjlv737ktmt4eswrq0kdhcj"

	evt, err := parseWasmEvent[ibcReceivePublishEvent](logger, contractAddress, "receive_publish", event)
	require.NoError(t, err)
	assert.Nil(t, evt)
}
