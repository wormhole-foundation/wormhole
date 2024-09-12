package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// TestWasmInstantiateAllowlistAll tests the querying of the wasm instantiate allow list
func TestWasmInstantiateAllowlistAll(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)

	// Query with nil request
	_, err := k.WasmInstantiateAllowlistAll(ctx, nil)
	require.Error(t, err)

	// Query with no contracts
	res, err := k.WasmInstantiateAllowlistAll(ctx, &types.QueryAllWasmInstantiateAllowlist{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, 0, len(res.Allowlist))

	// Set contract in allow list
	contract := types.WasmInstantiateAllowedContractCodeId{
		ContractAddress: "wormhole1du4amsmvx8yqr8whw7qc5m3c0zpwknmzelwqy6",
		CodeId:          1,
	}
	k.SetWasmInstantiateAllowlist(ctx, contract)

	// Query all allow lists
	res, err = k.WasmInstantiateAllowlistAll(ctx, &types.QueryAllWasmInstantiateAllowlist{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, 1, len(res.Allowlist))
	require.Equal(t, contract.ContractAddress, res.Allowlist[0].ContractAddress)
	require.Equal(t, contract.CodeId, res.Allowlist[0].CodeId)
}
