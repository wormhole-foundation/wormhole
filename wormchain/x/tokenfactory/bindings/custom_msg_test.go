package bindings_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/wormhole-foundation/wormchain/app"
	bindings "github.com/wormhole-foundation/wormchain/x/tokenfactory/bindings/types"
)

func TestCreateDenomMsg(t *testing.T) {
	creator := RandomAccountAddress()
	wormchain, ctx := SetupCustomApp(t, creator)

	lucky := RandomAccountAddress()
	reflect := instantiateReflectContract(t, ctx, wormchain, lucky)
	require.NotEmpty(t, reflect)

	// Fund reflect contract with 100 base denom creation fees
	// reflectAmount := sdk.NewCoins(sdk.NewCoin(types.DefaultParams().DenomCreationFee[0].Denom, types.DefaultParams().DenomCreationFee[0].Amount.MulRaw(100)))
	// fundAccount(t, ctx, wormchain, reflect, reflectAmount)

	msg := bindings.TokenMsg{
		CreateDenom: &bindings.CreateDenom{
			Subdenom: "SUN",
		},
	}
	err := executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	// query the denom and see if it matches
	/*query := bindings.TokenFactoryQuery{
		FullDenom: &bindings.FullDenom{
			CreatorAddr: reflect.String(),
			Subdenom:    "SUN",
		},
	}
	resp := bindings.FullDenomResponse{}
	queryCustom(t, ctx, wormchain, reflect, query, &resp)

	require.Equal(t, resp.Denom, fmt.Sprintf("factory/%s/SUN", reflect.String()))*/
}

func TestMintMsg(t *testing.T) {
	creator := RandomAccountAddress()
	wormchain, ctx := SetupCustomApp(t, creator)

	lucky := RandomAccountAddress()
	reflect := instantiateReflectContract(t, ctx, wormchain, lucky)
	require.NotEmpty(t, reflect)

	// Fund reflect contract with 100 base denom creation fees
	// reflectAmount := sdk.NewCoins(sdk.NewCoin(types.DefaultParams().DenomCreationFee[0].Denom, types.DefaultParams().DenomCreationFee[0].Amount.MulRaw(100)))
	// fundAccount(t, ctx, wormchain, reflect, reflectAmount)

	// lucky was broke
	balances := wormchain.BankKeeper.GetAllBalances(ctx, lucky)
	require.Empty(t, balances)

	// Create denom for minting
	msg := bindings.TokenMsg{CreateDenom: &bindings.CreateDenom{
		Subdenom: "SUN",
	}}
	err := executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)
	sunDenom := fmt.Sprintf("factory/%s/%s", reflect.String(), msg.CreateDenom.Subdenom)

	amount, ok := sdk.NewIntFromString("808010808")
	require.True(t, ok)
	msg = bindings.TokenMsg{MintTokens: &bindings.MintTokens{
		Denom:         sunDenom,
		Amount:        amount,
		MintToAddress: lucky.String(),
	}}
	err = executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	balances = wormchain.BankKeeper.GetAllBalances(ctx, lucky)
	require.Len(t, balances, 1)
	coin := balances[0]
	require.Equal(t, amount, coin.Amount)
	require.Contains(t, coin.Denom, "factory/")

	// query the denom and see if it matches
	/*query := bindings.TokenFactoryQuery{
		FullDenom: &bindings.FullDenom{
			CreatorAddr: reflect.String(),
			Subdenom:    "SUN",
		},
	}
	resp := bindings.FullDenomResponse{}
	queryCustom(t, ctx, wormchain, reflect, query, &resp)

	require.Equal(t, resp.Denom, coin.Denom)*/

	// mint the same denom again
	err = executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	balances = wormchain.BankKeeper.GetAllBalances(ctx, lucky)
	require.Len(t, balances, 1)
	coin = balances[0]
	require.Equal(t, amount.MulRaw(2), coin.Amount)
	require.Contains(t, coin.Denom, "factory/")

	// query the denom and see if it matches
	/*query = bindings.TokenFactoryQuery{
		FullDenom: &bindings.FullDenom{
			CreatorAddr: reflect.String(),
			Subdenom:    "SUN",
		},
	}
	resp = bindings.FullDenomResponse{}
	queryCustom(t, ctx, wormchain, reflect, query, &resp)

	require.Equal(t, resp.Denom, coin.Denom)*/

	// now mint another amount / denom
	// create it first
	msg = bindings.TokenMsg{CreateDenom: &bindings.CreateDenom{
		Subdenom: "MOON",
	}}
	err = executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)
	moonDenom := fmt.Sprintf("factory/%s/%s", reflect.String(), msg.CreateDenom.Subdenom)

	amount = amount.SubRaw(1)
	msg = bindings.TokenMsg{MintTokens: &bindings.MintTokens{
		Denom:         moonDenom,
		Amount:        amount,
		MintToAddress: lucky.String(),
	}}
	err = executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	balances = wormchain.BankKeeper.GetAllBalances(ctx, lucky)
	require.Len(t, balances, 2)
	coin = balances[0]
	require.Equal(t, amount, coin.Amount)
	require.Contains(t, coin.Denom, "factory/")

	// query the denom and see if it matches
	/*query = bindings.TokenFactoryQuery{
		FullDenom: &bindings.FullDenom{
			CreatorAddr: reflect.String(),
			Subdenom:    "MOON",
		},
	}
	resp = bindings.FullDenomResponse{}
	queryCustom(t, ctx, wormchain, reflect, query, &resp)

	require.Equal(t, resp.Denom, coin.Denom)*/

	// and check the first denom is unchanged
	coin = balances[1]
	require.Equal(t, amount.AddRaw(1).MulRaw(2), coin.Amount)
	require.Contains(t, coin.Denom, "factory/")

	// query the denom and see if it matches
	/*query = bindings.TokenFactoryQuery{
		FullDenom: &bindings.FullDenom{
			CreatorAddr: reflect.String(),
			Subdenom:    "SUN",
		},
	}
	resp = bindings.FullDenomResponse{}
	queryCustom(t, ctx, wormchain, reflect, query, &resp)

	require.Equal(t, resp.Denom, coin.Denom)*/
}

