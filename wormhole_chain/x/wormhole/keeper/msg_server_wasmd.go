package keeper

import (
	"bytes"
	"context"
	"encoding/binary"

	wasmdkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
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
	action, payload, err := k.VerifyVAAGovernance(ctx, v, WasmdModule)
	if err != nil {
		return nil, err
	}

	if GovernanceAction(action) != ActionStoreCode {
		return nil, types.ErrUnknownGovernanceAction
	}

	// verify payload is the sha3 256 hash of the wasm binary being uploaded
	expected_hash := sha3.Sum256(msg.WASMByteCode)
	if !bytes.Equal(payload, expected_hash[:]) {
		return nil, types.ErrInvalidHash
	}

	// Execute StoreCore normally
	permissionedKeeper := wasmdkeeper.NewDefaultPermissionKeeper(k.wasmdKeeper)
	msgServer := wasmdkeeper.NewMsgServerImpl(permissionedKeeper)
	req := msg.ToWasmd()
	res, err := msgServer.StoreCode(goCtx, &req)
	if err != nil {
		return nil, err
	}
	return &types.MsgStoreCodeResponse{
		CodeID: res.CodeID,
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
	action, payload, err := k.VerifyVAAGovernance(ctx, v, WasmdModule)
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
	expected_hash := sha3.Sum256(hash_base)
	if !bytes.Equal(payload, expected_hash[:]) {
		return nil, types.ErrInvalidHash
	}

	permissionedKeeper := wasmdkeeper.NewDefaultPermissionKeeper(k.wasmdKeeper)
	msgServer := wasmdkeeper.NewMsgServerImpl(permissionedKeeper)
	req := msg.ToWasmd()
	res, err := msgServer.InstantiateContract(goCtx, &req)
	if err != nil {
		return nil, err
	}
	return &types.MsgInstantiateContractResponse{
		Address: res.Address,
		Data:    res.Data,
	}, nil
}
