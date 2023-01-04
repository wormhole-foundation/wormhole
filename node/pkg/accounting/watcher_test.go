package accounting

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	tmAbci "github.com/tendermint/tendermint/abci/types"

	"go.uber.org/zap"
)

func TestParseWasmTransfer(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	eventJson := []byte("{\"type\":\"wasm-Transfer\",\"attributes\":[{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxNDY2bmYzenV4cHlhOHE5ZW14dWtkN3ZmdGFmNmg0cHNyMGEwN3NybDV6dzc0emg4NHlqcTRseWptaA==\",\"index\":true},{\"key\":\"dHhfaGFzaA==\",\"value\":\"Z3VvbE5zWFJaeGd3eTBrU0Q1UkhualMxUlphbzNUYWZ2Q1ptWm5wMlgwcz0=\",\"index\":true},{\"key\":\"dGltZXN0YW1w\",\"value\":\"MTY3Mjg2MjYxMQ==\",\"index\":true},{\"key\":\"bm9uY2U=\",\"value\":\"MA==\",\"index\":true},{\"key\":\"ZW1pdHRlcl9jaGFpbg==\",\"value\":\"Mg==\",\"index\":true},{\"key\":\"ZW1pdHRlcl9hZGRyZXNz\",\"value\":\"MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDI5MGZiMTY3MjA4YWY0NTViYjEzNzc4MDE2M2I3YjdhOWExMGMxNg==\",\"index\":true},{\"key\":\"c2VxdWVuY2U=\",\"value\":\"MTY3Mjg2MjYxMQ==\",\"index\":true},{\"key\":\"Y29uc2lzdGVuY3lfbGV2ZWw=\",\"value\":\"MTU=\",\"index\":true},{\"key\":\"cGF5bG9hZA==\",\"value\":\"QVFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQTNndHJPblpBQUFBQUFBQUFBQUFBQUFBQUFBTFl2bXZ3dXFkT0NwQndGbWVjcnBHUTZBM1FvQUFnQUFBQUFBQUFBQUFBQUFBTUVJSUpnL00wVnM1NzZ6b0ViMXFEK2pUd0o5RENBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQT09\",\"index\":true}]}")
	event := tmAbci.Event{}
	err := json.Unmarshal(eventJson, &event)
	require.NoError(t, err)

	xfer, err := parseWasmTransfer(logger, event)
	require.NoError(t, err)
	require.NotNil(t, xfer)
	assert.Equal(t, 1672862611, xfer.Timestamp)
}

type WasmTransferAsStrings struct {
	TxHash           string `json:"tx_hash"`
	Timestamp        string `json:"timestamp"`
	Nonce            string `json:"nonce"`
	EmitterChain     string `json:"emitter_chain"`
	EmitterAddress   string `json:"emitter_address"`
	Sequence         string `json:"sequence"`
	ConsistencyLevel string `json:"consistency_level"`
	Payload          string `json:"payload"`
}

func TestParseWasmTransferAsStrings(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	eventJson := []byte("{\"type\":\"wasm-Transfer\",\"attributes\":[{\"key\":\"X2NvbnRyYWN0X2FkZHJlc3M=\",\"value\":\"d29ybWhvbGUxNDY2bmYzenV4cHlhOHE5ZW14dWtkN3ZmdGFmNmg0cHNyMGEwN3NybDV6dzc0emg4NHlqcTRseWptaA==\",\"index\":true},{\"key\":\"dHhfaGFzaA==\",\"value\":\"Z3VvbE5zWFJaeGd3eTBrU0Q1UkhualMxUlphbzNUYWZ2Q1ptWm5wMlgwcz0=\",\"index\":true},{\"key\":\"dGltZXN0YW1w\",\"value\":\"MTY3Mjg2MjYxMQ==\",\"index\":true},{\"key\":\"bm9uY2U=\",\"value\":\"MA==\",\"index\":true},{\"key\":\"ZW1pdHRlcl9jaGFpbg==\",\"value\":\"Mg==\",\"index\":true},{\"key\":\"ZW1pdHRlcl9hZGRyZXNz\",\"value\":\"MDAwMDAwMDAwMDAwMDAwMDAwMDAwMDAwMDI5MGZiMTY3MjA4YWY0NTViYjEzNzc4MDE2M2I3YjdhOWExMGMxNg==\",\"index\":true},{\"key\":\"c2VxdWVuY2U=\",\"value\":\"MTY3Mjg2MjYxMQ==\",\"index\":true},{\"key\":\"Y29uc2lzdGVuY3lfbGV2ZWw=\",\"value\":\"MTU=\",\"index\":true},{\"key\":\"cGF5bG9hZA==\",\"value\":\"QVFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQTNndHJPblpBQUFBQUFBQUFBQUFBQUFBQUFBTFl2bXZ3dXFkT0NwQndGbWVjcnBHUTZBM1FvQUFnQUFBQUFBQUFBQUFBQUFBTUVJSUpnL00wVnM1NzZ6b0ViMXFEK2pUd0o5RENBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQUFBQT09\",\"index\":true}]}")
	event := tmAbci.Event{}
	err := json.Unmarshal(eventJson, &event)
	require.NoError(t, err)

	attrs := make(map[string]string)
	for _, attr := range event.Attributes {

		logger.Debug("acctwatcher: attribute", zap.String("key", string(attr.Key)), zap.String("value", string(attr.Value)))
		attrs[string(attr.Key)] = string(attr.Value)
	}

	attrBytes, err := json.Marshal(attrs)
	require.NoError(t, err)

	evt := new(WasmTransferAsStrings)
	err = json.Unmarshal(attrBytes, evt)
	require.NoError(t, err)
	assert.Equal(t, string("1672862611"), evt.Timestamp)
}
