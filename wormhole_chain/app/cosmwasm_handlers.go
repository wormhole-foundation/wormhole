package app

import (
	"context"
	"errors"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	capabilitykeeper "github.com/cosmos/cosmos-sdk/x/capability/keeper"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	ibcappkeeper "github.com/cosmos/ibc-go/v3/modules/apps/transfer/keeper"
	connectiontypes "github.com/cosmos/ibc-go/v3/modules/core/03-connection/types"
	ibckeeper "github.com/cosmos/ibc-go/v3/modules/core/04-channel/keeper"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	ibcportkeeper "github.com/cosmos/ibc-go/v3/modules/core/05-port/keeper"
	ibcexported "github.com/cosmos/ibc-go/v3/modules/core/exported"
)

// This defines which modules we actually want to expose cosmwasm contracts.
// Right now we only permit methods that are read-only.
// See https://github.com/CosmWasm/wasmd/blob/d63bea442bedf5b3055f3821472c7e6cafc3d813/x/wasm/types/expected_keepers.go

type BankViewKeeperHandler struct {
	Keeper bankkeeper.Keeper
}
type BurnerHandler struct {
	Keeper bankkeeper.Keeper
}

type BankKeeperHandler struct {
	BankViewKeeperHandler
	BurnerHandler
}

type AccountKeeperHandler struct {
	AccountKeeper authkeeper.AccountKeeper
}
type DistributionKeeperHandler struct {
	Keeper distrkeeper.Keeper
}
type StakingKeeperHandler struct {
	Keeper stakingkeeper.Keeper
}
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
type CapabilityKeeperHandler struct {
	ScopedKeeper capabilitykeeper.ScopedKeeper
}
type ICS20TransferPortSourceHandler struct {
	Keeper ibcappkeeper.Keeper
}

var _ wasmtypes.BankViewKeeper = &BankViewKeeperHandler{}

func (b *BankViewKeeperHandler) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return b.Keeper.GetAllBalances(ctx, addr)
}
func (b *BankViewKeeperHandler) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return b.Keeper.GetBalance(ctx, addr, denom)
}

var _ wasmtypes.Burner = &BurnerHandler{}

func (b *BurnerHandler) BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	return errors.New("not permitted")
}

func (b *BurnerHandler) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return errors.New("not permitted")
}

var _ wasmtypes.BankKeeper = &BankKeeperHandler{}

func (b *BankKeeperHandler) IsSendEnabledCoins(ctx sdk.Context, coins ...sdk.Coin) error {
	return errors.New("not permitted")
}
func (b *BankKeeperHandler) BlockedAddr(addr sdk.AccAddress) bool {
	return false
}
func (b *BankKeeperHandler) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return errors.New("not permitted")
}

var _ wasmtypes.AccountKeeper = &AccountKeeperHandler{}

func (b *AccountKeeperHandler) NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI {
	// New accounts are needed for new contracts
	return b.AccountKeeper.NewAccountWithAddress(ctx, addr)
}

// Retrieve an account from the store.
func (b *AccountKeeperHandler) GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI {
	return b.AccountKeeper.GetAccount(ctx, addr)
}

// Set an account in the store.
func (b *AccountKeeperHandler) SetAccount(ctx sdk.Context, acc authtypes.AccountI) {
	// New accounts are needed for new contracts
	b.AccountKeeper.SetAccount(ctx, acc)
}

var _ wasmtypes.DistributionKeeper = &DistributionKeeperHandler{}

func (b *DistributionKeeperHandler) DelegationRewards(c context.Context, req *distrtypes.QueryDelegationRewardsRequest) (*distrtypes.QueryDelegationRewardsResponse, error) {
	return b.Keeper.DelegationRewards(c, req)
}

var _ wasmtypes.StakingKeeper = &StakingKeeperHandler{}

func (b *StakingKeeperHandler) BondDenom(ctx sdk.Context) (res string) {
	return b.Keeper.BondDenom(ctx)
}

func (b *StakingKeeperHandler) GetValidator(ctx sdk.Context, addr sdk.ValAddress) (validator stakingtypes.Validator, found bool) {
	return b.Keeper.GetValidator(ctx, addr)
}

func (b *StakingKeeperHandler) GetBondedValidatorsByPower(ctx sdk.Context) []stakingtypes.Validator {
	return b.Keeper.GetBondedValidatorsByPower(ctx)
}

func (b *StakingKeeperHandler) GetAllDelegatorDelegations(ctx sdk.Context, delegator sdk.AccAddress) []stakingtypes.Delegation {
	return b.Keeper.GetAllDelegatorDelegations(ctx, delegator)
}

func (b *StakingKeeperHandler) GetDelegation(ctx sdk.Context,
	delAddr sdk.AccAddress, valAddr sdk.ValAddress) (delegation stakingtypes.Delegation, found bool) {
	return b.Keeper.GetDelegation(ctx, delAddr, valAddr)
}

func (b *StakingKeeperHandler) HasReceivingRedelegation(ctx sdk.Context,
	delAddr sdk.AccAddress, valDstAddr sdk.ValAddress) bool {
	return b.Keeper.HasReceivingRedelegation(ctx, delAddr, valDstAddr)
}

var _ wasmtypes.ChannelKeeper = &ChannelKeeperHandler{}

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

var _ wasmtypes.ClientKeeper = &ClientKeeperHandler{}

func (b *ClientKeeperHandler) GetClientConsensusState(ctx sdk.Context, clientID string) (ibcexported.ConsensusState, bool) {
	return nil, false
}

var _ wasmtypes.ConnectionKeeper = &ConnectionKeeperHandler{}

func (b *ConnectionKeeperHandler) GetConnection(ctx sdk.Context, connectionID string) (connection connectiontypes.ConnectionEnd, found bool) {
	return connectiontypes.ConnectionEnd{}, false
}

var _ wasmtypes.PortKeeper = &PortKeeperHandler{}

func (b *PortKeeperHandler) BindPort(ctx sdk.Context, portID string) *capabilitytypes.Capability {
	return nil
}

var _ wasmtypes.CapabilityKeeper = &CapabilityKeeperHandler{}

func (b *CapabilityKeeperHandler) GetCapability(ctx sdk.Context, name string) (*capabilitytypes.Capability, bool) {
	return b.ScopedKeeper.GetCapability(ctx, name)
}
func (b *CapabilityKeeperHandler) ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error {
	return b.ScopedKeeper.ClaimCapability(ctx, cap, name)
}
func (b *CapabilityKeeperHandler) AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool {
	return b.ScopedKeeper.AuthenticateCapability(ctx, cap, name)
}

var _ wasmtypes.ICS20TransferPortSource = &ICS20TransferPortSourceHandler{}

func (b *ICS20TransferPortSourceHandler) GetPort(ctx sdk.Context) string {
	// not permitted
	return ""
}
