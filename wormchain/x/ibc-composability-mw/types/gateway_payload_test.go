package types_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"
)

func TestGatewayPayloads(t *testing.T) {

	for _, tc := range []struct {
		memo      string // use memo if present, otherwise marshal tbPayload with json
		tbPayload types.GatewayIbcTokenBridgePayload
		shouldErr bool
	}{
		{
			memo:      "abc123",
			shouldErr: true,
		},
		{
			tbPayload: types.GatewayIbcTokenBridgePayload{},
			shouldErr: true,
		},
		{
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

		memo := tc.memo

		if memo == "" {
			bz, err := json.Marshal(tc.tbPayload)
			require.NoError(t, err)
			memo = string(bz)
		}

		payload, err := types.VerifyAndParseGatewayPayload(memo)

		if tc.shouldErr {
			require.Error(t, err)
			// continue to next case if err
			continue
		} else {
			require.NoError(t, err)
		}

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
}
