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

// disallow hot-swapping validator addresses when guardian set size is >1
func TestRegisterAccountAsGuardianBlockHotSwap(t *testing.T) {
	// setup -- create guardian set of size 2
	k, ctx := keepertest.WormholeKeeper(t)
	guardians, privateKeys := createNGuardianValidator(k, ctx, 2)
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

	// assert that we are unable to associate the guardian address with a new validator address when the set size is >1
	_, err = msgServer.RegisterAccountAsGuardian(context, &types.MsgRegisterAccountAsGuardian{
		Signer:    newValAddr.String(),
		Signature: sig,
	})
	assert.Error(t, types.ErrConsensusSetNotUpdatable, err)
}
