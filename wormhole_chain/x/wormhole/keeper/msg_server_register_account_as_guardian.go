package keeper

import (
	"bytes"
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) RegisterAccountAsGuardian(goCtx context.Context, msg *types.MsgRegisterAccountAsGuardian) (*types.MsgRegisterAccountAsGuardianResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// verify that the signer provided their own public key.  This wouldn't
	// strictly be necessary (can just use the signer directly), but since it's
	// this key that gets signed, it's easier to report here if there's a
	// mistake.
	claimedSigner, err := sdk.ValAddressFromBech32(msg.AddressBech32)
	if err != nil {
		return nil, err
	}

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(claimedSigner, signer) {
		return nil, types.ErrSignerMismatch
	}

	// recover guardian key from signature
	signerHash := crypto.Keccak256Hash(signer)
	guardianKey, err := crypto.Ecrecover(signerHash.Bytes(), msg.Signature)

	guardianKeyAddrFromSignature := common.BytesToAddress(crypto.Keccak256(guardianKey[1:])[12:])
	guardianKeyAddr := common.BytesToAddress(crypto.Keccak256(msg.GuardianPubkey.Key[1:])[12:])

	// check the recovered guardian key matches the one in the message
	if guardianKeyAddrFromSignature != guardianKeyAddr {
		return nil, types.ErrGuardianSignatureMismatch
	}

	// next we check if this guardian key is in the most recent guardian set.
	// we don't allow registration of arbitrary public keys, since that would
	// enable a DoS vector
	latestGuardianSetIndex := k.Keeper.GetLatestGuardianSetIndex(ctx)
	latestGuardianSet, found := k.Keeper.GetGuardianSet(ctx, latestGuardianSetIndex)

	if !found {
		return nil, types.ErrGuardianSetNotFound
	}

	// TODO: should we just take an index and look up in the list, instead of
	// iterating through?
	contains := false
	for _, key := range latestGuardianSet.Keys {
		bytes.Equal(guardianKeyAddr.Bytes(), key)
		contains = true
		break
	}

	if !contains {
		return nil, types.ErrGuardianNotFound
	}

	// TODO(csongor): implement k.GetValidatorByGuardianAddr/SetValidatorByGuardianAddr

	// register validator in store for guardian
	k.Keeper.SetGuardianValidator(ctx, types.GuardianValidator{
		GuardianKey:   guardianKey,
		ValidatorAddr: claimedSigner,
	})

	// call the after-registration hook
	// TODO(csongor): k.AfterGuardianRegistered(ctx, ...)

	// TODO(csongor): register guardian in store
	_ = ctx

	// TODO(csongor): emit event about guardian registration

	k.Keeper.tryNewConsensusGuardianSet(ctx)

	return &types.MsgRegisterAccountAsGuardianResponse{}, nil
}

// TODO(csongor): call this when a new guardian set is added (maybe move the function?)
func (k Keeper) tryNewConsensusGuardianSet(ctx sdk.Context) error {
	latestGuardianSetIndex := k.GetLatestGuardianSetIndex(ctx)
	consensusGuardianSetIndex, found := k.GetActiveGuardianSetIndex(ctx)
	if !found {
		return types.ErrGuardianNotFound
	}

	// nothing to do if the latest set is already the consensus set
	if latestGuardianSetIndex == consensusGuardianSetIndex.Index {
		return nil
	}

	consensusGuardianSet, found := k.GetGuardianSet(ctx, consensusGuardianSetIndex.Index)
	if !found {
		return types.ErrGuardianNotFound
	}

	// count how many registrations we have
	registered := 0
	for _, key := range consensusGuardianSet.Keys {
		_, found := k.GetGuardianValidator(ctx, key)
		if found {
			registered++
		}
	}

	// see if we have enough validators registered to produce blocks.
	// TODO(csongor): this has to be kept in sync with tendermint consensus
	quorum := CalculateQuorum(len(consensusGuardianSet.Keys))
	if registered >= quorum {
		// we have enough, set consensus set to the latest one. Guardian set upgrade complete.
		k.SetActiveGuardianSetIndex(ctx, types.ActiveGuardianSetIndex{
			Index: latestGuardianSetIndex,
		})
	}

	return nil
}
