package wasm_handlers

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
)

type BankViewKeeperHandler struct {
	Keeper bankkeeper.Keeper
}
type BurnerHandler struct {
	Keeper bankkeeper.Keeper
}

type BankKeeperHandler struct {
	BankViewKeeperHandler
	BurnerHandler
}

var _ wasmtypes.BankViewKeeper = &BankViewKeeperHandler{}
var _ wasmtypes.Burner = &BurnerHandler{}
var _ wasmtypes.BankKeeper = &BankKeeperHandler{}

func (b *BankViewKeeperHandler) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	return b.Keeper.GetAllBalances(ctx, addr)
}
func (b *BankViewKeeperHandler) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	return b.Keeper.GetBalance(ctx, addr, denom)
}
func (b *BankViewKeeperHandler) GetSupply(ctx sdk.Context, denom string) sdk.Coin {
	return b.Keeper.GetSupply(ctx, denom)
}

func (b *BurnerHandler) BurnCoins(ctx sdk.Context, moduleName string, amt sdk.Coins) error {
	return b.Keeper.BurnCoins(ctx, moduleName, amt)
}

func (b *BurnerHandler) SendCoinsFromAccountToModule(ctx sdk.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return b.Keeper.SendCoinsFromAccountToModule(ctx, senderAddr, recipientModule, amt)
}

func (b *BankKeeperHandler) IsSendEnabledCoins(ctx sdk.Context, coins ...sdk.Coin) error {
	return b.BankViewKeeperHandler.Keeper.IsSendEnabledCoins(ctx, coins...)
}
func (b *BankKeeperHandler) BlockedAddr(addr sdk.AccAddress) bool {
	return b.BankViewKeeperHandler.Keeper.BlockedAddr(addr)
}
func (b *BankKeeperHandler) SendCoins(ctx sdk.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	return b.BankViewKeeperHandler.Keeper.SendCoins(ctx, fromAddr, toAddr, amt)
}
