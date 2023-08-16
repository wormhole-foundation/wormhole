package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgExecuteGatewayGovernanceVaa{}

func (msg *MsgExecuteGatewayGovernanceVaa) Route() string {
	return RouterKey
}

func (msg *MsgExecuteGatewayGovernanceVaa) Type() string {
	return "MsgExecuteGatewayGovernanceVaa"
}

func (msg *MsgExecuteGatewayGovernanceVaa) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgExecuteGatewayGovernanceVaa) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgExecuteGatewayGovernanceVaa) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid signer address (%s)", err)
	}

	return nil
}
