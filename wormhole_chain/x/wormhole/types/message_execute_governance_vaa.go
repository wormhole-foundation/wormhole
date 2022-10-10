package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgExecuteGovernanceVAA{}

func NewMsgExecuteGovernanceVAA(vaa []byte, signer string) *MsgExecuteGovernanceVAA {
	return &MsgExecuteGovernanceVAA{
		Vaa:    vaa,
		Signer: signer,
	}
}

func (msg *MsgExecuteGovernanceVAA) Route() string {
	return RouterKey
}

func (msg *MsgExecuteGovernanceVAA) Type() string {
	return "ExecuteGovernanceVAA"
}

func (msg *MsgExecuteGovernanceVAA) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgExecuteGovernanceVAA) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgExecuteGovernanceVAA) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "string could not be parsed as address: %v", err)
	}

	return nil
}
