package ibc_hooks

import (
	// external libraries
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	// ibc-go
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"
)

var _ porttypes.Middleware = &IBCMiddleware{}

type IBCMiddleware struct {
	App            porttypes.IBCModule
	ICS4Middleware *ICS4Middleware
}

func NewIBCMiddleware(app porttypes.IBCModule, ics4 *ICS4Middleware) IBCMiddleware {
	return IBCMiddleware{
		App:            app,
		ICS4Middleware: ics4,
	}
}

// OnChanOpenInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenInitOverrideHooks); ok {
		return hook.OnChanOpenInitOverride(im, ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)
	}

	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenInitBeforeHooks); ok {
		hook.OnChanOpenInitBeforeHook(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)
	}

	finalVersion, err := im.App.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version)

	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenInitAfterHooks); ok {
		hook.OnChanOpenInitAfterHook(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, version, finalVersion, err)
	}
	return version, err
}

// OnChanOpenTry implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID,
	channelID string,
	channelCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (string, error) {
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenTryOverrideHooks); ok {
		return hook.OnChanOpenTryOverride(im, ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)
	}

	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenTryBeforeHooks); ok {
		hook.OnChanOpenTryBeforeHook(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)
	}

	version, err := im.App.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion)

	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenTryAfterHooks); ok {
		hook.OnChanOpenTryAfterHook(ctx, order, connectionHops, portID, channelID, channelCap, counterparty, counterpartyVersion, version, err)
	}
	return version, err
}

// OnChanOpenAck implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID,
	channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenAckOverrideHooks); ok {
		return hook.OnChanOpenAckOverride(im, ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
	}

	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenAckBeforeHooks); ok {
		hook.OnChanOpenAckBeforeHook(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
	}
	err := im.App.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenAckAfterHooks); ok {
		hook.OnChanOpenAckAfterHook(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion, err)
	}

	return err
}

// OnChanOpenConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanOpenConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenConfirmOverrideHooks); ok {
		return hook.OnChanOpenConfirmOverride(im, ctx, portID, channelID)
	}

	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenConfirmBeforeHooks); ok {
		hook.OnChanOpenConfirmBeforeHook(ctx, portID, channelID)
	}
	err := im.App.OnChanOpenConfirm(ctx, portID, channelID)
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanOpenConfirmAfterHooks); ok {
		hook.OnChanOpenConfirmAfterHook(ctx, portID, channelID, err)
	}
	return err
}

// OnChanCloseInit implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseInit(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Here we can remove the limits when a new channel is closed. For now, they can remove them  manually on the contract
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanCloseInitOverrideHooks); ok {
		return hook.OnChanCloseInitOverride(im, ctx, portID, channelID)
	}

	if hook, ok := im.ICS4Middleware.Hooks.(OnChanCloseInitBeforeHooks); ok {
		hook.OnChanCloseInitBeforeHook(ctx, portID, channelID)
	}
	err := im.App.OnChanCloseInit(ctx, portID, channelID)
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanCloseInitAfterHooks); ok {
		hook.OnChanCloseInitAfterHook(ctx, portID, channelID, err)
	}

	return err
}

// OnChanCloseConfirm implements the IBCMiddleware interface
func (im IBCMiddleware) OnChanCloseConfirm(
	ctx sdk.Context,
	portID,
	channelID string,
) error {
	// Here we can remove the limits when a new channel is closed. For now, they can remove them  manually on the contract
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanCloseConfirmOverrideHooks); ok {
		return hook.OnChanCloseConfirmOverride(im, ctx, portID, channelID)
	}

	if hook, ok := im.ICS4Middleware.Hooks.(OnChanCloseConfirmBeforeHooks); ok {
		hook.OnChanCloseConfirmBeforeHook(ctx, portID, channelID)
	}
	err := im.App.OnChanCloseConfirm(ctx, portID, channelID)
	if hook, ok := im.ICS4Middleware.Hooks.(OnChanCloseConfirmAfterHooks); ok {
		hook.OnChanCloseConfirmAfterHook(ctx, portID, channelID, err)
	}

	return err
}

// OnRecvPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	if hook, ok := im.ICS4Middleware.Hooks.(OnRecvPacketOverrideHooks); ok {
		return hook.OnRecvPacketOverride(im, ctx, packet, relayer)
	}

	if hook, ok := im.ICS4Middleware.Hooks.(OnRecvPacketBeforeHooks); ok {
		hook.OnRecvPacketBeforeHook(ctx, packet, relayer)
	}

	ack := im.App.OnRecvPacket(ctx, packet, relayer)

	if hook, ok := im.ICS4Middleware.Hooks.(OnRecvPacketAfterHooks); ok {
		hook.OnRecvPacketAfterHook(ctx, packet, relayer, ack)
	}

	return ack
}

// OnAcknowledgementPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	if hook, ok := im.ICS4Middleware.Hooks.(OnAcknowledgementPacketOverrideHooks); ok {
		return hook.OnAcknowledgementPacketOverride(im, ctx, packet, acknowledgement, relayer)
	}
	if hook, ok := im.ICS4Middleware.Hooks.(OnAcknowledgementPacketBeforeHooks); ok {
		hook.OnAcknowledgementPacketBeforeHook(ctx, packet, acknowledgement, relayer)
	}

	err := im.App.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)

	if hook, ok := im.ICS4Middleware.Hooks.(OnAcknowledgementPacketAfterHooks); ok {
		hook.OnAcknowledgementPacketAfterHook(ctx, packet, acknowledgement, relayer, err)
	}

	return err
}

// OnTimeoutPacket implements the IBCMiddleware interface
func (im IBCMiddleware) OnTimeoutPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) error {
	if hook, ok := im.ICS4Middleware.Hooks.(OnTimeoutPacketOverrideHooks); ok {
		return hook.OnTimeoutPacketOverride(im, ctx, packet, relayer)
	}

	if hook, ok := im.ICS4Middleware.Hooks.(OnTimeoutPacketBeforeHooks); ok {
		hook.OnTimeoutPacketBeforeHook(ctx, packet, relayer)
	}
	err := im.App.OnTimeoutPacket(ctx, packet, relayer)
	if hook, ok := im.ICS4Middleware.Hooks.(OnTimeoutPacketAfterHooks); ok {
		hook.OnTimeoutPacketAfterHook(ctx, packet, relayer, err)
	}

	return err
}

// SendPacket implements the ICS4 Wrapper interface
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
) error {
	return im.ICS4Middleware.SendPacket(ctx, chanCap, packet)
}

// WriteAcknowledgement implements the ICS4 Wrapper interface
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	return im.ICS4Middleware.WriteAcknowledgement(ctx, chanCap, packet, ack)
}

func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID, channelID string) (string, bool) {
	return im.ICS4Middleware.GetAppVersion(ctx, portID, channelID)
}
