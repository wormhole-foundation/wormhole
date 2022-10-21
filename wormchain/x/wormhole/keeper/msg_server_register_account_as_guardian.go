package keeper

import (
	"bytes"
	"context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
)

// TODO(csongor): high-level overview of what this does
func (k msgServer) RegisterAccountAsGuardian(goCtx context.Context, msg *types.MsgRegisterAccountAsGuardian) (*types.MsgRegisterAccountAsGuardianResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, err
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
	consensusGuardianSetIndex, found := k.GetConsensusGuardianSetIndex(ctx)

	if found && latestGuardianSetIndex == consensusGuardianSetIndex.Index {
		return nil, types.ErrConsensusSetNotUpdatable
	}

	latestGuardianSet, found := k.Keeper.GetGuardianSet(ctx, latestGuardianSetIndex)

	if !found {
		return nil, types.ErrGuardianSetNotFound
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
