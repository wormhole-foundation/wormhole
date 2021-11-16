package types

import (
	"github.com/certusone/wormhole-chain/x/wormhole/types"
	"github.com/certusone/wormhole/node/pkg/vaa"
	sdk "github.com/cosmos/cosmos-sdk/types"
	btypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

type AccountKeeper interface {
	// Methods imported from account should be defined here
	GetModuleAddress(moduleName string) sdk.AccAddress
}

type BankKeeper interface {
	// Methods imported from bank should be defined here
	MintCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error
	SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SetDenomMetaData(ctx sdk.Context, denomMetaData btypes.Metadata)
	GetDenomMetaData(ctx sdk.Context, denom string) (denomMetaData btypes.Metadata, found bool)
	GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

type WormholeKeeper interface {
	// Methods imported from wormhole should be defined here
	VerifyVAA(ctx sdk.Context, vaa *vaa.VAA) error
	GetConfig(ctx sdk.Context) (val types.Config, found bool)
	PostMessage(ctx sdk.Context, emitter types.EmitterAddress, nonce uint32, data []byte) error
}
