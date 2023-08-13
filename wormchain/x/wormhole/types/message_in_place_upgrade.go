package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgInPlaceUpgrade{}

func (msg *MsgInPlaceUpgrade) Route() string {
	return RouterKey
}

func (msg *MsgInPlaceUpgrade) Type() string {
	return "CreateAllowlistEntryRequest"
}

func (msg *MsgInPlaceUpgrade) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgInPlaceUpgrade) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgInPlaceUpgrade) ValidateBasic() error {
	return nil
}
