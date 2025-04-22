package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestExecuteSlashingParamsUpdate(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 10)
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})
	signer_bz := [20]byte{}
	signer := sdk.AccAddress(signer_bz[:])

	set := createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{Index: set.Index})

	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// create governance to update slashing params
	payloadBody := vaa.BodyGatewaySlashingParamsUpdate{
		SignedBlocksWindow:      uint64(100),
		MinSignedPerWindow:      sdk.NewDecWithPrec(5, 1).BigInt().Uint64(),
		DowntimeJailDuration:    uint64(600 * time.Second),
		SlashFractionDoubleSign: sdk.NewDecWithPrec(5, 2).BigInt().Uint64(),
		SlashFractionDowntime:   sdk.NewDecWithPrec(1, 2).BigInt().Uint64(),
	}
	payloadBz, err := payloadBody.Serialize()
	assert.NoError(t, err)

	v := generateVaa(set.Index, privateKeys, vaa.ChainID(vaa.GovernanceChain), payloadBz)
	vBz, _ := v.Marshal()
	res, err := msgServer.ExecuteGatewayGovernanceVaa(context, &types.MsgExecuteGatewayGovernanceVaa{
		Signer: signer.String(),
		Vaa:    vBz,
	})
	assert.NoError(t, err)
	assert.Equal(t, &types.EmptyResponse{}, res)
}
