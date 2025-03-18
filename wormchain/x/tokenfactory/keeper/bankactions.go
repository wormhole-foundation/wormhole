package keeper

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
	denoms "github.com/wormhole-foundation/wormchain/x/tokenfactory/types"
)

var MainnetUseConditionalHeight = int64(12275950)

func (k Keeper) mintTo(ctx sdk.Context, amount sdk.Coin, mintTo string) error {
	// verify that denom is an x/tokenfactory denom
	_, _, err := types.DeconstructDenom(amount.Denom)
	if err != nil {
		return err
	}

	// We enable the new conditional approximately two weeks after block 12,066,314 for mainnet and 13,361,706 on testnet, which is
	// calculated by dividing the number of seconds in a week by the average block time (~6s).
	// On testnet, the block height is different (and so is the block time) with a block time of ~6s.
	// On mainnet, the average block time is 5.77 seconds according to the PR https://github.com/wormhole-foundation/wormhole/pull/3946/files.
	// At 5.77 seconds/block, this is ~209,636 blocks for mainnet. On testnet at 6 seconds/block, this is ~201,600 blocks for testnet.
	// Therefore, mainnet cutover height is 12,066,314 + 209,636 = 12,275,950 and testnet cutover height is 13,361,706 + 201,600 = 13,563,306.
	// The target is about ~7:30pm UTC January 28th, 2025.
	isMainnet := ctx.ChainID() == "wormchain"
	isTestnet := ctx.ChainID() == "wormchain-testnet-0"

	if (isMainnet && ctx.BlockHeight() >= MainnetUseConditionalHeight) || (isTestnet && ctx.BlockHeight() >= 13563306) {
		// Cutover is required because the call to GetSupply() will use more gas, which would result in a consensus failure.
		totalSupplyCurrent := k.bankKeeper.GetSupply(ctx, amount.Denom)
		TotalSupplyAfter := totalSupplyCurrent.Add(amount) // Can't integer overflow because of a ValidateBasic() check on this amount
		if TotalSupplyAfter.Amount.GTE(denoms.MintAmountLimit) {
			return fmt.Errorf("failed to mint - surpassed maximum mint amount")
		}
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
