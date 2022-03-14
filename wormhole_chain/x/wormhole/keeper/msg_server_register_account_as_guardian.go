package keeper

import (
	"bytes"
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/certusone/wormhole-chain/x/wormhole/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// TODO(csongor): high-level overview of what this does
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

	k.Keeper.TrySwitchToNewConsensusGuardianSet(ctx)

	return &types.MsgRegisterAccountAsGuardianResponse{}, nil
}
