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

// TestPostMessageProposal tests possible scenarios for how a governance wormhole message proposal can be handled.
func TestPostMessageProposal(t *testing.T) {
	// get app & ctx
	app, ctx := keepertest.SetupWormchainAndContext(t)

	// get keeper & msg server
	k := app.WormholeKeeper
	msgServer := keeper.NewMsgServerImpl(k)

	// create message
	msg := &types.MsgGovernanceWormholeMessageProposal{
		Authority:   "invalid-authority",
		Action:      0,
		Module:      []byte{},
		TargetChain: 0,
		Payload:     []byte{},
	}

	// TEST: FAIL - invalid authority
	_, err := msgServer.GovernanceWormholeMessageProposal(ctx, msg)
	require.Error(t, err)

	// set valid authority
	msg.Authority = authtypes.NewModuleAddress(govtypes.ModuleName).String()

	// TEST: FAIL - no configuration
	_, err = msgServer.GovernanceWormholeMessageProposal(ctx, msg)
	require.Error(t, err)

	// set config with valid emitter
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter: vaa.GovernanceEmitter[:],
	})

	// TEST: SUCCESS - valid authority & config
	_, err = msgServer.GovernanceWormholeMessageProposal(ctx, msg)
	require.NoError(t, err)
}
