package keeper

import (
	"bytes"
	"context"

	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/wormhole-foundation/wormchain/x/wormhole/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"golang.org/x/crypto/sha3"
)

var WASMD_CONTRACT_ADMIN = sdk.AccAddress("wormchain_wasmd_owner")

// Simple wrapper of x/wasmd StoreCode that requires a VAA
func (k msgServer) StoreCode(goCtx context.Context, msg *types.MsgStoreCode) (*types.MsgStoreCodeResponse, error) {
	if !k.setWasmd {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrNotSupported, "x/wasmd not set")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}
	// Verify VAA
	action, payload, err := k.VerifyGovernanceVAA(ctx, v, vaa.WasmdModule)
	if err != nil {
		return nil, err
	}

	if vaa.GovernanceAction(action) != vaa.ActionStoreCode {
		return nil, types.ErrUnknownGovernanceAction
	}

	// verify payload is the sha3 256 hash of the wasm binary being uploaded
	var expected_hash [32]byte
	keccak := sha3.NewLegacyKeccak256()
	keccak.Write(msg.WASMByteCode)
	keccak.Sum(expected_hash[:0])
	if !bytes.Equal(payload, expected_hash[:]) {
		return nil, types.ErrInvalidHash
	}
	// Execute StoreCode normally
	senderAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "signer")
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(sdk.AttributeKeySender, msg.Signer),
	))
	codeID, chksum, err := k.wasmdKeeper.Create(ctx, senderAddr, msg.WASMByteCode, &wasmdtypes.DefaultUploadAccess)
	if err != nil {
		return nil, err
	}
	return &types.MsgStoreCodeResponse{
		CodeID:   codeID,
		Checksum: chksum,
	}, nil
}

// Simple wrapper of x/wasmd InstantiateContract that requires a VAA
func (k msgServer) InstantiateContract(goCtx context.Context, msg *types.MsgInstantiateContract) (*types.MsgInstantiateContractResponse, error) {
	if !k.setWasmd {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrNotSupported, "x/wasmd not set")
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}

	// Verify VAA
	action, payload, err := k.VerifyGovernanceVAA(ctx, v, vaa.WasmdModule)
	if err != nil {
		return nil, err
	}

	if vaa.GovernanceAction(action) != vaa.ActionInstantiateContract {
		return nil, types.ErrUnknownGovernanceAction
	}

	// Need to verify the instantiation arguments
	// The vaa governance payload must contain the hash of the expected args.
	expected_hash := vaa.CreateInstatiateCosmwasmContractHash(msg.CodeID, msg.Label, msg.Msg)
	if !bytes.Equal(payload, expected_hash[:]) {
		return nil, types.ErrInvalidHash
	}

	// Execute Instantiate normally
	senderAddr, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "signer")
	}
	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(sdk.AttributeKeySender, msg.Signer),
	))
	contract_addr, data, err := k.wasmdKeeper.Instantiate(ctx, msg.CodeID, senderAddr, WASMD_CONTRACT_ADMIN, msg.Msg, msg.Label, sdk.Coins{})
	if err != nil {
		return nil, err
	}
	return &types.MsgInstantiateContractResponse{
		Address: contract_addr.String(),
		Data:    data,
	}, nil
}

func (k msgServer) MigrateContract(goCtx context.Context, msg *types.MsgMigrateContract) (*types.MsgMigrateContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// Parse VAA
	v, err := ParseVAA(msg.Vaa)
	if err != nil {
		return nil, err
	}

	// Verify VAA
	action, payload, err := k.VerifyGovernanceVAA(ctx, v, vaa.WasmdModule)
	if err != nil {
		return nil, err
	}

	if vaa.GovernanceAction(action) != vaa.ActionMigrateContract {
		return nil, types.ErrUnknownGovernanceAction
	}

	// Need to verify the instantiation arguments
	// The vaa governance payload must contain the hash of the expected args.
	expected_hash := vaa.CreateMigrateCosmwasmContractHash(msg.CodeID, msg.Contract, msg.Msg)
	if !bytes.Equal(payload, expected_hash[:]) {
		return nil, types.ErrInvalidHash
	}

	_, err = sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "contract")
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(sdk.AttributeKeySender, msg.Signer),
	))

	data, err := k.wasmdKeeper.Migrate(ctx, contractAddr, WASMD_CONTRACT_ADMIN, msg.CodeID, msg.Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgMigrateContractResponse{
		Data: data,
	}, nil
}
