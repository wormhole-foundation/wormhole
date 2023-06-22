package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgInstantiateContract{}
var _ sdk.Msg = &MsgStoreCode{}

func (msg *MsgInstantiateContract) Route() string {
	return RouterKey
}

func (msg *MsgInstantiateContract) Type() string {
	return "InstantiateContract"
}

func (msg *MsgInstantiateContract) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgInstantiateContract) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgInstantiateContract) ValidateBasic() error {
	return msg.ToWasmd().ValidateBasic()
}

func (msg *MsgStoreCode) Route() string {
	return RouterKey
}

func (msg *MsgStoreCode) Type() string {
	return "StoreCode"
}

func (msg *MsgStoreCode) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgStoreCode) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgStoreCode) ValidateBasic() error {
	return msg.ToWasmd().ValidateBasic()
}

func (msg *MsgMigrateContract) Route() string {
	return RouterKey
}

func (msg *MsgMigrateContract) Type() string {
	return "MigrateContract"
}

func (msg *MsgMigrateContract) GetSigners() []sdk.AccAddress {
	signer, err := sdk.AccAddressFromBech32(msg.Signer)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{signer}
}

func (msg *MsgMigrateContract) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgMigrateContract) ValidateBasic() error {
	return msg.ToWasmd().ValidateBasic()
}
