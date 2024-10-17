package interchaintest

import (
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
)

// TestChainStart asserts the chain will start with a single validator
func TestChainStart(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	// Base setup
	guardians := guardians.CreateValSet(t, 2)
	chains := CreateLocalChain(t, *guardians)
	ic, ctx, _, _, _, _ := BuildInterchain(t, chains)
	require.NotNil(t, ic)
	require.NotNil(t, ctx)

	// Confirm 5 blocks are produced
	chain := chains[0].(*cosmos.CosmosChain)
	testutil.WaitForBlocks(ctx, 5, chain)
}
