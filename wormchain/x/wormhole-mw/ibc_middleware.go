package wormhole_mw

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/wormhole-foundation/wormchain/x/wormhole-mw/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	transfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
)

var _ porttypes.Middleware = &IBCMiddleware{}

// IBCMiddleware implements the ICS26 callbacks for the wormhole middleware given the
// forward keeper and the underlying application.
type IBCMiddleware struct {
	app    porttypes.IBCModule
	wasmKeeper *wasmkeeper.PermissionedKeeper

	retriesOnTimeout uint8
	forwardTimeout   time.Duration
	refundTimeout    time.Duration
}

func (im *IBCMiddleware) SetWasmKeeper(keeper *wasmkeeper.PermissionedKeeper) {
	im.wasmKeeper = keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application.
func NewIBCMiddleware(
	app porttypes.IBCModule,
	k *wasmkeeper.PermissionedKeeper,
	retriesOnTimeout uint8,
	forwardTimeout time.Duration,
	refundTimeout time.Duration,
) IBCMiddleware {
	return IBCMiddleware{
		app:              app,
		wasmKeeper:       k,
		retriesOnTimeout: retriesOnTimeout,
		forwardTimeout:   forwardTimeout,
		refundTimeout:    refundTimeout,
	}
}

// OnChanOpenInit implements the IBCModule interface.
func (im IBCMiddleware) OnChanOpenInit(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID string,
	channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	version string,
) (string, error) {
	return im.app.OnChanOpenInit(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, version)
}

// OnChanOpenTry implements the IBCModule interface.
func (im IBCMiddleware) OnChanOpenTry(
	ctx sdk.Context,
	order channeltypes.Order,
	connectionHops []string,
	portID, channelID string,
	chanCap *capabilitytypes.Capability,
	counterparty channeltypes.Counterparty,
	counterpartyVersion string,
) (version string, err error) {
	return im.app.OnChanOpenTry(ctx, order, connectionHops, portID, channelID, chanCap, counterparty, counterpartyVersion)
}

// OnChanOpenAck implements the IBCModule interface.
func (im IBCMiddleware) OnChanOpenAck(
	ctx sdk.Context,
	portID, channelID string,
	counterpartyChannelID string,
	counterpartyVersion string,
) error {
	return im.app.OnChanOpenAck(ctx, portID, channelID, counterpartyChannelID, counterpartyVersion)
}

// OnChanOpenConfirm implements the IBCModule interface.
func (im IBCMiddleware) OnChanOpenConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanOpenConfirm(ctx, portID, channelID)
}

// OnChanCloseInit implements the IBCModule interface.
func (im IBCMiddleware) OnChanCloseInit(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseInit(ctx, portID, channelID)
}

// OnChanCloseConfirm implements the IBCModule interface.
func (im IBCMiddleware) OnChanCloseConfirm(ctx sdk.Context, portID, channelID string) error {
	return im.app.OnChanCloseConfirm(ctx, portID, channelID)
}

// OnRecvPacket checks the memo field on this packet and if the memo indicates this packet
// should be handled by the wormhole middleware, it updates the memo according to the payload
func (im IBCMiddleware) OnRecvPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	relayer sdk.AccAddress,
) ibcexported.Acknowledgement {
	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	d := make(map[string]interface{})
	err := json.Unmarshal([]byte(data.Memo), &d)
	if err != nil || d["gateway_ibc_token_bridge_payload"] == nil {
		// not a packet that should be forwarded
		return im.app.OnRecvPacket(ctx, packet, relayer)
	}
	m := &types.GatewayIbcTokenBridgePayload{}
	err = json.Unmarshal([]byte(data.Memo), m)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(fmt.Errorf("wormhole-mw: error parsing gateway ibc token bridge payload, %s", err))
	}

	// Look up chain id's channel
	// If doesn't exist, create ibc-hooks memo
	// If exists, create PFM memo
	// Check the payload type
	// If simple
	// If CC

	return nil
}

// OnAcknowledgementPacket implements the IBCModule interface.
func (im IBCMiddleware) OnAcknowledgementPacket(
	ctx sdk.Context,
	packet channeltypes.Packet,
	acknowledgement []byte,
	relayer sdk.AccAddress,
) error {
	return im.app.OnAcknowledgementPacket(ctx, packet, acknowledgement, relayer)
}

// OnTimeoutPacket implements the IBCModule interface.
func (im IBCMiddleware) OnTimeoutPacket(ctx sdk.Context, packet channeltypes.Packet, relayer sdk.AccAddress) error {
	return im.app.OnTimeoutPacket(ctx, packet, relayer)
}

// SendPacket implements the ICS4 Wrapper interface.
func (im IBCMiddleware) SendPacket(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
) error {
	panic("Wormhole-mw should not be wired for SendPacket")
}

// WriteAcknowledgement implements the ICS4 Wrapper interface.
func (im IBCMiddleware) WriteAcknowledgement(
	ctx sdk.Context,
	chanCap *capabilitytypes.Capability,
	packet ibcexported.PacketI,
	ack ibcexported.Acknowledgement,
) error {
	panic("Wormhole-mw should not be wired for WriteAcknowledgement")
}

func (im IBCMiddleware) GetAppVersion(ctx sdk.Context, portID string, channelID string) (string, bool) {
	panic("Wormhole-mw should not be wired for ICS4")
}
