package keeper

import (
	"testing"
	"time"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/app/apptesting"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"

	tmproto "github.com/tendermint/tendermint/proto/tendermint/types"
)

func WormholeKeeper(t *testing.T) (*keeper.Keeper, sdk.Context) {
	k, _, _, ctx := WormholeKeeperAndWasmd(t)
	return k, ctx
}

func WormholeKeeperAndWasmd(t *testing.T) (*keeper.Keeper, wasmkeeper.Keeper, *wasmkeeper.PermissionedKeeper, sdk.Context) {
	app := apptesting.Setup(t, false, 0)

	ctx := app.BaseApp.NewContext(false, tmproto.Header{
		ChainID: apptesting.SimAppChainID,
		// The height should be at least 1, because the allowlist antehandler
		// passes everything at height 0 for gen tx's.
		Height: 1,
		Time:   time.Now(),
	})

	return &app.WormholeKeeper, *app.GetWasmKeeper(), app.ContractKeeper, ctx
}
