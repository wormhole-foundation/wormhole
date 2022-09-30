package simulation

import (
	"math/rand"

	"github.com/certusone/wormhole-chain/x/wormhole/keeper"
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
)

func SimulateMsgRegisterAccountAsGuardian(
	k keeper.Keeper,
) simtypes.Operation {
	return func(r *rand.Rand, app *baseapp.BaseApp, ctx sdk.Context, accs []simtypes.Account, chainID string,
	) (simtypes.OperationMsg, []simtypes.FutureOperation, error) {
		simAccount, _ := simtypes.RandomAcc(r, accs)
		msg := &types.MsgRegisterAccountAsGuardian{
			Signer: simAccount.Address.String(),
		}

		// TODO: Handling the RegisterAccountAsGuardian simulation

		return simtypes.NoOpMsg(types.ModuleName, msg.Type(), "RegisterAccountAsGuardian simulation not implemented"), nil, nil
	}
}
