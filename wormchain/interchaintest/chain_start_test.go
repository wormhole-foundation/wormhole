package interchaintest

import (
	"testing"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"
)

// TestChainStart asserts the chain will start with a single validator
func TestChainStart(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	t.Parallel()

	// Base setup
	chains := CreateThisBranchChain(t, 1, 0)
	ic, ctx, _, _ := BuildInitialChain(t, chains)
	require.NotNil(t, ic)
	require.NotNil(t, ctx)

	// Confirm 10 blocks are produced
	chain := chains[0].(*cosmos.CosmosChain)
	testutil.WaitForBlocks(ctx, 10, chain)

	t.Cleanup(func() {
		_ = ic.Close()
	})
}
