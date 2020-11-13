package msg

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ Msg = Send{}

// SendData - value part of Send
type SendData struct {
	FromAddress AccAddress `json:"from_address"`
	ToAddress   AccAddress `json:"to_address"`
	Amount      sdk.Coins  `json:"amount"`
}

// Send - high level transaction of the bank module
type Send struct {
	Type  string   `json:"type"`
	Value SendData `json:"value"`
}

// NewSend - create Send
func NewSend(fromAddr, toAddr AccAddress, amount sdk.Coins) Send {
	return Send{
		Type: "bank/MsgSend",
		Value: SendData{
			FromAddress: fromAddr,
			ToAddress:   toAddr,
			Amount:      amount,
		},
	}
}

// GetType - Msg interface
func (m Send) GetType() string {
	return "send"
}

// GetSignBytes - Msg interface
func (m Send) GetSignBytes() []byte {
	bz, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return sdk.MustSortJSON(bz)
}

// GetSigners - Msg interface
func (m Send) GetSigners() []AccAddress {
	return []AccAddress{m.Value.FromAddress}
}

// GetSendCoins - return send coins for tax calculation
func (m Send) GetSendCoins() Coins {
	return m.Value.Amount
}
