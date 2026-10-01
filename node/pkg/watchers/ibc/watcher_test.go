package ibc

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"sync"
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
		`{ "key": "bWVzc2FnZS5jaGFpbl9pZA==", "value": "NDAwMA==", "index": true },` +
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
			TxID:           txHash.Bytes(),
			EmitterAddress: expectedSender,
			EmitterChain:   vaa.ChainIDCosmoshub,
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

// fakeLcd stands in for the wormchain LCD endpoint that serves the channel ID to chain ID mapping.
// It records how many times it was queried so that tests can assert on caching behavior.
type fakeLcd struct {
	mutex   sync.Mutex
	mapping map[string]vaa.ChainID
	failing bool
	queries int
}

func (f *fakeLcd) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.queries++

	if f.failing {
		// The watcher does not check the status code, but an empty result is rejected by the query parser.
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{}`))
		return
	}

	channelChains := make([][]interface{}, 0, len(f.mapping))
	for channelID, chainID := range f.mapping {
		channelChains = append(channelChains, []interface{}{
			base64.StdEncoding.EncodeToString([]byte(channelID)),
			uint16(chainID),
		})
	}

	err := json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{"channels_chains": channelChains},
	})
	if err != nil {
		panic(err)
	}
}

func (f *fakeLcd) setMapping(mapping map[string]vaa.ChainID) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.mapping = mapping
}

func (f *fakeLcd) setFailing(failing bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.failing = failing
}

func (f *fakeLcd) queryCount() int {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.queries
}

// newWatcherForChannelMapTest creates a watcher pointed at a fake LCD serving the given mapping.
func newWatcherForChannelMapTest(t *testing.T, mapping map[string]vaa.ChainID) (*Watcher, *fakeLcd) {
	t.Helper()

	lcd := &fakeLcd{mapping: mapping}
	server := httptest.NewServer(lcd)
	t.Cleanup(server.Close)

	return &Watcher{
		lcdUrl:                server.URL,
		contractAddress:       "wormhole1ibcreceivercontract",
		logger:                zap.NewNop(),
		channelIdToChainIdMap: make(map[string]vaa.ChainID),
	}, lcd
}

// expireChannelIdToChainIdMap ages the cached mapping out so that the next lookup refreshes it.
func expireChannelIdToChainIdMap(w *Watcher) {
	w.channelIdToChainIdMapTime = time.Now().Add(-2 * channelIdToChainIdMapMaxAge)
}

// ageChannelIdToChainIdMapPastMinQueryInterval ages the cached mapping past the throttle that applies to unknown
// channels, but leaves it well within the max age, so only a lookup of an unknown channel will refresh it.
func ageChannelIdToChainIdMapPastMinQueryInterval(w *Watcher) {
	w.channelIdToChainIdMapTime = time.Now().Add(-2 * channelIdToChainIdMapMinQueryInterval)
}

func TestGetChainIdFromChannelIDCachesWithinMaxAge(t *testing.T) {
	w, lcd := newWatcherForChannelMapTest(t, map[string]vaa.ChainID{"channel-0": vaa.ChainIDOsmosis})

	chainID, err := w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDOsmosis, chainID)
	assert.Equal(t, 1, lcd.queryCount())

	// Subsequent lookups within the max age are served from the cache.
	for i := 0; i < 5; i++ {
		chainID, err = w.getChainIdFromChannelID("channel-0")
		require.NoError(t, err)
		assert.Equal(t, vaa.ChainIDOsmosis, chainID)
	}
	assert.Equal(t, 1, lcd.queryCount())
}

// TestGetChainIdFromChannelIDPicksUpGovernanceUpdate verifies that a governance update to an already cached
// channel is picked up once the cached mapping goes stale, without requiring a restart.
func TestGetChainIdFromChannelIDPicksUpGovernanceUpdate(t *testing.T) {
	w, lcd := newWatcherForChannelMapTest(t, map[string]vaa.ChainID{"channel-0": vaa.ChainIDOsmosis})

	chainID, err := w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDOsmosis, chainID)

	// Governance overwrites the mapping for an existing channel.
	lcd.setMapping(map[string]vaa.ChainID{"channel-0": vaa.ChainIDCosmoshub})

	// The old value is still returned until the cached mapping goes stale.
	chainID, err = w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDOsmosis, chainID)
	assert.Equal(t, 1, lcd.queryCount())

	expireChannelIdToChainIdMap(w)

	chainID, err = w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDCosmoshub, chainID)
	assert.Equal(t, 2, lcd.queryCount())
}

// TestGetChainIdFromChannelIDPicksUpRevocation verifies that revoking a channel takes effect on a running
// watcher. Revocation is done by overwriting the entry with the unset chain ID, since there is no delete action.
func TestGetChainIdFromChannelIDPicksUpRevocation(t *testing.T) {
	w, lcd := newWatcherForChannelMapTest(t, map[string]vaa.ChainID{
		"channel-0": vaa.ChainIDOsmosis,
		"channel-1": vaa.ChainIDCosmoshub,
	})

	chainID, err := w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDOsmosis, chainID)

	lcd.setMapping(map[string]vaa.ChainID{
		"channel-0": vaa.ChainIDUnset,
		"channel-1": vaa.ChainIDCosmoshub,
	})
	expireChannelIdToChainIdMap(w)

	// The caller treats the unset chain ID as an unknown channel and drops the observation.
	chainID, err = w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDUnset, chainID)

	// The other channel is unaffected.
	chainID, err = w.getChainIdFromChannelID("channel-1")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDCosmoshub, chainID)
}

// TestGetChainIdFromChannelIDPicksUpNewChannel verifies that a channel registered by governance after the last
// refresh is picked up without waiting for the full max age, so valid messages on it are not dropped.
func TestGetChainIdFromChannelIDPicksUpNewChannel(t *testing.T) {
	w, lcd := newWatcherForChannelMapTest(t, map[string]vaa.ChainID{"channel-0": vaa.ChainIDOsmosis})

	_, err := w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	require.Equal(t, 1, lcd.queryCount())

	// Governance registers a new channel just after the mapping was refreshed.
	lcd.setMapping(map[string]vaa.ChainID{
		"channel-0": vaa.ChainIDOsmosis,
		"channel-1": vaa.ChainIDCosmoshub,
	})

	// The first event arrives immediately, so the query is still throttled.
	chainID, err := w.getChainIdFromChannelID("channel-1")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDUnset, chainID)
	assert.Equal(t, 1, lcd.queryCount())

	// Once the throttle has elapsed the new channel is picked up, well before the max age.
	ageChannelIdToChainIdMapPastMinQueryInterval(w)

	chainID, err = w.getChainIdFromChannelID("channel-1")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDCosmoshub, chainID)
	assert.Equal(t, 2, lcd.queryCount())
}

// TestGetChainIdFromChannelIDThrottlesQueriesForUnknownChannel verifies that an unregistered channel cannot drive
// a query on every event, since anyone can establish a channel to the contract and emit events over it.
func TestGetChainIdFromChannelIDThrottlesQueriesForUnknownChannel(t *testing.T) {
	w, lcd := newWatcherForChannelMapTest(t, map[string]vaa.ChainID{"channel-0": vaa.ChainIDOsmosis})

	_, err := w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	require.Equal(t, 1, lcd.queryCount())

	for i := 0; i < 5; i++ {
		chainID, err := w.getChainIdFromChannelID("channel-4242")
		require.NoError(t, err)
		assert.Equal(t, vaa.ChainIDUnset, chainID)
	}
	assert.Equal(t, 1, lcd.queryCount(), "unknown channel should not query on every event")

	// After the throttle elapses the mapping is requeried once, then throttled again.
	ageChannelIdToChainIdMapPastMinQueryInterval(w)

	for i := 0; i < 5; i++ {
		chainID, err := w.getChainIdFromChannelID("channel-4242")
		require.NoError(t, err)
		assert.Equal(t, vaa.ChainIDUnset, chainID)
	}
	assert.Equal(t, 2, lcd.queryCount(), "unknown channel should trigger at most one query per interval")
}

// TestGetChainIdFromChannelIDDoesNotThrottleKnownChannel verifies that lookups of a cached channel are served from
// the cache and never trigger a query while the mapping is within its max age.
func TestGetChainIdFromChannelIDDoesNotThrottleKnownChannel(t *testing.T) {
	w, lcd := newWatcherForChannelMapTest(t, map[string]vaa.ChainID{"channel-0": vaa.ChainIDOsmosis})

	_, err := w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	require.Equal(t, 1, lcd.queryCount())

	ageChannelIdToChainIdMapPastMinQueryInterval(w)

	for i := 0; i < 5; i++ {
		chainID, err := w.getChainIdFromChannelID("channel-0")
		require.NoError(t, err)
		assert.Equal(t, vaa.ChainIDOsmosis, chainID)
	}
	assert.Equal(t, 1, lcd.queryCount())
}

// TestGetChainIdFromChannelIDUsesCachedMapOnQueryFailure verifies that a failed refresh keeps serving the
// previously cached mapping, and that the failing query is retried at most once per interval.
func TestGetChainIdFromChannelIDUsesCachedMapOnQueryFailure(t *testing.T) {
	w, lcd := newWatcherForChannelMapTest(t, map[string]vaa.ChainID{"channel-0": vaa.ChainIDOsmosis})

	chainID, err := w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	require.Equal(t, vaa.ChainIDOsmosis, chainID)
	require.Equal(t, 1, lcd.queryCount())

	lcd.setFailing(true)
	expireChannelIdToChainIdMap(w)

	chainID, err = w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDOsmosis, chainID)
	assert.Equal(t, 2, lcd.queryCount())

	// The failed query should not be retried on every event.
	for i := 0; i < 5; i++ {
		chainID, err = w.getChainIdFromChannelID("channel-0")
		require.NoError(t, err)
		assert.Equal(t, vaa.ChainIDOsmosis, chainID)
	}
	assert.Equal(t, 2, lcd.queryCount())
}

// TestGetChainIdFromChannelIDFailsWhenNothingCached verifies that a query failure before anything has been
// cached returns an error rather than silently dropping observations as unknown channels.
func TestGetChainIdFromChannelIDFailsWhenNothingCached(t *testing.T) {
	w, lcd := newWatcherForChannelMapTest(t, map[string]vaa.ChainID{"channel-0": vaa.ChainIDOsmosis})
	lcd.setFailing(true)

	chainID, err := w.getChainIdFromChannelID("channel-0")
	require.Error(t, err)
	assert.Equal(t, vaa.ChainIDUnset, chainID)

	// It should keep retrying rather than caching the failure.
	_, err = w.getChainIdFromChannelID("channel-0")
	require.Error(t, err)
	assert.Equal(t, 2, lcd.queryCount())

	// Once the query recovers, the mapping is picked up.
	lcd.setFailing(false)
	chainID, err = w.getChainIdFromChannelID("channel-0")
	require.NoError(t, err)
	assert.Equal(t, vaa.ChainIDOsmosis, chainID)
}
