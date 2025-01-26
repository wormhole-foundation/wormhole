package ictest

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v4/modules/apps/transfer/types"
)

// TestMalformedPayload tests the state of wormhole/osmosis chains when a GatewayIbcTokenBridge payload is malformed
// and what tokens are created
func TestMalformedPayload(t *testing.T) {
	// Base setup
	numVals := 1
	guardians := guardians.CreateValSet(t, numVals)
	chains := CreateChains(t, "v2.23.0", *guardians)
	ctx, r, eRep, _ := BuildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)
	gaia := chains[1].(*cosmos.CosmosChain)
	osmosis := chains[2].(*cosmos.CosmosChain)

	osmoToWormChannel, err := ibc.GetTransferChannel(ctx, r, eRep, osmosis.Config().ChainID, wormchain.Config().ChainID)
	require.NoError(t, err)
	wormToOsmoChannel := osmoToWormChannel.Counterparty
	gaiaToWormChannel, err := ibc.GetTransferChannel(ctx, r, eRep, gaia.Config().ChainID, wormchain.Config().ChainID)
	require.NoError(t, err)
	wormToGaiaChannel := gaiaToWormChannel.Counterparty

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", int64(10_000_000_000), wormchain, gaia, osmosis, osmosis)
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
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, vaa.ChainIDWormchain, guardians)
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
	fmt.Println("ibc_translator code id: ", ibcTranslatorCodeId)

	// Instantiate ibc translator contract
	ibcTranslatorInstantiateMsg := helpers.IbcTranslatorContractInstantiateMsg(t, tbContractAddr)
	ibcTranslatorContractAddr := helpers.InstantiateContract(t, ctx, wormchain, "faucet", ibcTranslatorCodeId, "ibc_translator", ibcTranslatorInstantiateMsg, guardians)
	fmt.Println("Ibc translator contract address: ", ibcTranslatorContractAddr)

	helpers.SetMiddlewareContract(t, ctx, wormchain, "faucet", wormchain.Config(), ibcTranslatorContractAddr, guardians)

	// Allowlist worm/osmo chain id / channel
	wormOsmoAllowlistMsg := helpers.SubmitUpdateChainToChannelMapMsg(t, OsmoChainID, wormToOsmoChannel.ChannelID, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, wormOsmoAllowlistMsg)
	require.NoError(t, err)

	// Allowlist worm/gaia chain id / channel
	wormGaiaAllowlistMsg := helpers.SubmitUpdateChainToChannelMapMsg(t, GaiaChainID, wormToGaiaChannel.ChannelID, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, wormGaiaAllowlistMsg)
	require.NoError(t, err)

	// Get Asset 1 CW20 contract address
	var tbQueryRsp helpers.TbQueryRsp
	tbQueryReq := helpers.CreateCW20Query(t, Asset1ChainID, Asset1ContractAddr)
	wormchain.QueryContract(ctx, tbContractAddr, tbQueryReq, &tbQueryRsp)
	cw20Address := tbQueryRsp.Data.Address

	// Set up cw20 req/resp
	var cw20QueryRsp helpers.Cw20WrappedQueryRsp
	cw20QueryReq := helpers.Cw20WrappedQueryMsg{TokenInfo: helpers.Cw20TokenInfo{}}

	// Get the Osmo/IBC denom of asset1
	cw20AddressBz := helpers.MustAccAddressFromBech32(cw20Address, wormchain.Config().Bech32Prefix)
	subdenom := base58.Encode(cw20AddressBz)
	asset1TokenFactoryDenom := fmt.Sprint("factory/", ibcTranslatorContractAddr, "/", subdenom)
	osmoAsset1Denom := transfertypes.GetPrefixedDenom("transfer", osmoToWormChannel.ChannelID, asset1TokenFactoryDenom)
	osmoIbcAsset1Denom := transfertypes.ParseDenomTrace(osmoAsset1Denom).IBCDenom()

	// ***************** Start of interesting test cases ***************************************************************************************
	// #                         Test                                                             |          Result                            |
	//                                                                                            | Cw20 minted | TB minted | Final location   |
	// -----------------------------------------------------------------------------------------------------------------------------------------
	// 1. GW Transfer: chain id isn't allowlisted                                                 |   X (0)        X (0)          X      (0)   |
	// 2. GW Transfer: recipient has invalid bech32 addr                                          |   + (100)      + (100)     wormchain (100) |
	// 3. GW TransferWithPayload: recipient is valid, but not a contract                          |   + (200)      + (200)     wormchain (200) |
	// 4. GW TransferWithPayload: Memo malformed: ibc hooks: invalid "wasm" root keyword          |   + (300)      + (300)     osmosis   (100) |
	// 5. GW TransferWithPayload: Memo malformed: ibc hooks: invalid recipient (bech32 invalid)   |   + (400)      + (400)     wormchain (300) |
	// 6. GW TransferWithPayload: Memo malformed: ibc hooks: invalid recipient (not a contract)   |   + (500)      + (500)     wormchain (400) |
	// 7. GW TransferWithPayload: Memo malformed: ibc hooks: msg: invalid execute method          |   + (600)      + (600)     wormchain (500) |
	// 8. GW TransferWithPayload: Memo malformed: ibc hooks: msg: invalid "forward to" recipient  |   + (700)      + (700)     wormchain (600) |
	// -----------------------------------------------------------------------------------------------------------------------------------------

	// Test 1 (GW Tranfer has 100 added to osmo chain id to make it denied / no chain id -> channel mapping)
	simplePayload := helpers.CreateGatewayIbcTokenBridgePayloadTransfer(t, OsmoChainID+100, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix), 0, 1)
	externalSender := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}
	payload3 := helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg := helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, wormchain, osmosis)
	require.NoError(t, err)

	// Check the asset 1 CW20 total supply
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	require.Equal(t, "0", cw20QueryRsp.Data.TotalSupply, "Asset 1 CW20 total supply should be 0")

	// Check ibc-translator asset 1 denom balance
	asset1DenomBalance, err := wormchain.GetBalance(ctx, ibcTranslatorContractAddr, asset1TokenFactoryDenom)
	require.NoError(t, err)
	require.Equal(t, int64(0), asset1DenomBalance, "Ibc translator asset 1 denom balance should be 0")

	// Test 2 (GW Transfer has a cosmos/gaia prefix for recipient address)
	simplePayload = helpers.CreateGatewayIbcTokenBridgePayloadTransfer(t, OsmoChainID, osmoUser1.Bech32Address(gaia.Config().Bech32Prefix), 0, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 20, wormchain, osmosis)
	require.NoError(t, err)

	// Check the asset 1 CW20 total supply
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	require.Equal(t, "100", cw20QueryRsp.Data.TotalSupply, "Asset 1 CW20 total supply should be 100")

	// Check ibc-translator asset 1 denom balance
	asset1DenomBalance, err = wormchain.GetBalance(ctx, ibcTranslatorContractAddr, asset1TokenFactoryDenom)
	require.NoError(t, err)
	require.Equal(t, int64(100), asset1DenomBalance, "Ibc translator asset 1 denom balance should be 100")

	// Test 3 (GW TransferWithPayload has osmo user1 as recipient and not a contract)
	ibcHooksPayload := helpers.CreateIbcHooksMsg(t, ibcHooksContractAddr, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload := helpers.CreateGatewayIbcTokenBridgePayloadTransferWithPayload(t, OsmoChainID, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix), ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 20, wormchain, osmosis)
	require.NoError(t, err)

	// Check the asset 1 CW20 total supply
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	require.Equal(t, "200", cw20QueryRsp.Data.TotalSupply, "Asset 1 CW20 total supply should be 200")

	// Check ibc-translator asset 1 denom balance
	asset1DenomBalance, err = wormchain.GetBalance(ctx, ibcTranslatorContractAddr, asset1TokenFactoryDenom)
	require.NoError(t, err)
	require.Equal(t, int64(200), asset1DenomBalance, "Ibc translator asset 1 denom balance should be 200")

	// Test 4 (GW TransferWithPayload - change wasm root in memo)
	ibcHooksPayload = CreateInvalidIbcHooksMsgWasm(t, ibcHooksContractAddr, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadTransferWithPayload(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 20, wormchain, osmosis)
	require.NoError(t, err)

	// Check the asset 1 CW20 total supply
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	require.Equal(t, "300", cw20QueryRsp.Data.TotalSupply, "Asset 1 CW20 total supply should be 300")

	// Check ibc-translator asset 1 denom balance
	asset1DenomBalance, err = osmosis.GetBalance(ctx, ibcHooksContractAddr, osmoIbcAsset1Denom)
	require.NoError(t, err)
	require.Equal(t, int64(100), asset1DenomBalance, "Ibchooks asset 1 denom balance should be 100")

	// Test 5 (GW TransferWithPayload's ibc hook payload has osmo user1 as recipient and not a contract)
	cosmosIbcHooksContractAddr := swapBech32Prefix(ibcHooksContractAddr, osmosis.Config().Bech32Prefix, gaia.Config().Bech32Prefix)
	ibcHooksPayload = helpers.CreateIbcHooksMsg(t, cosmosIbcHooksContractAddr, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadTransferWithPayload(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 20, wormchain, osmosis)
	require.NoError(t, err)

	// Check the asset 1 CW20 total supply
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	require.Equal(t, "400", cw20QueryRsp.Data.TotalSupply, "Asset 1 CW20 total supply should be 400")

	// Check ibc-translator asset 1 denom balance
	asset1DenomBalance, err = wormchain.GetBalance(ctx, ibcTranslatorContractAddr, asset1TokenFactoryDenom)
	require.NoError(t, err)
	require.Equal(t, int64(300), asset1DenomBalance, "Ibc translator asset 1 denom balance should be 300")

	// Test 6 (GW TransferWithPayload's ibc hook payload has osmo user1 as recipient and not a contract)
	ibcHooksPayload = helpers.CreateIbcHooksMsg(t, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix), osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadTransferWithPayload(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 20, wormchain, osmosis)
	require.NoError(t, err)

	// Check the asset 1 CW20 total supply
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	require.Equal(t, "500", cw20QueryRsp.Data.TotalSupply, "Asset 1 CW20 total supply should be 500")

	// Check ibc-translator asset 1 denom balance
	asset1DenomBalance, err = wormchain.GetBalance(ctx, ibcTranslatorContractAddr, asset1TokenFactoryDenom)
	require.NoError(t, err)
	require.Equal(t, int64(400), asset1DenomBalance, "Ibc translator asset 1 denom balance should be 400")

	// Test 7 (GW TransferWithPayload has invalid execute method for ibc hooks contract)
	ibcHooksPayload = CreateInvalidIbcHooksMsgExecute(t, ibcHooksContractAddr, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadTransferWithPayload(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 20, wormchain, osmosis)
	require.NoError(t, err)

	// Check the asset 1 CW20 total supply
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	require.Equal(t, "600", cw20QueryRsp.Data.TotalSupply, "Asset 1 CW20 total supply should be 600")

	// Check ibc-translator asset 1 denom balance
	asset1DenomBalance, err = wormchain.GetBalance(ctx, ibcTranslatorContractAddr, asset1TokenFactoryDenom)
	require.NoError(t, err)
	require.Equal(t, int64(500), asset1DenomBalance, "Ibc translator asset 1 denom balance should be 500")

	// Test 8 (GW TransferWithPayload's ibc hook payload has recipient with cosmos/gaia bech32 prefix)
	ibcHooksPayload = helpers.CreateIbcHooksMsg(t, ibcHooksContractAddr, osmoUser2.Bech32Address(gaia.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadTransferWithPayload(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 20, wormchain, osmosis)
	require.NoError(t, err)

	// Check the asset 1 CW20 total supply
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	require.Equal(t, "700", cw20QueryRsp.Data.TotalSupply, "Asset 1 CW20 total supply should be 700")

	// Check ibc-translator asset 1 denom balance
	asset1DenomBalance, err = wormchain.GetBalance(ctx, ibcTranslatorContractAddr, asset1TokenFactoryDenom)
	require.NoError(t, err)
	require.Equal(t, int64(600), asset1DenomBalance, "Ibc translator asset 1 denom balance should be 600")
}

type IbcHooksWasm struct {
	Payload helpers.IbcHooksPayload `json:"was"` // invalid keyword
}

func CreateInvalidIbcHooksMsgWasm(t *testing.T, contract string, recipient string) []byte {
	msg := IbcHooksWasm{
		Payload: helpers.IbcHooksPayload{
			Contract: contract,
			Msg: helpers.IbcHooksExecute{
				Forward: helpers.IbcHooksForward{
					Recipient: recipient,
				},
			},
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return msgBz
}

type IbcHooks struct {
	Payload IbcHooksPayload `json:"wasm"`
}

type IbcHooksPayload struct {
	Contract string          `json:"contract"`
	Msg      IbcHooksExecute `json:"msg"`
}

type IbcHooksExecute struct {
	Forward helpers.IbcHooksForward `json:"forward_tokens1"` // invalid method
}

func CreateInvalidIbcHooksMsgExecute(t *testing.T, contract string, recipient string) []byte {
	msg := IbcHooks{
		Payload: IbcHooksPayload{
			Contract: contract,
			Msg: IbcHooksExecute{
				Forward: helpers.IbcHooksForward{
					Recipient: recipient,
				},
			},
		},
	}

	msgBz, err := json.Marshal(msg)
	require.NoError(t, err)

	return msgBz
}

func swapBech32Prefix(address string, currentBech32Prefix string, newBech32Prefix string) string {
	accAddr := helpers.MustAccAddressFromBech32(address, currentBech32Prefix)
	return sdk.MustBech32ifyAddressBytes(newBech32Prefix, accAddr)
}
