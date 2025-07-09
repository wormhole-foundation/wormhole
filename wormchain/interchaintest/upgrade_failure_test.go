package ictest

import (
	"context"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
)

// TestUpgradeFailure starts wormchain on v2.24.3.2, then attempts to re-start with version v3.0.0 by bypassing the upgrade handler logic.
// Test will fail because it did not successfully migrate stores properly.
func TestUpgradeFailure(t *testing.T) {
	// Base setup
	numVals := 5
	guardians := guardians.CreateValSet(t, numVals)

	chains := CreateChain(t, *guardians, ibc.DockerImage{
		Repository: WormchainRemoteRepo,
		Version:    "v2.24.3.2",
		UidGid:     WormchainImage.UidGid,
	})

	wormchain := chains[0].(*cosmos.CosmosChain)

	_, ctx, _, _, client, _ := BuildInterchain(t, chains)

	blocksAfterUpgrade := uint64(5)

	err := wormchain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	// upgrade version on all nodes
	wormchain.UpgradeVersion(ctx, client, WormchainLocalRepo, WormchainLocalVersion)

	err = wormchain.StartAllNodes(ctx)
	require.Error(t, err)

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*20)
	defer timeoutCtxCancel()

	// Wait for blocks
	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), wormchain)
	require.Error(t, err)

	// Get current height
	_, err = wormchain.Height(ctx)
	require.Error(t, err)

	testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), wormchain)

	_, err = wormchain.Height(ctx)
	require.Error(t, err)
}
