package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgAddWasmInstantiateAllowlist{}
var _ sdk.Msg = &MsgDeleteWasmInstantiateAllowlist{}

func (msg *MsgAddWasmInstantiateAllowlist) Route() string {
	return RouterKey
}

func (msg *MsgAddWasmInstantiateAllowlist) Type() string {
	return "AddWasmInstantiateAllowlist"
}

func (msg *MsgAddWasmInstantiateAllowlist) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgAddWasmInstantiateAllowlist) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgAddWasmInstantiateAllowlist) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid allowlist address (%s)", err)
	}

	return nil
}

func (msg *MsgDeleteWasmInstantiateAllowlist) Route() string {
	return RouterKey
}

func (msg *MsgDeleteWasmInstantiateAllowlist) Type() string {
	return "DeleteWasmInstantiateAllowlist"
}

func (msg *MsgDeleteWasmInstantiateAllowlist) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgDeleteWasmInstantiateAllowlist) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgDeleteWasmInstantiateAllowlist) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}

	_, err = sdk.AccAddressFromBech32(msg.Address)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid allowlist address (%s)", err)
	}

	return nil
}
