package keeper_test

import (
	_ "embed"
	"encoding/json"
	"testing"

	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"
	wormholetypes "github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// TestPackets ensure that packets are handled correctly by the ibc composability middleware.
// This test will only be able to process IBC hooks messages and not PFM because the test
// requires a full chain setup with interchaintest.
func TestPackets(t *testing.T) {
	// setup app & get keepers & ctx
	app, ctx := keepertest.SetupWormchainAndContext(t)
	whKeeper := app.WormholeKeeper
	keeper := app.IbcComposabilityMwKeeper

	// set ibc composability contract
	whKeeper.StoreIbcComposabilityMwContract(ctx, wormholetypes.IbcComposabilityMwContract{
		ContractAddress: "wormhole1du4amsmvx8yqr8whw7qc5m3c0zpwknmzelwqy6",
	})

	// define a packet with no ibc token bridge payload
	packetDataNoPayload, err := json.Marshal(transfertypes.FungibleTokenPacketData{
		Denom:    "uworm",
		Amount:   "100",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     "",
	})
	require.NoError(t, err)

	// define gateway payload for packet
	gatewayTBPayload, err := json.Marshal(types.GatewayIbcTokenBridgePayload{
		GatewayIbcTokenBridgePayloadObj: types.GatewayIbcTokenBridgePayloadObj{
			Transfer: types.GatewayTransfer{
				Chain:     1,
				Recipient: []byte("recipient"),
				Fee:       "0uworm",
				Nonce:     1,
			},
		},
	})
	require.NoError(t, err)

	// define a packet with a valid ibc token bridge payload
	packetDataWithPayload, err := json.Marshal(transfertypes.FungibleTokenPacketData{
		Denom:    "uworm",
		Amount:   "100",
		Sender:   "sender",
		Receiver: "receiver",
		Memo:     string(gatewayTBPayload),
	})
	require.NoError(t, err)

	for _, tc := range []struct {
		testName  string
		packet    channeltypes.Packet
		shouldErr bool
	}{
		{
			testName:  "empty packet - expect error",
			shouldErr: true,
		},
		{
			testName: "packet with no data - expect error",
			packet: channeltypes.Packet{
				Data: []byte("wrong data format"),
			},
			shouldErr: true,
		},
		{
			testName: "packet with no memo in data - expect error",
			packet: channeltypes.Packet{
				Data: packetDataNoPayload,
			},
			shouldErr: true,
		},
		{
			testName: "packet with payload - expect success",
			packet: channeltypes.Packet{
				Sequence:           1,
				SourcePort:         "transfer",
				SourceChannel:      "channel-0",
				DestinationPort:    "transfer",
				DestinationChannel: "channel-0",
				Data:               packetDataWithPayload,
			},
			shouldErr: false,
		},
	} {
		packet, ack := keeper.OnRecvPacket(ctx, tc.packet)

		t.Run(tc.testName, func(t *testing.T) {
			if tc.shouldErr {
				require.NotNil(t, ack)
			} else {
				require.NotNil(t, packet)
				require.Nil(t, ack)

				// Should return nil because the packet is not transposed (it is an ibc hooks packet)
				res := keeper.GetAndClearTransposedData(ctx, tc.packet.DestinationChannel, tc.packet.DestinationPort, tc.packet.Sequence)
				require.Nil(t, res)
			}
		})
	}
}