// Capability disabled
/*func TestForceTransfer(t *testing.T) {
	creator := RandomAccountAddress()
	wormchain, ctx := SetupCustomApp(t, creator)

	lucky := RandomAccountAddress()
	rcpt := RandomAccountAddress()
	reflect := instantiateReflectContract(t, ctx, wormchain, lucky)
	require.NotEmpty(t, reflect)

	// Fund reflect contract with 100 base denom creation fees
	reflectAmount := sdk.NewCoins(sdk.NewCoin(types.DefaultParams().DenomCreationFee[0].Denom, types.DefaultParams().DenomCreationFee[0].Amount.MulRaw(100)))
	fundAccount(t, ctx, wormchain, reflect, reflectAmount)

	// lucky was broke
	balances := wormchain.BankKeeper.GetAllBalances(ctx, lucky)
	require.Empty(t, balances)

	// Create denom for minting
	msg := bindings.TokenFactoryMsg{CreateDenom: &bindings.CreateDenom{
		Subdenom: "SUN",
	}}
	err := executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)
	sunDenom := fmt.Sprintf("factory/%s/%s", reflect.String(), msg.CreateDenom.Subdenom)

	amount, ok := sdk.NewIntFromString("808010808")
	require.True(t, ok)

	// Mint new tokens to lucky
	msg = bindings.TokenFactoryMsg{MintTokens: &bindings.MintTokens{
		Denom:         sunDenom,
		Amount:        amount,
		MintToAddress: lucky.String(),
	}}
	err = executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	// Force move 100 tokens from lucky to rcpt
	msg = bindings.TokenFactoryMsg{ForceTransfer: &bindings.ForceTransfer{
		Denom:       sunDenom,
		Amount:      sdk.NewInt(100),
		FromAddress: lucky.String(),
		ToAddress:   rcpt.String(),
	}}
	err = executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	// check the balance of rcpt
	balances = wormchain.BankKeeper.GetAllBalances(ctx, rcpt)
	require.Len(t, balances, 1)
	coin := balances[0]
	require.Equal(t, sdk.NewInt(100), coin.Amount)
}*/

