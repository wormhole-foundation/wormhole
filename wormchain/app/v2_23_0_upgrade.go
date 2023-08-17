package app

import (
	store "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/module"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	packetforwardtypes "github.com/strangelove-ventures/packet-forward-middleware/v4/router/types"
	ibccomposabilitytypes "github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/types"
	ibchookstypes "github.com/wormhole-foundation/wormchain/x/ibc-hooks/types"
	tokenfactorytypes "github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
)

var V2_23_0_Upgrade = Upgrade{
	UpgradeName:          "v2.23.0",
	CreateUpgradeHandler: CreateV2_23_0_UpgradeHandler,
	StoreUpgrades: store.StoreUpgrades{
		Added: []string{
			tokenfactorytypes.ModuleName,
			ibchookstypes.StoreKey,
			packetforwardtypes.StoreKey,
			ibccomposabilitytypes.StoreKey,
		},
	},
}

func CreateV2_23_0_UpgradeHandler(
	mm *module.Manager,
	cfg module.Configurator,
	app *App,
) upgradetypes.UpgradeHandler {
	return func(ctx sdk.Context, _ upgradetypes.Plan, vm module.VersionMap) (module.VersionMap, error) {
		logger := ctx.Logger().With("upgrade", "v2.23.0")

		// TokenFactory
		newTokenFactoryParams := tokenfactorytypes.Params{
			DenomCreationFee:        nil,
			DenomCreationGasConsume: 0,
		}

		app.TokenFactoryKeeper.SetParams(ctx, newTokenFactoryParams)
		logger.Info("set tokenfactory params")

		// Packet Forward middleware initial params
		app.PacketForwardKeeper.SetParams(ctx, packetforwardtypes.DefaultParams())

		return mm.RunMigrations(ctx, cfg, vm)
	}
}
