package ictest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
)

// TestUpgradeFailure starts wormchain on v2.18.1, then attempts to upgrade 1 validator at a time to v2.23.0
func TestUpgradeFailure(t *testing.T) {
	// Base setup
	numVals := 5
	guardians := guardians.CreateValSet(t, numVals)
	chains := CreateChains(t, "v2.18.1", *guardians)
	ctx, _, _, client := BuildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)

	blocksAfterUpgrade := uint64(5)

	// upgrade version on all nodes
	wormchain.UpgradeVersion(ctx, client, "v2.23.0")

	for i := 0; i < numVals; i++ {
		haltHeight, err := wormchain.Height(ctx)
		require.NoError(t, err)
		fmt.Println("Halt height:", i, " : ", haltHeight)

		// bring down node to prepare for upgrade
		err = wormchain.StopANode(ctx, i)
		require.NoError(t, err, "error stopping node(s)")

		// start node back up with new binary
		err = wormchain.StartANode(ctx, i)
		require.NoError(t, err, "error starting upgraded node(s)")

		// Restart the fullnode with the last validator
		if i+1 == numVals {
			err = wormchain.StopANode(ctx, i+1)
			require.NoError(t, err, "error stopping node(s)")
			err = wormchain.StartANode(ctx, i+1)
			require.NoError(t, err, "error starting upgraded node(s)")
		}

		timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*20)
		defer timeoutCtxCancel()

		// Wait for 5 blocks (2sec/block) or 20 seconds
		testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), wormchain)
	}
	// Get current height
	height1, err := wormchain.Height(ctx)
	require.NoError(t, err)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*20)
	defer timeoutCtxCancel()

	// Wait for 5 blocks (2sec/block) or 20 seconds
	testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), wormchain)

	height2, err := wormchain.Height(ctx)
	require.NoError(t, err, "error fetching height after upgrade")
	fmt.Println("Checked height: ", height2)

	// height1 and height2 should be equal since we don't produce blocks with this upgrade path
	require.Equal(t, height1, height2, "height incremented after upgrade, so upgrade succeeded and test failed")
}
