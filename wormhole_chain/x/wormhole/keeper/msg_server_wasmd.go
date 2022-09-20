package keeper

import (
	"bytes"
	"context"
	"encoding/binary"

	wasmdtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/wormhole-foundation/wormhole-chain/x/wormhole/types"
	"golang.org/x/crypto/sha3"
)

// WasmdModule is the identifier of the Wasmd module (which is used for governance messages)
var WasmdModule = [32]byte{00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 00, 0x57, 0x61, 0x73, 0x6D, 0x64, 0x4D, 0x6F, 0x64, 0x75, 0x6C, 0x65}

var (
	ActionStoreCode           GovernanceAction = 1
	ActionInstantiateContract GovernanceAction = 2
)

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
	action, payload, err := k.VerifyGovernanceVAA(ctx, v, WasmdModule)
	if err != nil {
		return nil, err
	}

	if GovernanceAction(action) != ActionStoreCode {
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
	codeID, err := k.wasmdKeeper.Create(ctx, senderAddr, msg.WASMByteCode, &wasmdtypes.DefaultUploadAccess)
	if err != nil {
		return nil, err
	}
	return &types.MsgStoreCodeResponse{
		CodeID: codeID,
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
	action, payload, err := k.VerifyGovernanceVAA(ctx, v, WasmdModule)
	if err != nil {
		return nil, err
	}

	if GovernanceAction(action) != ActionInstantiateContract {
		return nil, types.ErrUnknownGovernanceAction
	}

	// Need to verify the msg contents by checking sha3.Sum(BigEndian(CodeID) || Label || Msg)
	// The vaa governance payload must contain this hash.
	hash_base := make([]byte, 8)
	binary.BigEndian.PutUint64(hash_base, msg.CodeID)
	hash_base = append(hash_base, []byte(msg.Label)...)
	hash_base = append(hash_base, msg.Msg...)

	var expected_hash [32]byte
	keccak := sha3.NewLegacyKeccak256()
	keccak.Write(hash_base)
	keccak.Sum(expected_hash[:0])

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
	contract_addr, data, err := k.wasmdKeeper.Instantiate(ctx, msg.CodeID, senderAddr, sdk.AccAddress{}, msg.Msg, msg.Label, sdk.Coins{})
	if err != nil {
		return nil, err
	}
	return &types.MsgInstantiateContractResponse{
		Address: contract_addr.String(),
		Data:    data,
	}, nil
}
