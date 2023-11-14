package app

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
)

// Temp upgrade handler, replace with targeted version
var V2_24_0_Upgrade = Upgrade{
	UpgradeName:          "v2.24.0",
	CreateUpgradeHandler: CreateV2_24_0_UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{},
	},
}

func CreateV2_24_0_UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	app *App,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger().With("upgrade", "v2.24.0")
		logger.Info("empty v2.24.0 upgrade handler")

		return mm.RunMigrations(ctx, cfg, vm)
	}
}
