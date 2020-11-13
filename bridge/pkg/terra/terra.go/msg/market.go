package msg

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ Msg = Swap{}
var _ Msg = SwapSend{}

// SwapData contains a swap request
type SwapData struct {
	Trader    AccAddress `json:"trader"`     // Address of the trader
	OfferCoin Coin       `json:"offer_coin"` // Coin being offered
	AskDenom  string     `json:"ask_denom"`  // Denom of the coin to swap to
}

// Swap - high level transaction of the wasm module
type Swap struct {
	Type  string   `json:"type"`
	Value SwapData `json:"value"`
}

// NewSwap - create Swap
func NewSwap(trader AccAddress, offerCoin Coin, askDenom string) Swap {
	return Swap{
		Type: "market/MsgSwap",
		Value: SwapData{
			Trader:    trader,
			OfferCoin: offerCoin,
			AskDenom:  askDenom,
		},
	}
}

// SwapSendData contains a swap request
type SwapSendData struct {
	FromAddress AccAddress `json:"from_address"` // Address of the offer coin payer
	ToAddress   AccAddress `json:"to_address"`   // Address of the recipient
	OfferCoin   Coin       `json:"offer_coin"`   // Coin being offered
	AskDenom    string     `json:"ask_denom"`    // Denom of the coin to swap to
}

// SwapSend - high level transaction of the wasm module
type SwapSend struct {
	Type  string       `json:"type"`
	Value SwapSendData `json:"value"`
}

// NewSwapSend - create SwapSend
func NewSwapSend(fromAddress, toAddress AccAddress, offerCoin Coin, askDenom string) SwapSend {
	return SwapSend{
		Type: "market/MsgSwapSend",
		Value: SwapSendData{
			FromAddress: fromAddress,
			ToAddress:   toAddress,
			OfferCoin:   offerCoin,
			AskDenom:    askDenom,
		},
	}
}

// GetType - Msg interface
func (m Swap) GetType() string {
	return "swap"
}

// GetSignBytes - Msg interface
func (m Swap) GetSignBytes() []byte {
	bz, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return sdk.MustSortJSON(bz)
}

// GetSigners - Msg interface
func (m Swap) GetSigners() []AccAddress {
	return []AccAddress{m.Value.Trader}
}

// GetSendCoins - return send coins for tax calculation
func (m Swap) GetSendCoins() Coins {
	return Coins{}
}

// GetType - Msg interface
func (m SwapSend) GetType() string {
	return "swapsend"
}

// GetSignBytes - Msg interface
func (m SwapSend) GetSignBytes() []byte {
	bz, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return sdk.MustSortJSON(bz)
}

// GetSigners - Msg interface
func (m SwapSend) GetSigners() []AccAddress {
	return []AccAddress{m.Value.FromAddress}
}

// GetSendCoins - return send coins for tax calculation
func (m SwapSend) GetSendCoins() Coins {
	return Coins{m.Value.OfferCoin}
}
