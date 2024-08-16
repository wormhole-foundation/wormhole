package keeper_test

import (
	"testing"

	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// TestGuardianSetUpdateProposal tests possible scenarios for how a guardian set update proposal can be handled
func TestGuardianSetUpdateProposal(t *testing.T) {
	// get app & ctx
	app, ctx := keepertest.SetupWormchainAndContext(t)

	// get keeper & msg server
	k := app.WormholeKeeper
	msgServer := keeper.NewMsgServerImpl(k)

	// create message
	msg := &types.MsgGuardianSetUpdateProposal{
		Authority:      "invalid-authority",
		NewGuardianSet: types.GuardianSet{Index: 1},
	}

	// TEST: FAIL - invalid authority
	_, err := msgServer.GuardianSetUpdateProposal(ctx, msg)
	require.Error(t, err)

	// set valid authority
	msg.Authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// TEST: FAIL - no configuration
	_, err = msgServer.GuardianSetUpdateProposal(ctx, msg)
	require.Error(t, err)

	// set valid configuration
	k.SetConfig(ctx, types.Config{
		ChainId:           uint32(vaa.ChainIDWormchain),
		GovernanceEmitter: vaa.GovernanceEmitter[:],
		GovernanceChain:   uint32(vaa.GovernanceChain),
	})

	// TEST: FAIL - guardian set not found
	_, err = msgServer.GuardianSetUpdateProposal(ctx, msg)
	require.Error(t, err)

	// init keep with first guardian set
	_, err = k.AppendGuardianSet(ctx, types.GuardianSet{Index: 0})
	require.NoError(t, err)

	// TEST: SUCCESS - guardian set updated to Index=1
	_, err = msgServer.GuardianSetUpdateProposal(ctx, msg)
	require.NoError(t, err)

	// TEST: FAIL - guardian set not sequential (index should be 2)
	msg.NewGuardianSet.Index = 3
	_, err = msgServer.GuardianSetUpdateProposal(ctx, msg)
	require.Error(t, err)
	msg.NewGuardianSet.Index = 2

	// TEST: SUCCESS - keeper overrides expiration to 0, so the set will never expire
	msg.NewGuardianSet.ExpirationTime = 1
	_, err = msgServer.GuardianSetUpdateProposal(ctx, msg)
	require.NoError(t, err)

	// TEST: FAIL - invalid governance emitter
	k.SetConfig(ctx, types.Config{
		ChainId:           uint32(vaa.ChainIDWormchain),
		GovernanceEmitter: []byte("invalid-emitter"),
		GovernanceChain:   uint32(vaa.GovernanceChain),
	})
	_, err = msgServer.GuardianSetUpdateProposal(ctx, msg)
	require.Error(t, err)
}
