package ictest

import (
	"encoding/json"
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// TestMalformedPayload tests the state of wormhole/osmosis chains when a GatewayIbcTokenBridge payload is malformed
// and what tokens are created
func TestMalformedPayload(t *testing.T) {
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
	wrappedAssetCodeId := helpers.StoreContract(t, ctx, wormchain,"faucet", "./contracts/cw20_wrapped_2.wasm", guardians)
	fmt.Println("CW20 wrapped_2 code id: ", wrappedAssetCodeId)

	// Store token bridge contract
	tbContractCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/token_bridge.wasm", guardians)
	fmt.Println("Token bridge contract code id: ", tbContractCodeId)

	// Instantiate token bridge contract
	tbInstantiateMsg:= helpers.TbContractInstantiateMsg(t, wormchainConfig, coreContractAddr, wrappedAssetCodeId)
	tbContractAddr := helpers.InstantiateContract(t, ctx, wormchain, "faucet", tbContractCodeId, "token_bridge", tbInstantiateMsg, guardians)
	fmt.Println("Token bridge contract address: ", tbContractAddr)

	// Register a new external chain
	tbRegisterChainMsg := helpers.TbRegisterChainMsg(t, ExternalChainId, ExternalChainEmitterAddr, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", tbContractAddr, string(tbRegisterChainMsg))
	require.NoError(t, err)

	// Register a new foreign asset (Asset1) originating on externalChain
	tbRegisterForeignAssetMsg := helpers.TbRegisterForeignAsset(t, Asset1ContractAddr, Asset1ChainID, ExternalChainEmitterAddr, Asset1Decimals, Asset1Symbol, Asset1Name, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", tbContractAddr, string(tbRegisterForeignAssetMsg))
	require.NoError(t, err)
	
	// Store ibc translator contract
	ibcTranslatorCodeId := helpers.StoreContract(t, ctx, wormchain,"faucet", "./contracts/ibc_translator.wasm", guardians)
	fmt.Println("ibc_translator code id: ", ibcTranslatorCodeId)

	// Instantiate ibc translator contract
	ibcTranslatorInstantiateMsg := helpers.IbcTranslatorContractInstantiateMsg(t, tbContractAddr, coreContractAddr)
	ibcTranslatorContractAddr := helpers.InstantiateContract(t, ctx, wormchain, "faucet", ibcTranslatorCodeId, "ibc_translator", ibcTranslatorInstantiateMsg, guardians)
	fmt.Println("Ibc translator contract address: ", ibcTranslatorContractAddr)

	// Allowlist worm/osmo chain id / channel
	wormOsmoAllowlistMsg := helpers.SubmitUpdateChainToChannelMapMsg(t, OsmoChainID, wormToOsmoChannel.ChannelID, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, wormOsmoAllowlistMsg)

	// Allowlist worm/gaia chain id / channel
	wormGaiaAllowlistMsg := helpers.SubmitUpdateChainToChannelMapMsg(t, GaiaChainID, wormToGaiaChannel.ChannelID, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, wormGaiaAllowlistMsg)

	var tbQueryRsp helpers.TbQueryRsp
	tbQueryReq := helpers.CreateCW20Query(t, Asset1ChainID, Asset1ContractAddr)
	wormchain.QueryContract(ctx, tbContractAddr, tbQueryReq, &tbQueryRsp)
	cw20Address := tbQueryRsp.Data.Address
	fmt.Println("Asset1 cw20 addr: ", cw20Address)

	var cw20QueryRsp helpers.Cw20WrappedQueryRsp
	cw20QueryReq := helpers.Cw20WrappedQueryMsg{TokenInfo: helpers.Cw20TokenInfo{}}
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply: ", cw20QueryRsp.Data.TotalSupply)

	// ***************** Start of interesting test cases *************************************************************************
	// #                         Test                                                 |          Result                          |
	//                                                                                | Cw20 minted | TB minted | Final location |
	// ---------------------------------------------------------------------------------------------------------------------------
	// 1. Simple payload: chain id isn't allowlisted                                  |     X            X           X           |
	// 2. Simple payload: recipient has invalid bech32 addr                           |     +            +         wormchain     |
	// 3. CC payload: recipient is valid, but not a contract                          |     +            +         wormchain     |
	// 4. CC payload: Memo malformed: ibc hooks: invalid "wasm" root keyword          |     +            +         osmosis       |
	// 5. CC payload: Memo malformed: ibc hooks: invalid recipient (bech32 invalid)   |     +            +         wormchain     |
	// 6. CC payload: Memo malformed: ibc hooks: invalid recipient (not a contract)   |     +            +         wormchain     |
	// 7. CC payload: Memo malformed: ibc hooks: msg: invalid execute method          |     +            +         wormchain     |
	// 8. CC payload: Memo malformed: ibc hooks: msg: invalid "forward to" recipient  |     +            +         wormchain     |
	// ---------------------------------------------------------------------------------------------------------------------------

	// Test 1 (Simple payload has 100 added to osmo chain id)
	simplePayload := helpers.CreateGatewayIbcTokenBridgePayloadSimple(t, OsmoChainID+100, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix), 0, 1)
	externalSender := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8 ,1, 2, 3, 4, 5, 6, 7, 8}
	payload3 := helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg := helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	err = testutil.WaitForBlocks(ctx, 2, wormchain)

	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply after test1: ", cw20QueryRsp.Data.TotalSupply)

	coins, err := wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)
	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	// Test 2 (Simple payload has a cosmos/gaia prefix for recipient address)
	simplePayload = helpers.CreateGatewayIbcTokenBridgePayloadSimple(t, OsmoChainID, osmoUser1.Bech32Address(gaia.Config().Bech32Prefix), 0, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	err = testutil.WaitForBlocks(ctx, 2, wormchain)

	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply after test2: ", cw20QueryRsp.Data.TotalSupply)

	coins, err = wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)
	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	// Test 3 (CC payload has osmo user1 as recipient and not a contract)
	ibcHooksPayload := helpers.CreateIbcHooksMsg(t, ibcHooksContractAddr, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload := helpers.CreateGatewayIbcTokenBridgePayloadContract(t, OsmoChainID, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix), ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	err = testutil.WaitForBlocks(ctx, 2, wormchain)

	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply after test3: ", cw20QueryRsp.Data.TotalSupply)

	coins, err = wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)
	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	// Test 4 (change wasm)
	ibcHooksPayload = CreateInvalidIbcHooksMsgWasm(t, ibcHooksContractAddr, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadContract(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	err = testutil.WaitForBlocks(ctx, 2, wormchain)

	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply after test4: ", cw20QueryRsp.Data.TotalSupply)

	coins, err = wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)
	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	// Test 5 (CC payload's ibc hook payload has osmo user1 as recipient and not a contract)
	cosmosIbcHooksContractAddr := swapBech32Prefix(ibcHooksContractAddr, osmosis.Config().Bech32Prefix,gaia.Config().Bech32Prefix)
	ibcHooksPayload = helpers.CreateIbcHooksMsg(t, cosmosIbcHooksContractAddr, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadContract(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	err = testutil.WaitForBlocks(ctx, 2, wormchain)

	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply after test5: ", cw20QueryRsp.Data.TotalSupply)

	coins, err = wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)
	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	// Test 6 (CC payload's ibc hook payload has osmo user1 as recipient and not a contract)
	ibcHooksPayload = helpers.CreateIbcHooksMsg(t, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix), osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadContract(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	err = testutil.WaitForBlocks(ctx, 2, wormchain)

	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply after test6: ", cw20QueryRsp.Data.TotalSupply)

	coins, err = wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)
	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	// Test 7
	ibcHooksPayload = CreateInvalidIbcHooksMsgExecute(t, ibcHooksContractAddr, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadContract(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	err = testutil.WaitForBlocks(ctx, 2, wormchain)

	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply after test7: ", cw20QueryRsp.Data.TotalSupply)

	coins, err = wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)
	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	// Test 8 (CC payload's ibc hook payload has recipient with cosmos/gaia bech32 prefix)
	ibcHooksPayload = helpers.CreateIbcHooksMsg(t, ibcHooksContractAddr, osmoUser2.Bech32Address(gaia.Config().Bech32Prefix))
	contractControlledPayload = helpers.CreateGatewayIbcTokenBridgePayloadContract(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), 100, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	err = testutil.WaitForBlocks(ctx, 2, wormchain)

	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply after test8: ", cw20QueryRsp.Data.TotalSupply)

	coins, err = wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)
	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	// wait for transfer
	err = testutil.WaitForBlocks(ctx, 10, wormchain)
	require.NoError(t, err)
	
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply: ", cw20QueryRsp.Data.TotalSupply)

	coins, err = wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)

	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	// wait for transfer
	err = testutil.WaitForBlocks(ctx, 60, wormchain)
	require.NoError(t, err)
	fmt.Println("*************** after 2 min ***********************")
	
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply: ", cw20QueryRsp.Data.TotalSupply)

	coins, err = wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)

	coins, err = osmosis.AllBalances(ctx, ibcHooksContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc hooks contract coins: ", coins)

	coins, err = gaia.AllBalances(ctx, gaiaUser.Bech32Address(gaia.Config().Bech32Prefix))
	require.NoError(t, err)
	fmt.Println("Gaia user coins: ", coins)
	
	coins, err = osmosis.AllBalances(ctx, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix))
	require.NoError(t, err)
	fmt.Println("Osmo user1 coins: ", coins)

	coins, err = osmosis.AllBalances(ctx, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	require.NoError(t, err)
	fmt.Println("Osmo user2 coins: ", coins)

}

type IbcHooksWasm struct {
	Payload helpers.IbcHooksPayload `json:"was"` // invalid keyword
}

func CreateInvalidIbcHooksMsgWasm(t *testing.T, contract string, recipient string) []byte {
	msg := IbcHooksWasm {
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
	Contract string `json:"contract"`
	Msg IbcHooksExecute `json:"msg"`
}

type IbcHooksExecute struct {
	Forward helpers.IbcHooksForward `json:"forward_tokens1"` // invalid method
}
func CreateInvalidIbcHooksMsgExecute(t *testing.T, contract string, recipient string) []byte {
	msg := IbcHooks {
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