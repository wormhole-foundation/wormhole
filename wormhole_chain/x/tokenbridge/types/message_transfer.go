package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

var _ sdk.Msg = &MsgTransfer{}

func NewMsgTransfer(creator string, amount sdk.Coin, toChain uint16, toAddress []byte, fee sdk.Coin) *MsgTransfer {
	return &MsgTransfer{
		Creator:   creator,
		Amount:    amount,
		ToChain:   uint32(toChain),
		ToAddress: toAddress,
		Fee:       fee,
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

	if err := msg.Amount.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAmount, err)
	}

	if msg.ToChain > uint32(^uint16(0)) {
		return ErrInvalidTargetChain
	}

	if len(msg.ToAddress) != 32 {
		return ErrInvalidToAddress
	}

	if err := msg.Fee.Validate(); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidFee, err)
	}

	if msg.Amount.Denom != msg.Fee.Denom {
		return fmt.Errorf("%w: Fee must have the same denom as Amount", ErrInvalidFee)
	}

	if msg.Amount.Amount.BigInt().Cmp(msg.Fee.Amount.BigInt()) != 1 {
		return ErrFeeTooHigh
	}

	return nil
}
