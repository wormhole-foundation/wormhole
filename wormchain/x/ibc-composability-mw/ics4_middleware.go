package ibc_composability_mw

import (
	// external libraries
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	// ibc-go
	clienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"

	"github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/keeper"
)

var _ porttypes.ICS4Wrapper = &ICS4Middleware{}

type ICS4Middleware struct {
	channel porttypes.ICS4Wrapper
	keeper  *keeper.Keeper
}

func NewICS4Middleware(channel porttypes.ICS4Wrapper, keeper *keeper.Keeper) ICS4Middleware {
	return ICS4Middleware{
		channel: channel,
		keeper:  keeper,
	}
}

func (i ICS4Middleware) SendPacket(ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	sourcePort string,
	sourceChannel string,
	timeoutHeight clienttypes.Height,
	timeoutTimestamp uint64,
	data []byte,
) (sequence uint64, err error) {
	return i.channel.SendPacket(ctx, chanCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
}

func (i ICS4Middleware) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	packet = i.keeper.WriteAcknowledgement(ctx, packet)
	err := i.channel.WriteAcknowledgement(ctx, chanCap, packet, ack)
	return err
}

func (i ICS4Middleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	version, err := i.channel.GetAppVersion(ctx, portID, channelID)
	return version, err
}
