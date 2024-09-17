package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"
)

// TestFormatIbcHooksMemo tests the FormatIbcHooksMemo function.
func TestFormatIbcHooksMemo(t *testing.T) {
	ibcTranslatorContract := "wormhole123abc"

	for _, tc := range []struct {
		testName  string
		payload   types.ParsedPayload
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
			shouldErr: false,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			res, err := types.FormatIbcHooksMemo(tc.payload, ibcTranslatorContract)

			if tc.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, res)

				// validate payload was formatted correctly
				if tc.payload.NoPayload {
					require.NotContains(t, res, "gateway_convert_and_transfer_with_payload")
					require.Contains(t, res, "recipient")
				} else {
					require.Contains(t, res, "gateway_convert_and_transfer_with_payload")
					require.NotContains(t, res, "recipient")
					require.Contains(t, res, "payload")
				}
			}
		})
	}
}
