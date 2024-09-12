package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// TestQueryAllowlist tests the allow list queries
func TestQueryAllowlist(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)

	// Check if no allowlist exists
	res, err := k.AllowlistAll(ctx, &types.QueryAllValidatorAllowlist{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, 0, len(res.Allowlist))

	value := types.ValidatorAllowedAddress{
		ValidatorAddress: "wormhole1du4amsmvx8yqr8whw7qc5m3c0zpwknmzelwqy6",
		AllowedAddress:   "wormhole13ztxpktzsng3ewkepe2w39ugxzfdf23teptu9n",
		Name:             "User1",
	}

	// Set validator allowed list
	k.SetValidatorAllowedAddress(ctx, value)

	// Query all allow lists
	res, err = k.AllowlistAll(ctx, &types.QueryAllValidatorAllowlist{})
	require.NoError(t, err)
	require.NotNil(t, res)
	require.Equal(t, 1, len(res.Allowlist))
	require.Equal(t, value.ValidatorAddress, res.Allowlist[0].ValidatorAddress)
	require.Equal(t, value.AllowedAddress, res.Allowlist[0].AllowedAddress)
	require.Equal(t, value.Name, res.Allowlist[0].Name)

	// Invalid query all
	_, err = k.Allowlist(ctx, nil)
	require.Error(t, err)

	// Query allow list by address
	res2, err := k.Allowlist(ctx, &types.QueryValidatorAllowlist{
		ValidatorAddress: value.ValidatorAddress,
	})
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.Equal(t, 1, len(res2.Allowlist))
	require.Equal(t, value.ValidatorAddress, res2.Allowlist[0].ValidatorAddress)
	require.Equal(t, value.AllowedAddress, res2.Allowlist[0].AllowedAddress)

	// Query with nil request
	_, err = k.Allowlist(ctx, nil)
	require.Error(t, err)

	// Query invalid address
	res2, err = k.Allowlist(ctx, &types.QueryValidatorAllowlist{
		ValidatorAddress: "invalid",
	})
	require.NoError(t, err)
	require.NotNil(t, res2)
	require.Equal(t, 0, len(res2.Allowlist))
}
