package app

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	slashingkeeper "github.com/cosmos/cosmos-sdk/x/slashing/keeper"
)

// Reject all messages if we're expecting a software update.
type MigrationDecorator struct {
	slashingKeeper slashingkeeper.Keeper
}

func NewMigrationDecorator(k slashingkeeper.Keeper) MigrationDecorator {
	return MigrationDecorator{
		slashingKeeper: k,
	}
}

// An ante handler for migrating parameters of other modules that would normally be set by
// InitGenesis.  This allows the chain to change parameters without a reset or fork.
func (wh MigrationDecorator) AnteHandle(request sdk.Request, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (newCtx sdk.Request, err error) {
	if request.BlockHeight() == 2000000 {
		params := wh.slashingKeeper.GetParams(request)
		// use 16k blocks or ~1 day as the allowed downtime before being jailed
		params.SignedBlocksWindow = 16 * 1000
		wh.slashingKeeper.SetParams(request, params)
	}
	return next(request, tx, simulate)
}
