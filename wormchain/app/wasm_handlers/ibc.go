package wasm_handlers

import (
	"errors"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	ibcappkeeper "github.com/cosmos/ibc-go/v4/modules/apps/transfer/keeper"
	connectiontypes "github.com/cosmos/ibc-go/v4/modules/core/03-connection/types"
	ibckeeper "github.com/cosmos/ibc-go/v4/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v4/modules/core/04-channel/types"
	ibcportkeeper "github.com/cosmos/ibc-go/v4/modules/core/05-port/keeper"
	ibcexported "github.com/cosmos/ibc-go/v4/modules/core/exported"
)

// This defines which modules we actually want to expose cosmwasm contracts.
// Right now we only permit methods that are read-only.
// See https://github.com/CosmWasm/wasmd/blob/d63bea442bedf5b3055f3821472c7e6cafc3d813/x/wasm/types/expected_keepers.go

type ChannelKeeperHandler struct {
	Keeper ibckeeper.Keeper
}
type ClientKeeperHandler struct {
	Keeper ibckeeper.Keeper
}
type ConnectionKeeperHandler struct {
	Keeper ibckeeper.Keeper
}
type PortKeeperHandler struct {
	Keeper ibcportkeeper.Keeper
}
type ICS20TransferPortSourceHandler struct {
	Keeper ibcappkeeper.Keeper
}

var _ wasmtypes.ChannelKeeper = &ChannelKeeperHandler{}
var _ wasmtypes.ClientKeeper = &ClientKeeperHandler{}
var _ wasmtypes.ConnectionKeeper = &ConnectionKeeperHandler{}
var _ wasmtypes.PortKeeper = &PortKeeperHandler{}
var _ wasmtypes.ICS20TransferPortSource = &ICS20TransferPortSourceHandler{}

func (b *ChannelKeeperHandler) GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
	return b.Keeper.GetChannel(ctx, srcPort, srcChan)
}
func (b *ChannelKeeperHandler) GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool) {
	// not permitted
	return 0, false
}
func (b *ChannelKeeperHandler) SendPacket(ctx sdk.Context, channelCap *capabilitytypes.Capability, packet ibcexported.PacketI) error {
	return errors.New("not permitted")
}
func (b *ChannelKeeperHandler) ChanCloseInit(ctx sdk.Context, portID, channelID string, chanCap *capabilitytypes.Capability) error {
	return errors.New("not permitted")
}
func (b *ChannelKeeperHandler) GetAllChannels(ctx sdk.Context) (channels []channeltypes.IdentifiedChannel) {
	// not permitted
	return []channeltypes.IdentifiedChannel{}
}
func (b *ChannelKeeperHandler) IterateChannels(ctx sdk.Context, cb func(channeltypes.IdentifiedChannel) bool) {
	// not permitted
}
func (b *ChannelKeeperHandler) SetChannel(ctx sdk.Context, portID, channelID string, channel channeltypes.Channel) {
	// not permitted
}

func (b *ClientKeeperHandler) GetClientConsensusState(ctx sdk.Context, clientID string) (ibcexported.ConsensusState, bool) {
	return nil, false
}

func (b *ConnectionKeeperHandler) GetConnection(ctx sdk.Context, connectionID string) (connection connectiontypes.ConnectionEnd, found bool) {
	return connectiontypes.ConnectionEnd{}, false
}

func (b *PortKeeperHandler) BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability {
	return nil
}

func (b *ICS20TransferPortSourceHandler) GetPort(ctx sdk.Context) string {
	// not permitted
	return ""
}
