package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const (
	WormholeAddress1 = "wormhole1du4amsmvx8yqr8whw7qc5m3c0zpwknmzelwqy6"
	WormholeAddress2 = "wormhole13ztxpktzsng3ewkepe2w39ugxzfdf23teptu9n"
)

// TestAllowedAddressStore tests the setting, getting, and removing of allowed addresses.
func TestAllowedAddressStore(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)

	value := types.ValidatorAllowedAddress{
		ValidatorAddress: WormholeAddress1,
		AllowedAddress:   WormholeAddress2,
		Name:             "User1",
	}

	// Set validator allowed list
	k.SetValidatorAllowedAddress(ctx, value)

	// Check if address exists
	hasAddr := k.HasValidatorAllowedAddress(ctx, value.AllowedAddress)
	require.True(t, hasAddr)

	// Check faulty address - does not exist
	hasAddr = k.HasValidatorAllowedAddress(ctx, "invalid")
	require.False(t, hasAddr)

	// Retrieve & validate
	res := k.GetValidatorAllowedAddress(ctx, value.AllowedAddress)
	require.Equal(t, value.ValidatorAddress, res.ValidatorAddress)
	require.Equal(t, value.AllowedAddress, res.AllowedAddress)
	require.Equal(t, value.Name, res.Name)

	// Get all allowed addresses
	addrList := k.GetAllAllowedAddresses(ctx)
	require.Equal(t, 1, len(addrList))
	res = addrList[0]
	require.Equal(t, value.ValidatorAddress, res.ValidatorAddress)
	require.Equal(t, value.AllowedAddress, res.AllowedAddress)
	require.Equal(t, value.Name, res.Name)

	// Remove address
	k.RemoveValidatorAllowedAddress(ctx, value.AllowedAddress)

	// Check if address exists
	hasAddr = k.HasValidatorAllowedAddress(ctx, value.AllowedAddress)
	require.False(t, hasAddr)
}

// TestValidatorAsAllowedAddress tests if a validator is a guardian or future validator.
func TestValidatorAsAllowedAddress(t *testing.T) {
	k, ctx := keepertest.WormholeKeeper(t)

	// Create guardian set
	guardians, _ := createNGuardianValidator(k, ctx, 10)
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})

	createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{
		Index: 0,
	})

	// Get validator addr
	addr, err := sdk.Bech32ifyAddressBytes("wormhole", guardians[0].ValidatorAddr)
	require.NoError(t, err)

	// Check if validator belongs to a guardian
	_, found := k.GetGuardianValidatorByValidatorAddress(ctx, addr)
	require.True(t, found)

	// Check if validator is a current/future validator
	isVal := k.IsAddressValidatorOrFutureValidator(ctx, addr)
	require.True(t, isVal)

	// Check invalid addresses
	_, found = k.GetGuardianValidatorByValidatorAddress(ctx, WormholeAddress1)
	require.False(t, found)
	isVal = k.IsAddressValidatorOrFutureValidator(ctx, WormholeAddress1)
	require.False(t, isVal)
}
