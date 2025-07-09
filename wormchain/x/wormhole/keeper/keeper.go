package keeper

import (
	"fmt"

	"github.com/cometbft/cometbft/libs/log"

	"github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
	upgradekeeper "github.com/cosmos/cosmos-sdk/x/upgrade/keeper"
	clientkeeper "github.com/cosmos/ibc-go/v7/modules/core/02-client/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

type (
	Keeper struct {
		cdc      codec.BinaryCodec
		storeKey storetypes.StoreKey
		memKey   storetypes.StoreKey

		accountKeeper  types.AccountKeeper
		bankKeeper     types.BankKeeper
		wasmdKeeper    types.WasmdKeeper
		upgradeKeeper  *upgradekeeper.Keeper
		slashingKeeper slashingkeeper.Keeper
		clientKeeper   clientkeeper.Keeper

		authority string

		setWasmd        bool
		setUpgrade      bool
		setSlashing     bool
		setClientKeeper bool
	}
)

func NewKeeper(
	cdc codec.BinaryCodec,
	storeKey,
	memKey storetypes.StoreKey,
	accountKeeper types.AccountKeeper,
	bankKeeper types.BankKeeper,
	authority string,
) *Keeper {
	return &Keeper{
		cdc:           cdc,
		storeKey:      storeKey,
		memKey:        memKey,
		accountKeeper: accountKeeper,
		bankKeeper:    bankKeeper,
		authority:     authority,
	}
}

// This is necessary because x/staking relies on x/wormhole and x/wasmd relies on x/staking,
// So we must either:
// 1. make wormhole depend on staking and replace the modified functions from here.
// 2. add a new module that wraps x/wasmd instead of using x/wormhole.
// 3. (current) set wasmdKeeper late in init and use guards whenever it's referenced.
// Opted for (3) as we only reference in two places.
func (k *Keeper) SetWasmdKeeper(keeper types.WasmdKeeper) {
	k.wasmdKeeper = keeper
	k.setWasmd = true
}

func (k *Keeper) SetUpgradeKeeper(keeper *upgradekeeper.Keeper) {
	k.upgradeKeeper = keeper
	k.setUpgrade = true
}

func (k *Keeper) SetSlashingKeeper(keeper slashingkeeper.Keeper) {
	k.slashingKeeper = keeper
	k.setSlashing = true
}

func (k *Keeper) SetClientKeeper(keeper clientkeeper.Keeper) {
	k.clientKeeper = keeper
	k.setClientKeeper = true
}

func (k Keeper) Logger(ctx sdk.Context) log.Logger {
	return ctx.Logger().With("module", fmt.Sprintf("x/%s", types.ModuleName))
}
