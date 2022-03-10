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

	hash := crypto.Keccak256Hash(signer)
	guardianKey, err := crypto.Ecrecover(hash.Bytes(), msg.Signature)

	guardianKeyAddrFromSignature := common.BytesToAddress(crypto.Keccak256(guardianKey[1:])[12:])
	guardianKeyAddr := common.BytesToAddress(crypto.Keccak256(msg.GuardianPubkey.Key[1:])[12:])

	if guardianKeyAddrFromSignature != guardianKeyAddr {
		return nil, types.ErrGuardianSignatureMismatch
	}

	// TODO(csongor): check to see if the pubkey or sender has been registered before

	// TODO(csongor): implement k.GetValidatorByGuardianAddr/SetValidatorByGuardianAddr

	// call the after-registration hook
	// TODO(csongor): k.AfterGuardianRegistered(ctx, ...)

	// TODO(csongor): register guardian in store
	_ = ctx

	// TODO(csongor): emit event about guardian registration

	return &types.MsgRegisterAccountAsGuardianResponse{}, nil
}
