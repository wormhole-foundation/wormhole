package ictest

import (
	//"encoding/hex"
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

// TestWormchain runs through a simple test case for each deliverable
func TestCosmosToExternal(t *testing.T) {
	t.Parallel()

	// Base setup
	guardians := guardians.CreateValSet(t, numVals)
	chains := CreateChains(t, *guardians)
	ctx, r, eRep := BuildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)
	gaia := chains[1].(*cosmos.CosmosChain)
	osmosis := chains[2].(*cosmos.CosmosChain)

	wormchainFaucetAddrBz, err := wormchain.GetAddress(ctx, "faucet")
	require.NoError(t, err)
	wormchainFaucetAddr := sdk.MustBech32ifyAddressBytes(wormchain.Config().Bech32Prefix, wormchainFaucetAddrBz)
	fmt.Println("Wormchain faucet addr: ", wormchainFaucetAddr)

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

	// This query was added in a newer version of ibc-translator, so a migration should have taken place for this to succeed
	var queryChannelRsp helpers.IbcTranslatorQueryRspMsg
	queryChannelMsg := helpers.IbcTranslatorQueryMsg{IbcChannel: helpers.QueryIbcChannel{ChainID: OsmoChainID}}
	wormchain.QueryContract(ctx, ibcTranslatorContractAddr, queryChannelMsg, &queryChannelRsp)
	fmt.Println("Osmo channel: ", queryChannelRsp.Data.Channel)

	queryChannelMsg = helpers.IbcTranslatorQueryMsg{IbcChannel: helpers.QueryIbcChannel{ChainID: GaiaChainID}}
	wormchain.QueryContract(ctx, ibcTranslatorContractAddr, queryChannelMsg, &queryChannelRsp)
	fmt.Println("Gaia channel: ", queryChannelRsp.Data.Channel)
	
	// Create and process a simple ibc payload3: Transfers 1.231245 of asset1 from external chain through wormchain to gaia user
	// Add relayer fee
	simplePayload := helpers.CreateGatewayIbcTokenBridgePayloadSimple(t, GaiaChainID, gaiaUser.Bech32Address(gaia.Config().Bech32Prefix), 0, 1)
	externalSender := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8 ,1, 2, 3, 4, 5, 6, 7, 8}
	payload3 := helpers.CreatePayload3(wormchain.Config(), 1231245, Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg := helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)

	var tbQueryRsp helpers.TbQueryRsp
	tbQueryReq := helpers.CreateCW20Query(t, Asset1ChainID, Asset1ContractAddr)
	wormchain.QueryContract(ctx, tbContractAddr, tbQueryReq, &tbQueryRsp)
	cw20Address := tbQueryRsp.Data.Address
	fmt.Println("Asset1 cw20 addr: ", cw20Address)

	// Transfer some asset1 tokens back to faucet
	cw20AddressBz := helpers.MustAccAddressFromBech32(cw20Address, wormchain.Config().Bech32Prefix)
	subdenom := base58.Encode(cw20AddressBz)
	tokenFactoryDenom := fmt.Sprint("factory/", ibcTranslatorContractAddr, "/", subdenom)
	gaiaAsset1Denom := transfertypes.GetPrefixedDenom("transfer", gaiaToWormChannel.ChannelID, tokenFactoryDenom)
	gaiaIbcAsset1Denom := transfertypes.ParseDenomTrace(gaiaAsset1Denom).IBCDenom()
	
	
	// ************* Ibc hooks + Simple payload ****************
	// Call SimpleConvertAndTransfer and ContractControlledConvertAndTransfer
	// Using "externalSender" as recipient because it is 32 bytes
	simpleIbcHooksMsg := helpers.CreateIbcTranslatorIbcHooksSimpleMsg(t, ibcTranslatorContractAddr, Asset1ChainID, string(externalSender), 0, 1)
	transfer := ibc.WalletAmount{
		Address: ibcTranslatorContractAddr,
		Denom: gaiaIbcAsset1Denom,
		Amount: 10001,
	}
	gaiaHeight, err := gaia.Height(ctx)
	require.NoError(t, err)
	gaiaIbcTx, err := gaia.SendIBCTransfer(ctx, gaiaToWormChannel.ChannelID, gaiaUser.KeyName, transfer, ibc.TransferOptions{Memo: simpleIbcHooksMsg})
	require.NoError(t, err)
	
	// wait for transfer to ack
	_, err = testutil.PollForAck(ctx, gaia, gaiaHeight, gaiaHeight+30, gaiaIbcTx.Packet)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, wormchain, gaia)
	require.NoError(t, err)

	// ********* Ibc hooks + Contract controlled payload **********
	ccIbcHooksMsg := helpers.CreateIbcTranslatorIbcHooksContractControlledMsg(t, ibcTranslatorContractAddr, Asset1ChainID, string(externalSender), []byte("ExternalContractPayload"), 1)
	transfer = ibc.WalletAmount{
		Address: ibcTranslatorContractAddr,
		Denom: gaiaIbcAsset1Denom,
		Amount: 10002,
	}
	gaiaHeight, err = gaia.Height(ctx)
	require.NoError(t, err)
	gaiaIbcTx, err = gaia.SendIBCTransfer(ctx, gaiaToWormChannel.ChannelID, gaiaUser.KeyName, transfer, ibc.TransferOptions{Memo: ccIbcHooksMsg})
	require.NoError(t, err)
	
	// wait for transfer to ack
	_, err = testutil.PollForAck(ctx, gaia, gaiaHeight, gaiaHeight+30, gaiaIbcTx.Packet)
	require.NoError(t, err)
	err = testutil.WaitForBlocks(ctx, 1, wormchain, gaia)
	require.NoError(t, err)

	// wait for transfer
	err = testutil.WaitForBlocks(ctx, 3, wormchain)
	require.NoError(t, err)
	
	coins, err := wormchain.AllBalances(ctx, ibcTranslatorContractAddr)
	require.NoError(t, err)
	fmt.Println("Ibc Translator contract coins: ", coins)

	coins, err = wormchain.AllBalances(ctx, wormchainFaucetAddr)
	require.NoError(t, err)
	fmt.Println("Wormchain faucet coins: ", coins)

	coins, err = gaia.AllBalances(ctx, gaiaUser.Bech32Address(gaia.Config().Bech32Prefix))
	require.NoError(t, err)
	fmt.Println("Gaia user coins: ", coins)
	
	coins, err = osmosis.AllBalances(ctx, osmoUser1.Bech32Address(osmosis.Config().Bech32Prefix))
	require.NoError(t, err)
	fmt.Println("Osmo user1 coins: ", coins)

	coins, err = osmosis.AllBalances(ctx, osmoUser2.Bech32Address(osmosis.Config().Bech32Prefix))
	require.NoError(t, err)
	fmt.Println("Osmo user2 coins: ", coins)

	// IBC hooks required:
	// Send a bridged token from foreign cosmos chain, over ibc to wormchain and out of wormchain (stretch, nice-to-have)

	// PFM required:
	// Send a bridged token from a foreign cosmos chain, through wormchain, to a second foreign cosmos chain (deposited to addr)
	// Send a bridged token from a foreign cosmos chain, through wormchain, to a second foreign cosmos chain's contract

	// Out of scope:
	// Send a cosmos chain native asset to wormchain for external chain consumption

	err = testutil.WaitForBlocks(ctx, 2, wormchain)
	require.NoError(t, err)
}
