package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgExecuteVAA{}

func NewMsgExecuteVAA(creator string, vaa []byte) *MsgExecuteVAA {
	return &MsgExecuteVAA{
		Creator: creator,
		Vaa:     vaa,
	}
}

func (msg *MsgExecuteVAA) Route() string {
	return RouterKey
}

func (msg *MsgExecuteVAA) Type() string {
	return "ExecuteVAA"
}

func (msg *MsgExecuteVAA) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgExecuteVAA) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgExecuteVAA) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
