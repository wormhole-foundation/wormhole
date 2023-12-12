package keeper

import (
	"bytes"
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	wormholesdk "github.com/wormhole-foundation/wormhole/sdk"
)

// This function is used to onboard Wormhole Guardians as Validators on Wormchain.
// It creates a 1:1 association between a Guardian addresss and a Wormchain validator address.
// There is also a special case -- when the size of the Guardian set is 1, the Guardian is allowed to "hot-swap" their validator address in the mapping.
// We include the special case to make it easier to shuffle things in testnets and local devnets.
// 1. Guardian signs their validator address -- SIGNATURE=$(guardiand admin sign-wormchain-address <wormhole...>)
// 2. Guardian submits $SIGNATURE to Wormchain via this handler, using their new validator address as the signer of the Wormchain tx.
func (k msgServer) RegisterAccountAsGuardian(goCtx context.Context, msg *types.MsgRegisterAccountAsGuardian) (*types.MsgRegisterAccountAsGuardianResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, err
	}
	// recover guardian key from signature
	signerHash := crypto.Keccak256Hash(wormholesdk.SignedWormchainAddressPrefix, signer)
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
	guardianKeyAddr := common.BytesToAddress(crypto.Keccak256(guardianKey[1:])[12:])

	// next we check if this guardian key is in the most recent guardian set.
	// we don't allow registration of arbitrary public keys, since that would
	// enable a DoS vector
	latestGuardianSetIndex := k.Keeper.GetLatestGuardianSetIndex(ctx)
	latestGuardianSet, guardianSetFound := k.Keeper.GetGuardianSet(ctx, latestGuardianSetIndex)
	if !guardianSetFound {
		return nil, types.ErrGuardianSetNotFound
	}

	consensusGuardianSetIndex, consensusIndexFound := k.GetConsensusGuardianSetIndex(ctx)
	if !consensusIndexFound {
		return nil, types.ErrConsensusSetUndefined
	}

	// If the size of the guardian set is 1, allow hot-swapping the validator address.
	if consensusIndexFound && latestGuardianSetIndex == consensusGuardianSetIndex.Index && len(latestGuardianSet.Keys) > 1 {
		return nil, types.ErrConsensusSetNotUpdatable
	}

	if !latestGuardianSet.ContainsKey(guardianKeyAddr) {
		return nil, types.ErrGuardianNotFound
	}

	// Check if the tx signer was already registered as a guardian validator.
	for _, gv := range k.GetAllGuardianValidator(ctx) {
		if bytes.Equal(gv.ValidatorAddr, signer) {
			return nil, types.ErrSignerAlreadyRegistered
		}
	}

	// register validator in store for guardian
	k.Keeper.SetGuardianValidator(ctx, types.GuardianValidator{
		GuardianKey:   guardianKeyAddr.Bytes(),
		ValidatorAddr: signer,
	})

	err = ctx.EventManager().EmitTypedEvent(&types.EventGuardianRegistered{
		GuardianKey:  guardianKeyAddr.Bytes(),
		ValidatorKey: signer,
	})

	if err != nil {
		return nil, err
	}

	err = k.Keeper.TrySwitchToNewConsensusGuardianSet(ctx)

	if err != nil {
		return nil, err
	}

	return &types.MsgRegisterAccountAsGuardianResponse{}, nil
}
