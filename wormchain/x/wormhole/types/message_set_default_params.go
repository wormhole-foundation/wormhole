package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgSetTokenFactoryPfmDefaultParams{}

func (msg *MsgSetTokenFactoryPfmDefaultParams) Route() string {
	return RouterKey
}

func (msg *MsgSetTokenFactoryPfmDefaultParams) Type() string {
	return "CreateAllowlistEntryRequest"
}

func (msg *MsgSetTokenFactoryPfmDefaultParams) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgSetTokenFactoryPfmDefaultParams) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgSetTokenFactoryPfmDefaultParams) ValidateBasic() error {
	return nil
}
