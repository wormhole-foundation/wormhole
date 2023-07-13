package bindings

import (
	"github.com/CosmWasm/wasmd/x/wasm"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	tokenfactorykeeper "github.com/wormhole-foundation/wormchain/x/tokenfactory/keeper"
)

func RegisterCustomPlugins(
	bank *bankkeeper.BaseKeeper,
	tokenFactory *tokenfactorykeeper.Keeper,
) []wasmkeeper.Option {
	// Disabling tokenfactory custom querier because wormchain custom querier exists
	//wasmQueryPlugin := NewQueryPlugin(bank, tokenFactory)

	//queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
	//	Custom: CustomQuerier(wasmQueryPlugin),
	//})
	messengerDecoratorOpt := wasmkeeper.WithMessageHandlerDecorator(
		CustomMessageDecorator(bank, tokenFactory),
	)

	return []wasm.Option{
		//	queryPluginOpt,
		messengerDecoratorOpt,
	}
}