func TestBurnMsg(t *testing.T) {
	creator := RandomAccountAddress()
	wormchain, ctx := SetupCustomApp(t, creator)

	lucky := RandomAccountAddress()
	reflect := instantiateReflectContract(t, ctx, wormchain, lucky)
	require.NotEmpty(t, reflect)

	// Fund reflect contract with 100 base denom creation fees
	// reflectAmount := sdk.NewCoins(sdk.NewCoin(types.DefaultParams().DenomCreationFee[0].Denom, types.DefaultParams().DenomCreationFee[0].Amount.MulRaw(100)))
	// fundAccount(t, ctx, wormchain, reflect, reflectAmount)

	// lucky was broke
	balances := wormchain.BankKeeper.GetAllBalances(ctx, lucky)
	require.Empty(t, balances)

	// Create denom for minting
	msg := bindings.TokenMsg{CreateDenom: &bindings.CreateDenom{
		Subdenom: "SUN",
	}}
	err := executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)
	sunDenom := fmt.Sprintf("factory/%s/%s", reflect.String(), msg.CreateDenom.Subdenom)

	amount, ok := sdk.NewIntFromString("808010809")
	require.True(t, ok)

	msg = bindings.TokenMsg{MintTokens: &bindings.MintTokens{
		Denom:         sunDenom,
		Amount:        amount,
		MintToAddress: lucky.String(),
	}}
	err = executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	// Capability disabled
	// can burn from different address with burnFrom
	/*amt, ok := sdk.NewIntFromString("1")
	require.True(t, ok)
	msg = bindings.TokenFactoryMsg{BurnTokens: &bindings.BurnTokens{
		Denom:           sunDenom,
		Amount:          amt,
		BurnFromAddress: lucky.String(),
	}}
	err = executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)

	// lucky needs to send balance to reflect contract to burn it
	luckyBalance := wormchain.BankKeeper.GetAllBalances(ctx, lucky)
	err = wormchain.BankKeeper.SendCoins(ctx, lucky, reflect, luckyBalance)
	require.NoError(t, err)

	msg = bindings.TokenFactoryMsg{BurnTokens: &bindings.BurnTokens{
		Denom:           sunDenom,
		Amount:          amount.Abs().Sub(sdk.NewInt(1)),
		BurnFromAddress: reflect.String(),
	}}
	err = executeCustom(t, ctx, wormchain, reflect, lucky, msg, sdk.Coin{})
	require.NoError(t, err)*/
}

type ReflectExec struct {
	ReflectMsg    *ReflectMsgs    `json:"reflect_msg,omitempty"`
	ReflectSubMsg *ReflectSubMsgs `json:"reflect_sub_msg,omitempty"`
}

type ReflectMsgs struct {
	Msgs []wasmvmtypes.CosmosMsg `json:"msgs"`
}

type ReflectSubMsgs struct {
	Msgs []wasmvmtypes.SubMsg `json:"msgs"`
}

func executeCustom(t *testing.T, ctx sdk.Context, wormchain *app.App, contract sdk.AccAddress, sender sdk.AccAddress, msg bindings.TokenMsg, funds sdk.Coin) error { //nolint:unparam // funds is always nil but could change in the future.
	wrapped := bindings.TokenFactoryMsg{
		Token: &msg,
	}
	customBz, err := json.Marshal(wrapped)
	require.NoError(t, err)

	reflectMsg := ReflectExec{
		ReflectMsg: &ReflectMsgs{
			Msgs: []wasmvmtypes.CosmosMsg{{
				Custom: customBz,
			}},
		},
	}
	reflectBz, err := json.Marshal(reflectMsg)
	require.NoError(t, err)

	// no funds sent if amount is 0
	var coins sdk.Coins
	if !funds.Amount.IsNil() {
		coins = sdk.Coins{funds}
	}

	contractKeeper := keeper.NewDefaultPermissionKeeper(wormchain.GetWasmKeeper())
	_, err = contractKeeper.Execute(ctx, contract, sender, reflectBz, coins)
	return err
}
