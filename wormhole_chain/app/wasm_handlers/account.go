package wasm_handlers

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
)

type AccountKeeperHandler struct {
	AccountKeeper authkeeper.AccountKeeper
}

var _ wasmtypes.AccountKeeper = &AccountKeeperHandler{}

func (b *AccountKeeperHandler) NewAccountWithAddress(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI {
	// New accounts are needed for new contracts
	return b.AccountKeeper.NewAccountWithAddress(ctx, addr)
}

// Retrieve an account from the store.
func (b *AccountKeeperHandler) GetAccount(ctx sdk.Context, addr sdk.AccAddress) authtypes.AccountI {
	return b.AccountKeeper.GetAccount(ctx, addr)
}

// Set an account in the store.
func (b *AccountKeeperHandler) SetAccount(ctx sdk.Context, acc authtypes.AccountI) {
	// New accounts are needed for new contracts
	b.AccountKeeper.SetAccount(ctx, acc)
}
