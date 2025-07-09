package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// TestIbcComposabilityMwContractStore tests the setting and getting of the contract.
func TestIbcComposabilityMwContractStore(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)

	// Get contract, should be nil
	res := k.GetIbcComposabilityMwContract(ctx)
	require.Equal(t, "", res.ContractAddress)

	// Set the contract
	contract := types.IbcComposabilityMwContract{
		ContractAddress: "contractAddress",
	}
	k.StoreIbcComposabilityMwContract(ctx, contract)

	// Get contract from store
	res = k.GetIbcComposabilityMwContract(ctx)
	require.NotNil(t, res)
	require.Equal(t, contract.ContractAddress, res.ContractAddress)
}
