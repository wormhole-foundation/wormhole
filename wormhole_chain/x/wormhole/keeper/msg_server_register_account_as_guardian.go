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
	// TODO(csongor): I think it would actually be better if this wasn't an
	// explicit parameter. What are the possible mistakes? A guardian submits
	// the registration tx from the wrong machine?
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

	if err != nil {
		return nil, err
	}

	// ecrecover gave us a 65-byte public key, which we first need to
	// convert to a 20 byte ethereum-style address. The first byte of the
	// public key is just the prefix byte '0x04' which we drop first. Then
	// hash the public key, and take the last 20 bytes of the hash
	// (according to
	// https://ethereum.org/en/developers/docs/accounts/#account-creation)
	guardianKeyAddrFromSignature := common.BytesToAddress(crypto.Keccak256(guardianKey[1:])[12:])
	guardianKeyAddr := common.BytesToAddress(msg.GuardianPubkey.Key)

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

	if !latestGuardianSet.ContainsKey(guardianKeyAddr) {
		return nil, types.ErrGuardianNotFound
	}

	// register validator in store for guardian
	k.Keeper.SetGuardianValidator(ctx, types.GuardianValidator{
		GuardianKey:   guardianKey,
		ValidatorAddr: claimedSigner,
	})

	err = ctx.EventManager().EmitTypedEvent(&types.EventGuardianRegistered{
		GuardianKey:  guardianKey,
		ValidatorKey: claimedSigner.Bytes(),
	})

	if err != nil {
		return nil, err
	}

	k.Keeper.TrySwitchToNewConsensusGuardianSet(ctx)

	return &types.MsgRegisterAccountAsGuardianResponse{}, nil
}
