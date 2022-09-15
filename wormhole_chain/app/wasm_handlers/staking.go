package wasm_handlers

import (
	"context"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	distrkeeper "github.com/cosmos/cosmos-sdk/x/distribution/keeper"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
)

type DistributionKeeperHandler struct {
	Keeper distrkeeper.Keeper
}
type StakingKeeperHandler struct {
	Keeper stakingkeeper.Keeper
}

var _ wasmtypes.DistributionKeeper = &DistributionKeeperHandler{}
var _ wasmtypes.StakingKeeper = &StakingKeeperHandler{}

func (b *DistributionKeeperHandler) DelegationRewards(c context.Context, req *distrtypes.QueryDelegationRewardsRequest) (*distrtypes.QueryDelegationRewardsResponse, error) {
	return b.Keeper.DelegationRewards(c, req)
}

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
