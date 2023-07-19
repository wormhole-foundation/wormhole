package wormhole_mw

import (
	"encoding/json"
	"time"

	"github.com/wormhole-foundation/wormchain/x/wormhole-mw/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"

	transfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	porttypes "github.com/cosmos/ibc-go/v4/modules/core/05-port/types"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wormholekeeper "github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
)

var _ porttypes.Middleware = &IBCMiddleware{}

// IBCMiddleware implements the ICS26 callbacks for the wormhole middleware given the
// forward keeper and the underlying application.
type IBCMiddleware struct {
	app            porttypes.IBCModule
	wasmKeeper     *wasmkeeper.Keeper
	wormholeKeeper *wormholekeeper.Keeper

	retriesOnTimeout uint8
	forwardTimeout   time.Duration
	refundTimeout    time.Duration
}

func (im *IBCMiddleware) SetWasmKeeper(keeper *wasmkeeper.Keeper) {
	im.wasmKeeper = keeper
}

// NewIBCMiddleware creates a new IBCMiddleware given the keeper and underlying application.
func NewIBCMiddleware(
	app porttypes.IBCModule,
	k *wasmkeeper.Keeper,
	wk *wormholekeeper.Keeper,
	retriesOnTimeout uint8,
	forwardTimeout time.Duration,
	refundTimeout time.Duration,
) IBCMiddleware {
	return IBCMiddleware{
		app:              app,
		wasmKeeper:       k,
		wormholeKeeper:   wk,
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
	// If wasm keeper is not set, bypass this middleware
	if im.wasmKeeper == nil {
		return im.app.OnRecvPacket(ctx, packet, relayer)
	}

	var data transfertypes.FungibleTokenPacketData
	if err := transfertypes.ModuleCdc.UnmarshalJSON(packet.GetData(), &data); err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	memo := make(map[string]interface{})
	err := json.Unmarshal([]byte(data.Memo), &memo)
	if err != nil || memo["gateway_ibc_token_bridge_payload"] == nil {
		// not a packet that should be parsed
		return im.app.OnRecvPacket(ctx, packet, relayer)
	}

	parsedPayload, err := types.VerifyAndParseGatewayPayload(data.Memo)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	// Get ibc translator contract address
	wormholeMiddlewareContract := im.wormholeKeeper.GetMiddlewareContract(ctx)

	// Look up chain id's channel
	req := types.IbcTranslatorQueryMsg{
		IbcChannel: types.QueryIbcChannel{
			ChainID: parsedPayload.ChainId,
		},
	}
	reqBz, err := json.Marshal(req)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	ibcTranslatorAddr, err := sdk.AccAddressFromBech32(wormholeMiddlewareContract.ContractAddress)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}
	resp, err := im.wasmKeeper.QuerySmart(ctx, ibcTranslatorAddr, reqBz)
	
	var newMemo string
	if err == nil {
		// If response exists, create PFM memo
		newMemo, err = types.FormatPfmMemo(parsedPayload, resp)
		if err != nil {
			return channeltypes.NewErrorAcknowledgement(err)
		}
	} else {
		// If response doesn't exist, create ibc-hooks memo
		newMemo, err = types.FormatIbcHooksMemo(parsedPayload, wormholeMiddlewareContract.ContractAddress)
		if err != nil {
			return channeltypes.NewErrorAcknowledgement(err)
		}
	}

	data.Memo = newMemo
	newData, err := transfertypes.ModuleCdc.MarshalJSON(&data)
	if err != nil {
		return channeltypes.NewErrorAcknowledgement(err)
	}

	packet.Data = newData

	return im.app.OnRecvPacket(ctx, packet, relayer)
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
