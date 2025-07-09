package ibc_hooks

import (
	// external libraries
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	// ibc-go
	porttypes "github.com/cosmos/ibc-go/v7/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
)

var _ porttypes.ICS4Wrapper = &ICS4Middleware{}

type ICS4Middleware struct {
	channel porttypes.ICS4Wrapper

	// Hooks
	Hooks Hooks
}

func NewICS4Middleware(channel porttypes.ICS4Wrapper, hooks Hooks) ICS4Middleware {
	return ICS4Middleware{
		channel: channel,
		Hooks:   hooks,
	}
}

func (i ICS4Middleware) SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, sourcePort string, sourceChannel string, timeoutHeight ibcclienttypes.Height, timeoutTimestamp uint64, data []byte) (sequence uint64, err error) {
	if hook, ok := i.Hooks.(SendPacketOverrideHooks); ok {
		return hook.SendPacketOverride(i, ctx, channelCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}

	if hook, ok := i.Hooks.(SendPacketBeforeHooks); ok {
		hook.SendPacketBeforeHook(ctx, channelCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)
	}

	seq, err := i.channel.SendPacket(ctx, channelCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data)

	if hook, ok := i.Hooks.(SendPacketAfterHooks); ok {
		hook.SendPacketAfterHook(ctx, channelCap, sourcePort, sourceChannel, timeoutHeight, timeoutTimestamp, data, err)
	}

	return seq, err
}

func (i ICS4Middleware) WriteAcknowledgement(ctx sdk.Context, chanCap *capabilitytypes.Capability, packet ibcexported.PacketI, ack ibcexported.Acknowledgement) error {
	if hook, ok := i.Hooks.(WriteAcknowledgementOverrideHooks); ok {
		return hook.WriteAcknowledgementOverride(i, ctx, chanCap, packet, ack)
	}

	if hook, ok := i.Hooks.(WriteAcknowledgementBeforeHooks); ok {
		hook.WriteAcknowledgementBeforeHook(ctx, chanCap, packet, ack)
	}
	err := i.channel.WriteAcknowledgement(ctx, chanCap, packet, ack)
	if hook, ok := i.Hooks.(WriteAcknowledgementAfterHooks); ok {
		hook.WriteAcknowledgementAfterHook(ctx, chanCap, packet, ack, err)
	}

	return err
}

func (i ICS4Middleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	if hook, ok := i.Hooks.(GetAppVersionOverrideHooks); ok {
		return hook.GetAppVersionOverride(i, ctx, portID, channelID)
	}

	if hook, ok := i.Hooks.(GetAppVersionBeforeHooks); ok {
		hook.GetAppVersionBeforeHook(ctx, portID, channelID)
	}
	version, err := i.channel.GetAppVersion(ctx, portID, channelID)
	if hook, ok := i.Hooks.(GetAppVersionAfterHooks); ok {
		hook.GetAppVersionAfterHook(ctx, portID, channelID, version, err)
	}

	return version, err
}
