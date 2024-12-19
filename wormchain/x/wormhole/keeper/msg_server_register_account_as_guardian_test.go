package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	keepertest "github.com/wormhole-foundation/wormchain/testutil/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/keeper"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	wormholesdk "github.com/wormhole-foundation/wormhole/sdk"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// hot-swap validator address when guardian set size is 1 (for testnets and local devnets)
func TestRegisterAccountAsGuardianHotSwap(t *testing.T) {
	// setup -- create guardian set of size 1
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 1)
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})
	newValAddr_bz := [20]byte{}
	newValAddr := sdk.AccAddress(newValAddr_bz[:])

	set := createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{Index: set.Index})

	// execute msg_server function to associate new validator address account with the guardian address
	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// sign the new validator address as the new validator address
	addrHash := crypto.Keccak256Hash(wormholesdk.SignedWormchainAddressPrefix, newValAddr)
	sig, err := crypto.Sign(addrHash[:], privateKeys[0])
	require.NoErrorf(t, err, "failed to sign wormchain address: %v", err)

	_, err = msgServer.RegisterAccountAsGuardian(context, &types.MsgRegisterAccountAsGuardian{
		Signer:    newValAddr.String(),
		Signature: sig,
	})
	require.NoError(t, err)

	// assert that the guardian validator has the new validator address
	newGuardian, newGuardianFound := k.GetGuardianValidator(ctx, guardians[0].GuardianKey)
	require.Truef(t, newGuardianFound, "expected guardian not found in the keeper store")

	assert.Equal(t, newValAddr.Bytes(), newGuardian.ValidatorAddr)
}

// test hot swapping with validator size > 1
func TestRegisterAccountAsGuardianHotSwapMultipleValidators(t *testing.T) {
	// setup -- create guardian set of size 2
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 2)
	k.SetConfig(ctx, types.Config{
		GovernanceEmitter:     vaa.GovernanceEmitter[:],
		GovernanceChain:       uint32(vaa.GovernanceChain),
		ChainId:               uint32(vaa.ChainIDWormchain),
		GuardianSetExpiration: 86400,
	})

	set := createNewGuardianSet(k, ctx, guardians)
	k.SetConsensusGuardianSetIndex(ctx, types.ConsensusGuardianSetIndex{Index: set.Index})

	// execute msg_server function to associate new validator address account with the guardian address
	context := sdk.WrapSDKContext(ctx)
	msgServer := keeper.NewMsgServerImpl(*k)

	// store old val addr for later

	oldValAddr_bz := [20]byte{}
	copy(oldValAddr_bz[:], guardians[0].ValidatorAddr)
	oldValAddr := sdk.AccAddress(oldValAddr_bz[:])

	// hot swap to new val addr

	newValAddr_bz := [20]byte{}
	newValAddr := sdk.AccAddress(newValAddr_bz[:])

	// sign the new validator address as the new validator address
	addrHash := crypto.Keccak256Hash(wormholesdk.SignedWormchainAddressPrefix, newValAddr)
	sig, err := crypto.Sign(addrHash[:], privateKeys[0])
	require.NoErrorf(t, err, "failed to sign wormchain address: %v", err)

	// assert we can hot swap when validators > 1
	_, err = msgServer.RegisterAccountAsGuardian(context, &types.MsgRegisterAccountAsGuardian{
		Signer:    newValAddr.String(),
		Signature: sig,
	})
	assert.NoError(t, err)

	// assert that the guardian validator has the new validator address
	newGuardian, newGuardianFound := k.GetGuardianValidator(ctx, guardians[0].GuardianKey)
	require.Truef(t, newGuardianFound, "expected guardian not found in the keeper store")
	assert.Equal(t, newValAddr.Bytes(), newGuardian.ValidatorAddr)

	// -- hot swap back to old val addr --

	// sign the old validator address as the new validator address
	addrHash = crypto.Keccak256Hash(wormholesdk.SignedWormchainAddressPrefix, oldValAddr)
	sig, err = crypto.Sign(addrHash[:], privateKeys[0])
	require.NoErrorf(t, err, "failed to sign wormchain address: %v", err)

	// assert we can hot swap back to the old validator address
	_, err = msgServer.RegisterAccountAsGuardian(context, &types.MsgRegisterAccountAsGuardian{
		Signer:    oldValAddr.String(),
		Signature: sig,
	})
	assert.NoError(t, err)

	// assert that the guardian validator has the old validator address
	oldGuardian, oldGuardianFound := k.GetGuardianValidator(ctx, guardians[0].GuardianKey)
	require.Truef(t, oldGuardianFound, "expected guardian not found in the keeper store")
	assert.Equal(t, oldValAddr.Bytes(), oldGuardian.ValidatorAddr)
}
