package types_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"
)

// TestFormatPfmMemo tests the FormatPfmMemo function.
func TestFormatPfmMemo(t *testing.T) {
	for _, tc := range []struct {
		testName  string
		payload   types.ParsedPayload
		queryResp types.IbcTranslatorQueryRsp
		timeout   time.Duration
		retries   uint8
		shouldErr bool
	}{
		{
			testName: "Normal w/o payload - should pass",
			payload: types.ParsedPayload{
				NoPayload: true,
				ChainId:   1,
				Recipient: []byte{'a', 'b', 'c'},
				Fee:       "0uworm",
				Nonce:     1,
				Payload:   nil,
			},
			queryResp: types.IbcTranslatorQueryRsp{
				Channel: "channel",
			},
			timeout:   time.Hour,
			retries:   3,
			shouldErr: false,
		},
		{
			testName: "Provide payload when unnecessary - should pass",
			payload: types.ParsedPayload{
				NoPayload: true,
				ChainId:   1,
				Recipient: []byte{'a', 'b', 'c'},
				Fee:       "0uworm",
				Nonce:     1,
				Payload:   []byte("{\"payload\":\"data\"}"),
			},
			queryResp: types.IbcTranslatorQueryRsp{
				Channel: "channel",
			},
			timeout:   time.Hour,
			retries:   3,
			shouldErr: false,
		},
		{
			testName: "Normal w/ payload - should pass",
			payload: types.ParsedPayload{
				NoPayload: false,
				ChainId:   1,
				Recipient: []byte{'a', 'b', 'c'},
				Fee:       "0uworm",
				Nonce:     1,
				Payload:   []byte("{\"payload\":\"data\"}"),
			},
			queryResp: types.IbcTranslatorQueryRsp{
				Channel: "channel-34",
			},
			timeout:   time.Minute,
			retries:   21,
			shouldErr: false,
		},
		{
			testName: "Nil payload - should pass",
			payload: types.ParsedPayload{
				NoPayload: true,
				ChainId:   1,
				Recipient: []byte{'a', 'b', 'c'},
				Fee:       "0uworm",
				Nonce:     1,
				Payload:   nil,
			},
			queryResp: types.IbcTranslatorQueryRsp{
				Channel: "channel",
			},
			timeout:   time.Microsecond,
			retries:   255,
			shouldErr: false,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			// turn the query response into bytes
			queryRespBz, err := json.Marshal(tc.queryResp)
			require.NoError(t, err)

			res, err := types.FormatPfmMemo(tc.payload, queryRespBz, tc.timeout, tc.retries)

			if tc.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)

				// convert response back to packet metadata
				var packetMetadata types.PacketMetadata
				err = json.Unmarshal([]byte(res), &packetMetadata)
				require.NoError(t, err)

				// validation checks
				require.Equal(t, string(tc.payload.Recipient), packetMetadata.Forward.Receiver)
				require.Equal(t, "transfer", packetMetadata.Forward.Port)
				require.Equal(t, tc.queryResp.Channel, packetMetadata.Forward.Channel)
				require.Equal(t, tc.timeout, packetMetadata.Forward.Timeout)
				require.Equal(t, &tc.retries, packetMetadata.Forward.Retries)
			}
		})
	}
}
