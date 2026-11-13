package near

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"go.uber.org/zap/zaptest"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
)

const (
	testEmitterHex  = "148410499d3fcda4dcfd68a1ebfcdddda16ab28326448d4aae4d2f0465cdfcb7"
	testBlockHeight = uint64(76538233)
	testBlockTime   = uint64(1666142881)
	testSeq         = uint64(261)
)

// validPublishEvent returns a NEP-297 Wormhole publish event that processWormholeLog accepts.
// Each test case changes one field of it to check one validation rule.
func validPublishEvent() map[string]any {
	return map[string]any{
		"standard": "wormhole",
		"event":    "publish",
		"data":     "deadbeef",
		"nonce":    1,
		"emitter":  testEmitterHex,
		"seq":      testSeq,
		"block":    testBlockHeight,
	}
}

func eventLog(t *testing.T, fields map[string]any) string {
	t.Helper()
	b, err := json.Marshal(fields)
	require.NoError(t, err)
	return "EVENT_JSON:" + string(b)
}

func successValueFor(seq uint64) string {
	return base64.StdEncoding.EncodeToString([]byte(strconv.FormatUint(seq, 10)))
}

// TestProcessWormholeLog checks the validation rules in processWormholeLog.
// A rule that is removed or weakened lets a bad log become a MessagePublication.
func TestProcessWormholeLog(t *testing.T) {
	validTxHash := base58.Encode([]byte("_____________________________TX1"))
	validHeader := nearapi.BlockHeader{Height: testBlockHeight, Timestamp: testBlockTime}

	with := func(key string, value any) string {
		ev := validPublishEvent()
		ev[key] = value
		return eventLog(t, ev)
	}
	without := func(key string) string {
		ev := validPublishEvent()
		delete(ev, key)
		return eventLog(t, ev)
	}

	tests := []struct {
		name         string
		log          string
		successValue string
		txHash       string
		header       nearapi.BlockHeader
		wantErr      bool
		wantPublish  bool
	}{
		{name: "valid event is published", log: eventLog(t, validPublishEvent()), wantPublish: true},

		// Logs that are not Wormhole publish events are skipped without an error.
		{name: "log without EVENT_JSON prefix", log: strings.TrimPrefix(eventLog(t, validPublishEvent()), "EVENT_JSON:")},
		{name: "invalid JSON", log: "EVENT_JSON:{not json"},
		{name: "other standard", log: with("standard", "nep141")},
		{name: "other event type", log: with("event", "ft_transfer")},
		{name: "missing standard", log: without("standard")},

		// Wormhole publish events that fail a check are rejected with an error.
		{name: "wrong JSON type for seq", log: with("seq", "261"), wantErr: true},
		{name: "empty emitter", log: with("emitter", ""), wantErr: true},
		{name: "zero seq", log: with("seq", 0), successValue: successValueFor(0), wantErr: true},
		{name: "zero block", log: with("block", 0), wantErr: true},
		{name: "SuccessValue does not match seq", successValue: successValueFor(testSeq + 1), wantErr: true},
		{name: "SuccessValue is not base64", successValue: "?", wantErr: true},
		{name: "SuccessValue is not a number", successValue: base64.StdEncoding.EncodeToString([]byte("abc")), wantErr: true},
		{name: "event block does not match receipt block", log: with("block", testBlockHeight+1), wantErr: true},
		{name: "emitter is not hex", log: with("emitter", strings.Repeat("zz", 32)), wantErr: true},
		{name: "emitter is too short", log: with("emitter", testEmitterHex[:62]), wantErr: true},
		{name: "emitter is too long", log: with("emitter", testEmitterHex+"00"), wantErr: true},
		{name: "tx hash is not base58", txHash: "0OIl", wantErr: true},
		{name: "tx hash is not 32 bytes", txHash: base58.Encode([]byte("short")), wantErr: true},
		{name: "data is not hex", log: with("data", "xyz0"), wantErr: true},
		{name: "data has odd length", log: with("data", "abc"), wantErr: true},
		{name: "block timestamp out of range", header: nearapi.BlockHeader{Height: testBlockHeight, Timestamp: 1 << 33}, wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.log == "" {
				tc.log = eventLog(t, validPublishEvent())
			}
			if tc.successValue == "" {
				tc.successValue = successValueFor(testSeq)
			}
			if tc.txHash == "" {
				tc.txHash = validTxHash
			}
			if tc.header == (nearapi.BlockHeader{}) {
				tc.header = validHeader
			}

			msgC := make(chan *common.MessagePublication, 1)
			w := NewWatcher("", WORMHOLE_CONTRACT, msgC, nil, false)
			job := newTransactionProcessingJob(tc.txHash, "sender.near", false)

			err := w.processWormholeLog(zaptest.NewLogger(t), context.Background(), job, tc.header, tc.successValue, gjson.Parse(strconvQuote(tc.log)))

			if tc.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.Equal(t, tc.wantPublish, job.hasWormholeMsg)

			if !tc.wantPublish {
				require.Empty(t, msgC, "rejected or skipped log must not produce a MessagePublication")
				return
			}

			require.Len(t, msgC, 1)
			msg := <-msgC
			require.Equal(t, testSeq, msg.Sequence)
			require.Equal(t, uint32(1), msg.Nonce)
			require.Equal(t, testEmitterHex, msg.EmitterAddress.String())
			require.Equal(t, []byte{0xde, 0xad, 0xbe, 0xef}, msg.Payload)
			require.Equal(t, []byte("_____________________________TX1"), msg.TxID)
			require.Equal(t, int64(testBlockTime), msg.Timestamp.Unix())
		})
	}
}

// strconvQuote wraps a log line as a JSON string, as it appears in receipts_outcome.outcome.logs.
func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
