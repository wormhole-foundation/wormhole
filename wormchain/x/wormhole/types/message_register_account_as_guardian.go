package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const TypeMsgRegisterAccountAsGuardian = "register_account_as_guardian"

var _ sdk.Msg = &MsgRegisterAccountAsGuardian{}

func NewMsgRegisterAccountAsGuardian(signer string, signature []byte) *MsgRegisterAccountAsGuardian {
	return &MsgRegisterAccountAsGuardian{
		Signer:    signer,
		Signature: signature,
	}
}

func (msg *MsgRegisterAccountAsGuardian) Route() string {
	return RouterKey
}

func (msg *MsgRegisterAccountAsGuardian) Type() string {
	return TypeMsgRegisterAccountAsGuardian
}

func (msg *MsgRegisterAccountAsGuardian) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgRegisterAccountAsGuardian) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgRegisterAccountAsGuardian) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}
	return nil
}
