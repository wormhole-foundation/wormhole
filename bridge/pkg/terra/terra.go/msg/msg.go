package msg

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Msg - interface for wrap msg to single type
type Msg interface {
	GetType() string
	GetSignBytes() []byte
	GetSigners() []AccAddress
	GetSendCoins() Coins
}

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("terra", "terrapub")
	config.SetBech32PrefixForValidator("terravaloper", "terravaloperpub")
	config.SetBech32PrefixForValidator("terravalcons", "terravalconspub")
}
