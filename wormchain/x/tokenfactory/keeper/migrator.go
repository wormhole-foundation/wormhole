package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	"github.com/wormhole-foundation/wormchain/x/tokenfactory/exported"
	v2 "github.com/wormhole-foundation/wormchain/x/tokenfactory/migrations/v2"
)

// Migrator is a struct for handling in-place state migrations.
type Migrator struct {
	keeper         Keeper
	legacySubspace exported.Subspace
}

func NewMigrator(k Keeper, ss exported.Subspace) Migrator {
	return Migrator{
		keeper:         k,
		legacySubspace: ss,
	}
}

// Migrate1to2 migrates the x/tokenfactory module state from the consensus version 1 to
// version 2. Specifically, it takes the parameters that are currently stored
// and managed by the x/params modules and stores them directly into the x/tokenfactory
// module state.
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
	// Fixes hard forking genesis being invalid.
	// https://github.com/sei-protocol/sei-chain/pull/861
	iter := m.keeper.GetAllDenomsIterator(ctx)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		denom := string(iter.Value())
		denomMetadata, err := m.keeper.bankKeeper.GetDenomMetaData(ctx, denom)
		if err {
			panic(fmt.Errorf("denom %s does not exist", denom))
		}

		fmt.Printf("Migrating denom: %s\n", denom)
		m.SetMetadata(&denomMetadata)
		m.keeper.bankKeeper.SetDenomMetaData(ctx, denomMetadata)

	}

	return v2.Migrate(ctx, ctx.KVStore(m.keeper.storeKey), m.legacySubspace, m.keeper.cdc)
}

func (m Migrator) SetMetadata(denomMetadata *banktypes.Metadata) {
	if len(denomMetadata.Base) == 0 {
		panic(fmt.Errorf("no base exists for denom %v", denomMetadata))
	}
	if len(denomMetadata.Display) == 0 {
		denomMetadata.Display = denomMetadata.Base
		denomMetadata.Name = denomMetadata.Base
		denomMetadata.Symbol = denomMetadata.Base
	} else {
		fmt.Printf("Denom %s already has denom set", denomMetadata.Base)
	}
}
