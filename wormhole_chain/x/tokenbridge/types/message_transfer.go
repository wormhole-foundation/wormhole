package types

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgTransfer{}

func NewMsgTransfer(creator string, amount sdk.Coin, toChain uint16, toAddress []byte, fee *big.Int) *MsgTransfer {
	return &MsgTransfer{
		Creator:   creator,
		Amount:    amount,
		ToChain:   uint32(toChain),
		ToAddress: toAddress,
		Fee:       fee.String(),
	}
}

func (msg *MsgTransfer) Route() string {
	return RouterKey
}

func (msg *MsgTransfer) Type() string {
	return "Transfer"
}

func (msg *MsgTransfer) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

func (msg *MsgTransfer) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(msg)
	return sdk.MustSortJSON(bz)
}

func (msg *MsgTransfer) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidAddress, "invalid creator address (%s)", err)
	}

	err = msg.Amount.Validate()
	if err != nil {
		return err
	}

	if len(msg.ToAddress) != 32 {
		return ErrInvalidToAddress
	}

	if _, ok := new(big.Int).SetString(msg.Fee, 10); !ok {
		return ErrInvalidFee
	}

	return nil
}
