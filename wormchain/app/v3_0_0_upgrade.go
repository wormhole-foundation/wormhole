package app

import (
	"fmt"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	consensustypes "github.com/cosmos/cosmos-sdk/x/consensus/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	distrtypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	govv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	minttypes "github.com/cosmos/cosmos-sdk/x/mint/types"
	paramstypes "github.com/cosmos/cosmos-sdk/x/params/types"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"
	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibcexported "github.com/cosmos/ibc-go/v7/modules/core/exported"
	tokenfactorytypes "github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
	wormholetypes "github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

var V3_0_0_Upgrade = Upgrade{
	UpgradeName:          "v3.0.0",
	CreateUpgradeHandler: CreateV3_0_0_UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			crisistypes.ModuleName,    // add crisis module store
			consensustypes.ModuleName, // add consensus module store
		},
	},
}

func CreateV3_0_0_UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	app *App,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger().With("upgrade", "v3.0.0")
		logger.Info("Starting Upgrade")

		// set param key table for params module migration
		// ref: https://github.com/cosmos/cosmos-sdk/pull/12363/files
		for _, subspace := range app.ParamsKeeper.GetSubspaces() {
			subspace := subspace
			var keyTable paramstypes.KeyTable

			switch subspace.Name() {

			// cosmos-sdk modules
			case authtypes.ModuleName:
				keyTable = authtypes.ParamKeyTable() //nolint:staticcheck
			case banktypes.ModuleName:
				keyTable = banktypes.ParamKeyTable() //nolint:staticcheck
			case stakingtypes.ModuleName:
				keyTable = stakingtypes.ParamKeyTable() //nolint:staticcheck
			case minttypes.ModuleName:
				keyTable = minttypes.ParamKeyTable() //nolint:staticcheck
			case distrtypes.ModuleName:
				keyTable = distrtypes.ParamKeyTable() //nolint:staticcheck
			case slashingtypes.ModuleName:
				keyTable = slashingtypes.ParamKeyTable() //nolint:staticcheck
			case govtypes.ModuleName:
				keyTable = govv1.ParamKeyTable() //nolint:staticcheck
			case crisistypes.ModuleName:
				keyTable = crisistypes.ParamKeyTable() //nolint:staticcheck

			// ibc-go modules
			case ibctransfertypes.ModuleName:
				keyTable = ibctransfertypes.ParamKeyTable()

			// wasm modules
			case wasmtypes.ModuleName:
				keyTable = wasmtypes.ParamKeyTable() //nolint:staticcheck

			// wormhole modules
			case tokenfactorytypes.ModuleName:
				keyTable = tokenfactorytypes.ParamKeyTable()
			case wormholetypes.ModuleName:
				continue // skip wormhole module
			}

			if !subspace.HasKeyTable() {
				subspace.WithKeyTable(keyTable)
			}
		}

		// migrate params to new conensus params keeper
		// ref:https://github.com/cosmos/cosmos-sdk/blob/release/v0.47.x/UPGRADING.md#xconsensus
		baseAppLegacySS := app.ParamsKeeper.Subspace(baseapp.Paramspace).WithKeyTable(paramstypes.ConsensusParamsKeyTable())
		baseapp.MigrateParams(ctx, baseAppLegacySS, &app.ConsensusParamsKeeper)

		// Run migrations
		logger.Info(fmt.Sprintf("pre migrate version map: %v", vm))
		versionMap, err := mm.RunMigrations(ctx, cfg, vm)
		if err != nil {
			return nil, err
		}
		logger.Info(fmt.Sprintf("post migrate version map: %v", versionMap))

		// explicitly update the IBC 02-client params, adding the localhost client type
		// ref: https://github.com/cosmos/ibc-go/blob/main/docs/docs/05-migrations/09-v7-to-v7_1.md
		params := app.IBCKeeper.ClientKeeper.GetParams(ctx)
		params.AllowedClients = append(params.AllowedClients, ibcexported.Localhost)
		app.IBCKeeper.ClientKeeper.SetParams(ctx, params)

		return versionMap, err
	}
}
