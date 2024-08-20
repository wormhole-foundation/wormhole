package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
)

func (k Keeper) mintTo(ctx sdk.Context, amount sdk.Coin, mintTo string) error {
	// verify that denom is an x/tokenfactory denom
	_, _, err := types.DeconstructDenom(amount.Denom)
	if err != nil {
		return err
	}

	err = k.bankKeeper.MintCoins(ctx, types.ModuleName, sdk.NewCoins(amount))
	if err != nil {
		return err
	}

	addr, err := sdk.AccAddressFromBech32(mintTo)
	if err != nil {
		return err
	}

	if k.IsModuleAcc(ctx, addr) {
		return types.ErrModuleAccount
	}

	return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName,
		addr,
		sdk.NewCoins(amount))
}

func (k Keeper) burnFrom(ctx sdk.Context, amount sdk.Coin, burnFrom string) error {
	// verify that denom is an x/tokenfactory denom
	_, _, err := types.DeconstructDenom(amount.Denom)
	if err != nil {
		return err
	}

	addr, err := sdk.AccAddressFromBech32(burnFrom)
	if err != nil {
		return err
	}

	if k.IsModuleAcc(ctx, addr) {
		return types.ErrModuleAccount
	}

	err = k.bankKeeper.SendCoinsFromAccountToModule(ctx,
		addr,
		types.ModuleName,
		sdk.NewCoins(amount))
	if err != nil {
		return err
	}

	return k.bankKeeper.BurnCoins(ctx, types.ModuleName, sdk.NewCoins(amount))
}

func (k Keeper) forceTransfer(ctx sdk.Context, amount sdk.Coin, fromAddr string, toAddr string) error {
	// verify that denom is an x/tokenfactory denom
	_, _, err := types.DeconstructDenom(amount.Denom)
	if err != nil {
		return err
	}

	fromAcc, err := sdk.AccAddressFromBech32(fromAddr)
	if err != nil {
		return err
	}

	if k.IsModuleAcc(ctx, fromAcc) {
		return types.ErrModuleAccount
	}

	toAcc, err := sdk.AccAddressFromBech32(toAddr)
	if err != nil {
		return err
	}

	if k.IsModuleAcc(ctx, toAcc) {
		return types.ErrModuleAccount
	}

	return k.bankKeeper.SendCoins(ctx, fromAcc, toAcc, sdk.NewCoins(amount))
}

// IsModuleAcc checks if a given address is restricted
func (k Keeper) IsModuleAcc(_ sdk.Context, addr sdk.AccAddress) bool {
	return k.permAddrMap[addr.String()]
}
