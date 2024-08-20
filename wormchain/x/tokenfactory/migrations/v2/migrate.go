package v2

import (
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/wormhole-foundation/wormchain/x/tokenfactory/exported"
	"github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
)

const ModuleName = "tokenfactory"

var ParamsKey = []byte{0x00}

// Migrate migrates the x/tokenfactory module state from the consensus version 1 to
// version 2. Specifically, it takes the parameters that are currently stored
// and managed by the x/params modules and stores them directly into the x/tokenfactory
// module state.
func Migrate(
	_ sdk.Context,
	store sdk.KVStore,
	_ exported.Subspace,
	cdc codec.BinaryCodec,
) error {
	// Migrates mainnet params -> the new keeper params storeKey (from x/params)
	currParams := types.Params{
		DenomCreationFee:        nil,
		DenomCreationGasConsume: 2_000_000,
	}

	if err := currParams.Validate(); err != nil {
		return err
	}

	bz := cdc.MustMarshal(&currParams)
	store.Set(ParamsKey, bz)

	return nil
}
