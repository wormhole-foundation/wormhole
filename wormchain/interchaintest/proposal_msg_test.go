package interchaintest

import (
	"context"
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
	cosmosproto "github.com/cosmos/gogoproto/proto"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	wormholetypes "github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// haltHeightDelta is the number of blocks to wait for a proposal to pass
var (
	haltHeightDelta = int64(10)
	numGuardians    = 2
)

// setupProposalTest is a helper function to setup a wormchain test with 2 users:
// the first capable of submitting proposals, the second not.
func setupProposalTest(t *testing.T) (context.Context, *cosmos.CosmosChain, ibc.Wallet, ibc.Wallet) {
	// Base setup
	guardians := guardians.CreateValSet(t, numGuardians)
	chains := CreateLocalChain(t, *guardians)
	_, ctx, _, _, _, _ := BuildInterchain(t, chains)

	wormchain := chains[0].(*cosmos.CosmosChain)

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(10_000_000_000), wormchain, wormchain)
	user1 := users[0]
	user2 := users[1]

	val := wormchain.Validators[0]
	_, err := val.ExecTx(ctx, "validator", "wormhole", "create-allowed-address", user1.FormattedAddress(), "UserProposalSubmitter")
	require.NoError(t, err, "error creating allowed address")

	return ctx, wormchain, user1, user2
}

// TestGuardianSetUpdateProposal tests the process of submitting a guardian set update proposal
func TestGuardianSetUpdateProposal(t *testing.T) {
	ctx, wormchain, user1, user2 := setupProposalTest(t)

	var keys [][]byte

	for range numGuardians {
		guardian := guardians.CreateVal(t)
		keys = append(keys, guardian.Addr)
	}

	emitMsgProposal := []cosmosproto.Message{
		&wormholetypes.MsgGuardianSetUpdateProposal{
			Authority: "wormhole10d07y265gmmuvt4z0w9aw880jnsr700j5x7ea3",
			NewGuardianSet: wormholetypes.GuardianSet{
				Index:          1,
				Keys:           keys,
				ExpirationTime: 0,
			},
		},
	}

	proposalDraft, err := wormchain.BuildProposal(emitMsgProposal, "Emit Wormhole Message", "emit msg", "ipfs://CID", fmt.Sprintf(`500000000%s`, wormchain.Config().Denom))
	require.NoError(t, err, "error building proposal")

	// First attempt (should fail because user2 is not allowed to submit proposals)
	_, err = wormchain.SubmitProposal(ctx, user2.FormattedAddress(), proposalDraft)
	require.Error(t, err, "expected error submitting proposal")

	// Second attempt (should succeed because user1 is allowed to submit proposals)
	txProp, err := wormchain.SubmitProposal(ctx, user1.FormattedAddress(), proposalDraft)
	t.Log("txProp", txProp)
	require.NoError(t, err, "error submitting proposal")

	// Get height after proposal submission
	height, _ := wormchain.Height(ctx)

	proposalID, err := strconv.ParseInt(txProp.ProposalID, 10, 64)
	require.NoError(t, err, "failed to parse proposal ID")

	// Force all validators vote on proposal
	err = wormchain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	// Poll for proposal status to change to passed
	proposal, err := cosmos.PollForProposalStatus(ctx, wormchain, height, height+haltHeightDelta, proposalID, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")
	require.NotEmpty(t, proposal, "proposal not found")

	// Wait for blocks
	err = testutil.WaitForBlocks(ctx, 10, wormchain)
	require.NoError(t, err, "error waiting for blocks")
}

// TestGovernanceWormholeMessageProposal tests the process of submitting a governance proposal to emit a wormhole message
func TestGovernanceWormholeMessageProposal(t *testing.T) {
	ctx, wormchain, user1, user2 := setupProposalTest(t)

	emitMsgProposal := []cosmosproto.Message{
		&wormholetypes.MsgGovernanceWormholeMessageProposal{
			Authority:   "wormhole10d07y265gmmuvt4z0w9aw880jnsr700j5x7ea3",
			Action:      1,
			Module:      vaa.CoreModule,
			Payload:     []byte("payload"),
			TargetChain: 1,
		},
	}

	proposalDraft, err := wormchain.BuildProposal(emitMsgProposal, "Emit Wormhole Message", "emit msg", "ipfs://CID", fmt.Sprintf(`500000000%s`, wormchain.Config().Denom))
	require.NoError(t, err, "error building proposal")

	// First attempt (should fail because user2 is not allowed to submit proposals)
	_, err = wormchain.SubmitProposal(ctx, user2.FormattedAddress(), proposalDraft)
	require.Error(t, err, "expected error submitting proposal")

	// Second attempt (should succeed because user1 is allowed to submit proposals)
	txProp, err := wormchain.SubmitProposal(ctx, user1.FormattedAddress(), proposalDraft)
	t.Log("txProp", txProp)
	require.NoError(t, err, "error submitting proposal")

	// Get height after proposal submission
	height, _ := wormchain.Height(ctx)

	proposalID, err := strconv.ParseInt(txProp.ProposalID, 10, 64)
	require.NoError(t, err, "failed to parse proposal ID")

	// Force all validators vote on proposal
	err = wormchain.VoteOnProposalAllValidators(ctx, proposalID, cosmos.ProposalVoteYes)
	require.NoError(t, err, "failed to submit votes")

	// Poll for proposal status to change to passed
	proposal, err := cosmos.PollForProposalStatus(ctx, wormchain, height, height+haltHeightDelta, proposalID, govv1beta1.StatusPassed)
	require.NoError(t, err, "proposal status did not change to passed in expected number of blocks")
	require.NotEmpty(t, proposal, "proposal not found")
}
