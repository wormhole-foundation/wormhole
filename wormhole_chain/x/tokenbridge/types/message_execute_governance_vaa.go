package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgExecuteGovernanceVAA{}

func NewMsgExecuteGovernanceVAA(creator string, vaa []byte) *MsgExecuteGovernanceVAA {
	return &MsgExecuteGovernanceVAA{
		Creator: creator,
		Vaa:     vaa,
	}
}

func (msg *MsgExecuteGovernanceVAA) Route() string {
	return RouterKey
}

func (msg *MsgExecuteGovernanceVAA) Type() string {
	return "ExecuteGovernanceVAA"
}

func (msg *MsgExecuteGovernanceVAA) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgExecuteGovernanceVAA) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgExecuteGovernanceVAA) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}
	return nil
}
