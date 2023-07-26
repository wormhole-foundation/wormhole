package simulation

import (
	"fmt"
	"math/rand"

	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
	"github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
)

func ParamChanges(_ *rand.Rand) []simtypes.ParamChange {
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(
			types.ModuleName,
			string(types.KeyDenomCreationFee),
			func(r *rand.Rand) string {
				amount := RandDenomCreationFeeParam(r)
				return fmt.Sprintf("[{\"denom\":\"%v\",\"amount\":\"%v\"}]", amount[0].Denom, amount[0].Amount)
			},
		),
	}
}
