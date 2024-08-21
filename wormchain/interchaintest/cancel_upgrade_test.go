package interchaintest

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
)

// TestCancelUpgrade will start on wormchain v2.18.1.1, schedule an upgrade to v2.23.0, cancel the upgrade,
// and verify that block production does not stop at the cancelled scheduled upgrade height.
func TestCancelUpgrade(t *testing.T) {
	// Base setup
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)

	chains := CreateChain(t, *guardians, ibc.DockerImage{
		Repository: WormchainRemoteRepo,
		Version:    "v2.24.3.2",
		UidGid:     WormchainImage.UidGid,
	})

	wormchain := chains[0].(*cosmos.CosmosChain)

	_, ctx, _, _, _, _ := BuildInterchain(t, chains)

	// Set up upgrade
	blocksAfterUpgrade := uint64(10)
	height, err := wormchain.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade")
	fmt.Println("Height at sending schedule upgrade: ", height)

	haltHeight := uint64(height) + blocksAfterUpgrade
	fmt.Println("Height for scheduled upgrade: ", haltHeight)

	// Schedule upgrade
	helpers.ScheduleUpgrade(t, ctx, wormchain, "faucet", "v3.0.0", haltHeight, guardians)

	// Cancel upgrade
	testutil.WaitForBlocks(ctx, 2, wormchain)
	helpers.CancelUpgrade(t, ctx, wormchain, "faucet", guardians)

	timeoutCtx3, timeoutCtxCancel3 := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel3()

	// Wait for chain to reach/exceed originally scheduled upgrade height
	// If it times-out, the cancel upgrade did not work and the chain will have halted at the scheduled upgrade height
	// If it does not timeout, it will be one block after the originally scheduled upgrade height
	testutil.WaitForBlocks(timeoutCtx3, int(blocksAfterUpgrade), wormchain)

	height, err = wormchain.Height(ctx)
	require.NoError(t, err, "error fetching height after chain should have halted")

	// Ensure that the chain continued making blocks passed the upgrade height
	require.NotEqual(t, haltHeight, height, "height is equal to halt height, it shouldn't be")
	fmt.Println("***** Cancel upgrade test passed ******")
}
