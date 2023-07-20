package ictest

import (
	"fmt"
	"testing"

	"github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// *******  Setup  *******
// Clone: github.com/strangelove-ventures/heighliner
// Checkout: 44cba21d0e1cfd046d33916671b4f3ffb78d12ee
// Run: "go install"
// Go to wormhole repo with tokenfactory and middlewares
// From wormhole root, run: "heighliner build -c wormchain -g local --local"
// From this directory, run: "go test -v -timeout 10m -run ^TestExternalToCosmos$ github.com/wormhole-foundation/wormchain/interchaintest -count=1"

// Note: once wormchain is added to heighliner, setup will not be required. Will just need to run the test case / last step.

// TestExternalToCosmos runs through simple test cases for external to cosmos transfers
func TestExternalToCosmos(t *testing.T) {
	t.Parallel()

	// Base setup
	guardians := guardians.CreateValSet(t, numVals)
	chains := CreateChains(t, *guardians)
	ctx, r, eRep := BuildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)
	gaia := chains[1].(*cosmos.CosmosChain)
	osmosis := chains[2].(*cosmos.CosmosChain)

	osmoToWormChannel, err := ibc.GetTransferChannel(ctx, r, eRep, osmosis.Config().ChainID, wormchain.Config().ChainID)
	wormToOsmoChannel := osmoToWormChannel.Counterparty
	gaiaToWormChannel, err := ibc.GetTransferChannel(ctx, r, eRep, gaia.Config().ChainID, wormchain.Config().ChainID)
	wormToGaiaChannel := gaiaToWormChannel.Counterparty

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", int64(10_000_000_000), wormchain, gaia, osmosis, osmosis)
	_ = users[0] // Wormchain user
	gaiaUser := users[1]
	osmoUser1 := users[2]
	osmoUser2 := users[3]

	ibcHooksCodeId, err := osmosis.StoreContract(ctx, osmoUser1.KeyName, "./contracts/ibc_hooks.wasm")
	require.NoError(t, err)
	fmt.Println("IBC hooks code id: ", ibcHooksCodeId)

	ibcHooksContractAddr, err := osmosis.InstantiateContract(ctx, osmoUser1.KeyName, ibcHooksCodeId, "{}", true)
	require.NoError(t, err)
	fmt.Println("IBC hooks contract addr: ", ibcHooksContractAddr)

	// Store wormhole core contract
	coreContractCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/wormhole_core.wasm", guardians)
	fmt.Println("Core contract code id: ", coreContractCodeId)

	// Instantiate wormhole core contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	coreContractAddr := helpers.InstantiateContract(t, ctx, wormchain, "faucet", coreContractCodeId, "wormhole_core", coreInstantiateMsg, guardians)
	fmt.Println("Core contract address: ", coreContractAddr)

	// Store cw20_wrapped_2 contract
	wrappedAssetCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/cw20_wrapped_2.wasm", guardians)
	fmt.Println("CW20 wrapped_2 code id: ", wrappedAssetCodeId)

	// Store token bridge contract
	tbContractCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/token_bridge.wasm", guardians)
	fmt.Println("Token bridge contract code id: ", tbContractCodeId)

	// Instantiate token bridge contract
	tbInstantiateMsg := helpers.TbContractInstantiateMsg(t, wormchainConfig, coreContractAddr, wrappedAssetCodeId)
	tbContractAddr := helpers.InstantiateContract(t, ctx, wormchain, "faucet", tbContractCodeId, "token_bridge", tbInstantiateMsg, guardians)
	fmt.Println("Token bridge contract address: ", tbContractAddr)

	helpers.SubmitAllowlistInstantiateContract(t, ctx, wormchain, "faucet", wormchain.Config(), tbContractAddr, wrappedAssetCodeId, guardians)

	// Register a new external chain
	tbRegisterChainMsg := helpers.TbRegisterChainMsg(t, ExternalChainId, ExternalChainEmitterAddr, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", tbContractAddr, string(tbRegisterChainMsg))
	require.NoError(t, err)

	// Register a new foreign asset (Asset1) originating on externalChain
	tbRegisterForeignAssetMsg := helpers.TbRegisterForeignAsset(t, Asset1ContractAddr, Asset1ChainID, ExternalChainEmitterAddr, Asset1Decimals, Asset1Symbol, Asset1Name, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", tbContractAddr, string(tbRegisterForeignAssetMsg))
	require.NoError(t, err)

	// Store ibc translator contract
	ibcTranslatorCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/ibc_translator.wasm", guardians)
	fmt.Println("Ibc translator code id: ", ibcTranslatorCodeId)

	// Instantiate ibc translator contract
	ibcTranslatorInstantiateMsg := helpers.IbcTranslatorContractInstantiateMsg(t, tbContractAddr)
	ibcTranslatorContractAddr := helpers.InstantiateContract(t, ctx, wormchain, "faucet", ibcTranslatorCodeId, "ibc_translator", ibcTranslatorInstantiateMsg, guardians)
	fmt.Println("Ibc translator contract address: ", ibcTranslatorContractAddr)

	helpers.SetMiddlewareContract(t, ctx, wormchain, "faucet", wormchain.Config(), ibcTranslatorContractAddr, guardians)

	// Allowlist worm/osmo chain id / channel
	wormOsmoAllowlistMsg := helpers.SubmitUpdateChainToChannelMapMsg(t, OsmoChainID, wormToOsmoChannel.ChannelID, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, wormOsmoAllowlistMsg)

	// Allowlist worm/gaia chain id / channel
	wormGaiaAllowlistMsg := helpers.SubmitUpdateChainToChannelMapMsg(t, GaiaChainID, wormToGaiaChannel.ChannelID, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, wormGaiaAllowlistMsg)

	// Create and process a simple ibc payload3: Transfers 1.231245 of asset1 from external chain through wormchain to gaia user
	simplePayload := helpers.CreateGatewayIbcTokenBridgePayloadSimple(t, GaiaChainID, gaiaUser.Bech32Address(gaia.Config().Bech32Prefix), 0, 1)
	externalSender := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}
	payload3 := helpers.CreatePayload3(wormchain.Config(), 1231245, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg := helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)

	// Create and process a simple ibc payload3: Transfers 1.987654 of asset1 from external chain through wormchain to osmo user1
	simplePayload = helpers.CreateGatewayIbcTokenBridgePayloadSimple(t, OsmoChainID, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix), 0, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 1987654, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)

	// Create and process a contract controlled ibc payload3
	// Transfers 1.456789 of asset1 from external chain through wormchain to ibc hooks contract addr
	// IBC hooks is used to route the contract controlled payload to a test contract which forwards tokens to osmo user2
	ibcHooksPayload := helpers.CreateIbcHooksMsg(t, ibcHooksContractAddr, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload := helpers.CreateGatewayIbcTokenBridgePayloadContract(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 1456789, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)

	// wait for transfer
	err = testutil.WaitForBlocks(ctx, 3, wormchain)
	require.NoError(t, err)

	coins, err := wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)

	coins, err = gaia.AllBalances(ctx, gaiaUser.Bech32Address(gaia.Config().Bech32Prefix))
	require.NoError(t, err)
	fmt.Println("Gaia user coins: ", coins)

	coins, err = osmosis.AllBalances(ctx, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix))
	require.NoError(t, err)
	fmt.Println("Osmo user1 coins: ", coins)

	coins, err = osmosis.AllBalances(ctx, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	require.NoError(t, err)
	fmt.Println("Osmo user2 coins: ", coins)

	err = testutil.WaitForBlocks(ctx, 2, wormchain)
	require.NoError(t, err)
}
