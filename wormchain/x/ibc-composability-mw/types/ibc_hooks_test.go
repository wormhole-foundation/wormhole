package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"
)

func TestFormatIbcHooksMemo(t *testing.T) {

	ibcTranslatorContract := "wormhole123abc"

	for _, tc := range []struct {
		payload   types.ParsedPayload
		shouldErr bool
	}{
		// Normal w/ no payload
		{
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
		// Provide payload when unnecessary
		{
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
		// Normal w/ payload
		{
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
		// Nil payload should not err
		{
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
		res, err := types.FormatIbcHooksMemo(tc.payload, ibcTranslatorContract)

		if tc.shouldErr {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
			require.NotNil(t, res)
		}

		if tc.payload.NoPayload {
			require.NotContains(t, res, "gateway_convert_and_transfer_with_payload")
			require.Contains(t, res, "recipient")
		} else {
			require.Contains(t, res, "gateway_convert_and_transfer_with_payload")
			require.NotContains(t, res, "recipient")
			require.Contains(t, res, "payload")
		}
	}
}
