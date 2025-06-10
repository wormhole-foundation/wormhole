package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// TestWasmInstantiateAllowlist tests the setting, getting, and removing of allowed addresses.
func TestWasmInstantiateAllowlist(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)

	// Create entry
	entry := types.WasmInstantiateAllowedContractCodeId{
		ContractAddress: WormholeContractAddress1,
		CodeId:          1,
	}

	// Add contract to allow list
	k.SetWasmInstantiateAllowlist(ctx, entry)

	// Check if address exists
	hasAddr := k.HasWasmInstantiateAllowlist(ctx, entry.ContractAddress, entry.CodeId)
	require.True(t, hasAddr)

	// Check faulty address - does not exist
	hasAddr = k.HasWasmInstantiateAllowlist(ctx, "invalid", 0)
	require.False(t, hasAddr)

	// Get all allowed addresses
	addrList := k.GetAllWasmInstantiateAllowedAddresses(ctx)
	require.Equal(t, 1, len(addrList))
	require.Equal(t, entry.ContractAddress, addrList[0].ContractAddress)
	require.Equal(t, entry.CodeId, addrList[0].CodeId)

	// Remove address
	k.KeeperDeleteWasmInstantiateAllowlist(ctx, entry)

	// Check if address exists
	hasAddr = k.HasWasmInstantiateAllowlist(ctx, entry.ContractAddress, entry.CodeId)
	require.False(t, hasAddr)
}
