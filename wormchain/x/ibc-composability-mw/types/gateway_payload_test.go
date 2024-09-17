package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"
)

// TestGatewayPayloads tests the VerifyAndParseGatewayPayload function.
func TestGatewayPayloads(t *testing.T) {
	for _, tc := range []struct {
		testName  string
		memo      string // use memo if present, otherwise marshal tbPayload with json
		tbPayload types.GatewayIbcTokenBridgePayload
		shouldErr bool
	}{
		{
			testName:  "memo present, payload abscent - should error",
			memo:      "abc123",
			shouldErr: true,
		},
		{
			testName:  "memo abscent, invalid payload - should error",
			tbPayload: types.GatewayIbcTokenBridgePayload{},
			shouldErr: true,
		},
		{
			testName: "valid transfer no payload - should pass",
			tbPayload: types.GatewayIbcTokenBridgePayload{
				GatewayIbcTokenBridgePayloadObj: types.GatewayIbcTokenBridgePayloadObj{
					Transfer: types.GatewayTransfer{
						Chain:     1,
						Recipient: []byte("recipient"),
						Fee:       "0uworm",
						Nonce:     1,
					},
				},
			},
			shouldErr: false,
		},
		{
			testName: "valid transfer with payload - should pass",
			tbPayload: types.GatewayIbcTokenBridgePayload{
				GatewayIbcTokenBridgePayloadObj: types.GatewayIbcTokenBridgePayloadObj{
					TransferWithPayload: types.GatewayTransferWithPayload{
						Chain:    1,
						Contract: []byte("contract"),
						Payload:  []byte("{\"payload\":\"data\"}"),
						Nonce:    1,
					},
				},
			},
			shouldErr: false,
		},
	} {
		t.Run(tc.testName, func(t *testing.T) {
			memo := tc.memo

			if memo == "" {
				bz, err := json.Marshal(tc.tbPayload)
				require.NoError(t, err)
				memo = string(bz)
			}

			payload, err := types.VerifyAndParseGatewayPayload(memo)

			if tc.shouldErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)

				// validate payload was parsed correctly
				if payload.NoPayload {
					require.Equal(t, tc.tbPayload.GatewayIbcTokenBridgePayloadObj.Transfer.Chain, payload.ChainId)
					require.Equal(t, tc.tbPayload.GatewayIbcTokenBridgePayloadObj.Transfer.Recipient, payload.Recipient)
					require.Equal(t, tc.tbPayload.GatewayIbcTokenBridgePayloadObj.Transfer.Fee, payload.Fee)
					require.Equal(t, tc.tbPayload.GatewayIbcTokenBridgePayloadObj.Transfer.Nonce, payload.Nonce)
				} else {
					require.Equal(t, tc.tbPayload.GatewayIbcTokenBridgePayloadObj.TransferWithPayload.Chain, payload.ChainId)
					require.Equal(t, tc.tbPayload.GatewayIbcTokenBridgePayloadObj.TransferWithPayload.Contract, payload.Recipient)
					require.Equal(t, tc.tbPayload.GatewayIbcTokenBridgePayloadObj.TransferWithPayload.Payload, payload.Payload)
					require.Equal(t, tc.tbPayload.GatewayIbcTokenBridgePayloadObj.TransferWithPayload.Nonce, payload.Nonce)
				}
			}
		})
	}
}
