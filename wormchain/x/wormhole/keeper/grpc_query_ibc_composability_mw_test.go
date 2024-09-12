package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// TestQueryIbcComposabilityMwContract tests querying of the IbcComposabilityMwContract
func TestQueryIbcComposabilityMwContract(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)

	// Invalid query with nil request
	_, err := k.IbcComposabilityMwContract(ctx, nil)
	require.Error(t, err)

	// Query when no contract is set
	res, err := k.IbcComposabilityMwContract(ctx, &types.QueryIbcComposabilityMwContractRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, "", res.ContractAddress)

	// Set the contract in state store
	contractAddr := "wormhole1du4amsmvx8yqr8whw7qc5m3c0zpwknmzelwqy6"
	k.StoreIbcComposabilityMwContract(ctx, types.IbcComposabilityMwContract{
		ContractAddress: contractAddr,
	})

	// Query IbcComposabilityMwContract
	res, err = k.IbcComposabilityMwContract(ctx, &types.QueryIbcComposabilityMwContractRequest{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, contractAddr, res.ContractAddress)
}
