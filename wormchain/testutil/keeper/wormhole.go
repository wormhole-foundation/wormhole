package keeper

import (
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/app"
	"github.com/wormhole-foundation/wormchain/app/apptesting"
	ibccomposabilitymw "github.com/wormhole-foundation/wormchain/x/ibc-composability-mw/keeper"
	wormholekeeper "github.com/wormhole-foundation/wormchain/x/wormhole/keeper"

	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

func WormholeKeeper(t *testing.T) (*wormholekeeper.Keeper, sdk.Context) {
	app, ctx := SetupWormchainAndContext(t)
	return &app.WormholeKeeper, ctx
}

func WormholeKeeperAndIBCComposabilityMwKeeper(t *testing.T) (*wormholekeeper.Keeper, *ibccomposabilitymw.Keeper, sdk.Context) {
	app, ctx := SetupWormchainAndContext(t)
	return &app.WormholeKeeper, app.IbcComposabilityMwKeeper, ctx
}

func WormholeKeeperAndWasmd(t *testing.T) (*wormholekeeper.Keeper, *wasmkeeper.Keeper, *wasmkeeper.PermissionedKeeper, sdk.Context) {
	app, ctx := SetupWormchainAndContext(t)

	wasmGenState := wasmtypes.GenesisState{}
	wasmGenState.Params.CodeUploadAccess = wasmtypes.DefaultUploadAccess
	wasmGenState.Params.InstantiateDefaultPermission = wasmtypes.AccessTypeEverybody
	app.GetWasmKeeper().SetParams(ctx, wasmGenState.Params)

	return &app.WormholeKeeper, app.GetWasmKeeper(), app.ContractKeeper, ctx
}

func SetupWormchainAndContext(t *testing.T) (*app.App, sdk.Context) {
	app := apptesting.Setup(t, false, 0)

	ctx := app.BaseApp.NewContext(false, tmproto.Header{
		ChainID: apptesting.SimAppChainID,
		// The height should be at least 1, because the allowlist antehandler
		// passes everything at height 0 for gen tx's.
		Height: 1,
		Time:   time.Now(),
	})

	return app, ctx
}
