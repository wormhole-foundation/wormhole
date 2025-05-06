package ictest

import (
	"fmt"
	"strconv"
	"testing"

	"cosmossdk.io/math"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
)

var (
	GaiaChainID = uint16(11)
	OsmoChainID = uint16(12)

	ExternalChainId          = uint16(123)
	ExternalChainEmitterAddr = "0x123EmitterAddress"

	Asset1Name         = "Wrapped BTC"
	Asset1Symbol       = "XBTC"
	Asset1ContractAddr = "0xXBTC"
	Asset1ChainID      = ExternalChainId
	Asset1Decimals     = uint8(6)

	AmountExternalToGaiaUser1       = math.NewInt(10_000_018)
	AmountExternalToOsmoUser1       = math.NewInt(1_000_001)
	AmountExternalToOsmoUser2       = math.NewInt(1_000_002)
	AmountGaiaUser1ToExternalSimple = math.NewInt(1_000_003)
	AmountGaiaUser1ToExternalCC     = math.NewInt(1_000_004)
	AmountGaiaUser1ToOsmoUser1      = math.NewInt(1_000_005)
	AmountGaiaUser1ToOsmoUser2      = math.NewInt(1_000_006)
)

// TestWormchain runs through a simple test case for each deliverable
//   - Setup wormchain, gaia, and osmosis including contracts/allowlists/etc
//   - External->Cosmos: Send 10.000_018 to gaia user 1 (simple)
//   - External->Cosmos: Send 1.000_001 to osmo user 1 (simple)
//   - External->Cosmos: Send 1.000_002 to osmo user 2 (contract controlled via osmo ibc-hooks contract)
//   - Cosmos->External: Send 1.000_003 to external address (simple) from gaia user 1
//     -- gaia user 1 now has 9.000_015 of asset 1
//   - Cosmos->External: Send 1.000_004 to external address (contract controlled) from gaia user 1
//     -- gaia user 1 now has 8.000_011 of asset 1
//   - Cosmos->Cosmos: Send 1.000_005 to osmo user 1 (simple) from gaia user 1
//     -- gaia user 1 now has 7.000_006 of asset 1
//     -- osmo user 1 now has 2.000_006 of asset 1
//   - Cosmos->Cosmos: Send 1.000_006 to osmo user 2 (contract controlled via osmo ibc-hooks contract) from gaia user 1
//     -- gaia user 1 now has 6.000_000 of asset 1
//     -- osmo user 2 now has 2.000_008 of asset 1
//   - Verify asset 1 balance of gaia user 1, osmo user 1, osmo user 2, and cw20 contract total supply
func TestWormchain(t *testing.T) {
	// Base setup
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)
	chains := CreateLocalChain(t, *guardians)
	_, ctx, r, eRep, _, _ := BuildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)
	gaia := chains[1].(*cosmos.CosmosChain)
	osmosis := chains[2].(*cosmos.CosmosChain)

	wormchainFaucetAddrBz, err := wormchain.GetAddress(ctx, "faucet")
	require.NoError(t, err)
	wormchainFaucetAddr := sdk.MustBech32ifyAddressBytes(wormchain.Config().Bech32Prefix, wormchainFaucetAddrBz)
	fmt.Println("Wormchain faucet addr: ", wormchainFaucetAddr)

	osmoToWormChannel, err := ibc.GetTransferChannel(ctx, r, eRep, osmosis.Config().ChainID, wormchain.Config().ChainID)
	require.NoError(t, err)
	wormToOsmoChannel := osmoToWormChannel.Counterparty
	gaiaToWormChannel, err := ibc.GetTransferChannel(ctx, r, eRep, gaia.Config().ChainID, wormchain.Config().ChainID)
	require.NoError(t, err)
	wormToGaiaChannel := gaiaToWormChannel.Counterparty

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(10_000_000_000), wormchain, gaia, osmosis, osmosis)
	_ = users[0] // Wormchain user
	gaiaUser := users[1]
	osmoUser1 := users[2]
	osmoUser2 := users[3]

	ibcHooksCodeId, err := osmosis.StoreContract(ctx, osmoUser1.KeyName(), "./contracts/ibc_hooks.wasm")
	require.NoError(t, err)
	fmt.Println("IBC hooks code id: ", ibcHooksCodeId)

	ibcHooksContractAddr, err := osmosis.InstantiateContract(ctx, osmoUser1.KeyName(), ibcHooksCodeId, "{}", true)
	require.NoError(t, err)
	fmt.Println("IBC hooks contract addr: ", ibcHooksContractAddr)

	err = testutil.WaitForBlocks(ctx, 2, wormchain)
	require.NoError(t, err, "error waiting for 2 blocks")

	// Store wormhole core contract
	coreContractCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/wormhole_core.wasm", guardians)
	fmt.Println("Core contract code id: ", coreContractCodeId)

	// Instantiate wormhole core contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, WormchainConfig, vaa.ChainIDWormchain, guardians)
	coreContractAddr := helpers.InstantiateContract(t, ctx, wormchain, "faucet", coreContractCodeId, "wormhole_core", coreInstantiateMsg, guardians)
	fmt.Println("Core contract address: ", coreContractAddr)

	// Store cw20_wrapped_2 contract
	wrappedAssetCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/cw20_wrapped_2.wasm", guardians)
	fmt.Println("CW20 wrapped_2 code id: ", wrappedAssetCodeId)

	// Store token bridge contract
	tbContractCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/token_bridge.wasm", guardians)
	fmt.Println("Token bridge contract code id: ", tbContractCodeId)

	// Instantiate token bridge contract
	tbInstantiateMsg := helpers.TbContractInstantiateMsg(t, WormchainConfig, coreContractAddr, wrappedAssetCodeId)
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
	require.NoError(t, err)

	// Allowlist worm/gaia chain id / channel
	wormGaiaAllowlistMsg := helpers.SubmitUpdateChainToChannelMapMsg(t, GaiaChainID, wormToGaiaChannel.ChannelID, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, wormGaiaAllowlistMsg)
	require.NoError(t, err)

	// Create and process a simple ibc payload3: Transfers 10.000_018 of asset1 from external chain through wormchain to gaia user
	simplePayload := helpers.CreateGatewayIbcTokenBridgePayloadTransfer(t, GaiaChainID, gaiaUser.FormattedAddress(), 0, 1)
	externalSender := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}
	payload3 := helpers.CreatePayload3(wormchain.Config(), AmountExternalToGaiaUser1.Uint64(), Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg := helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)

	// Create and process a simple ibc payload3: Transfers 1.000_001 of asset1 from external chain through wormchain to osmo user1
	simplePayload = helpers.CreateGatewayIbcTokenBridgePayloadTransfer(t, OsmoChainID, osmoUser1.FormattedAddress(), 0, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), AmountExternalToOsmoUser1.Uint64(), Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)

	// Create and process a contract controlled ibc payload3
	// Transfers 1.000_002 of asset1 from external chain through wormchain to ibc hooks contract addr
	// IBC hooks is used to route the contract controlled payload to a test contract which forwards tokens to osmo user2
	ibcHooksPayload := helpers.CreateIbcHooksMsg(t, ibcHooksContractAddr, osmoUser2.FormattedAddress())
	contractControlledPayload := helpers.CreateGatewayIbcTokenBridgePayloadTransferWithPayload(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), AmountExternalToOsmoUser2.Uint64(), Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.NoError(t, err)

	// wait for transfer to ack
	err = testutil.WaitForBlocks(ctx, 10, wormchain, gaia)
	require.NoError(t, err)

	// Query the CW20 address of asset1
	var tbQueryRsp helpers.TbQueryRsp
	tbQueryReq := helpers.CreateCW20Query(t, Asset1ChainID, Asset1ContractAddr)
	wormchain.QueryContract(ctx, tbContractAddr, tbQueryReq, &tbQueryRsp)
	cw20Address := tbQueryRsp.Data.Address
	fmt.Println("Asset1 cw20 addr: ", cw20Address)

	// Get the Gaia/IBC denom of asset1
	cw20AddressBz := helpers.MustAccAddressFromBech32(cw20Address, wormchain.Config().Bech32Prefix)
	subdenom := base58.Encode(cw20AddressBz)
	tokenFactoryDenom := fmt.Sprint("factory/", ibcTranslatorContractAddr, "/", subdenom)
	gaiaAsset1Denom := transfertypes.GetPrefixedDenom("transfer", gaiaToWormChannel.ChannelID, tokenFactoryDenom)
	gaiaIbcAsset1Denom := transfertypes.ParseDenomTrace(gaiaAsset1Denom).IBCDenom()

	// Get the Osmo/IBC denom of asset1
	osmoAsset1Denom := transfertypes.GetPrefixedDenom("transfer", osmoToWormChannel.ChannelID, tokenFactoryDenom)
	osmoIbcAsset1Denom := transfertypes.ParseDenomTrace(osmoAsset1Denom).IBCDenom()

	// Verify Gaia user 1 has expected asset 1 balance
	gaiaUser1Asset1BalanceTemp, err := gaia.GetBalance(ctx, gaiaUser.FormattedAddress(), gaiaIbcAsset1Denom)
	require.NoError(t, err)
	fmt.Println("Gaia user asset1 coins: ", gaiaUser1Asset1BalanceTemp)

	// wait for transfer to ack
	err = testutil.WaitForBlocks(ctx, 2, wormchain, gaia)
	require.NoError(t, err)

	// Verify Gaia user 1 has expected asset 1 balance
	gaiaUser1Asset1BalanceTemp, err = gaia.GetBalance(ctx, gaiaUser.FormattedAddress(), gaiaIbcAsset1Denom)
	require.NoError(t, err)
	fmt.Println("Gaia user asset1 coins: ", gaiaUser1Asset1BalanceTemp)

	// *************  Cosmos->External: Simple payload (wormhole-mw + ibc-hooks)  ****************
	// Send 1.000_003 asset 1 from gaia user 1 to external
	simpleMemo := helpers.CreateIbcComposabilityMwMemoGatewayTransfer(t, Asset1ChainID, externalSender, 0, 1)
	transfer := ibc.WalletAmount{
		Address: ibcTranslatorContractAddr,
		Denom:   gaiaIbcAsset1Denom,
		Amount:  AmountGaiaUser1ToExternalSimple,
	}
	_, err = gaia.SendIBCTransfer(ctx, gaiaToWormChannel.ChannelID, gaiaUser.KeyName(), transfer, ibc.TransferOptions{Memo: simpleMemo})
	require.NoError(t, err)

	// wait for transfer to ack
	err = testutil.WaitForBlocks(ctx, 2, wormchain, gaia)
	require.NoError(t, err)

	// *************  Cosmos->External: Contract controlled payload (wormhole-mw + ibc-hooks)  ****************
	// Send 1.000_004 asset 1 from gaia user 1 to external
	ccIbcHooksMsg := helpers.CreateIbcComposabilityMwMemoGatewayTransferWithPayload(t, Asset1ChainID, externalSender, []byte("ExternalContractPayload"), 1)
	transfer = ibc.WalletAmount{
		Address: ibcTranslatorContractAddr,
		Denom:   gaiaIbcAsset1Denom,
		Amount:  AmountGaiaUser1ToExternalCC,
	}
	_, err = gaia.SendIBCTransfer(ctx, gaiaToWormChannel.ChannelID, gaiaUser.KeyName(), transfer, ibc.TransferOptions{Memo: ccIbcHooksMsg})
	require.NoError(t, err)

	// wait for transfer to ack
	err = testutil.WaitForBlocks(ctx, 2, wormchain, gaia)
	require.NoError(t, err)

	// **************  Cosmos->Cosmos: Simple payload (wormhole-mw + PFM)  ****************
	// Send 1.000_005 asset 1 from gaia user 1 to osmo user 1
	simplePfmMsg := helpers.CreateIbcComposabilityMwMemoGatewayTransfer(t, OsmoChainID, []byte(osmoUser1.FormattedAddress()), 0, 1)
	transfer = ibc.WalletAmount{
		Address: wormchainFaucetAddr,
		Denom:   gaiaIbcAsset1Denom,
		Amount:  AmountGaiaUser1ToOsmoUser1,
	}
	_, err = gaia.SendIBCTransfer(ctx, gaiaToWormChannel.ChannelID, gaiaUser.KeyName(), transfer, ibc.TransferOptions{
		Timeout: &ibc.IBCTimeout{
			NanoSeconds: 30_000_000_000,
		},
		Memo: simplePfmMsg,
	})
	require.NoError(t, err)

	// wait for transfer to ack
	err = testutil.WaitForBlocks(ctx, 2, wormchain, gaia)
	require.NoError(t, err)

	// **************  Cosmos->Cosmos: Contract controlled payload (wormhole-mw + PFM)  ****************
	// Send 1.000_006 asset 1 from gaia user 1 to osmo user 2
	ccPayload := helpers.CreateIbcHooksMsg(t, ibcHooksContractAddr, osmoUser2.FormattedAddress())
	ccPfmMsg := helpers.CreateIbcComposabilityMwMemoGatewayTransferWithPayload(t, OsmoChainID, []byte(ibcHooksContractAddr), ccPayload, 1)
	transfer = ibc.WalletAmount{
		Address: ibcTranslatorContractAddr,
		Denom:   gaiaIbcAsset1Denom,
		Amount:  AmountGaiaUser1ToOsmoUser2,
	}
	_, err = gaia.SendIBCTransfer(ctx, gaiaToWormChannel.ChannelID, gaiaUser.KeyName(), transfer, ibc.TransferOptions{
		Timeout: &ibc.IBCTimeout{
			NanoSeconds: 30_000_000_000,
		},
		Memo: ccPfmMsg,
	})
	require.NoError(t, err)

	// wait for transfer to ack
	err = testutil.WaitForBlocks(ctx, 15, wormchain, gaia)
	require.NoError(t, err)

	// Verify Gaia user 1 has expected asset 1 balance
	gaiaUser1Asset1Balance, err := gaia.GetBalance(ctx, gaiaUser.FormattedAddress(), gaiaIbcAsset1Denom)
	require.NoError(t, err)
	expectedGaiaUser1Amount := AmountExternalToGaiaUser1.Sub(AmountGaiaUser1ToExternalCC).Sub(AmountGaiaUser1ToExternalSimple).Sub(AmountGaiaUser1ToOsmoUser1).Sub(AmountGaiaUser1ToOsmoUser2)
	require.Equal(t, expectedGaiaUser1Amount.Int64(), gaiaUser1Asset1Balance.Int64())
	fmt.Println("Gaia user asset1 coins: ", gaiaUser1Asset1Balance)

	// Verify osmo user 1 has expected asset 1 balance
	osmoUser1Asset1Balance, err := osmosis.GetBalance(ctx, osmoUser1.FormattedAddress(), osmoIbcAsset1Denom)
	require.NoError(t, err)
	require.Equal(t, AmountExternalToOsmoUser1.Add(AmountGaiaUser1ToOsmoUser1).Int64(), osmoUser1Asset1Balance.Int64())
	fmt.Println("Osmo user1 asset1 coins: ", osmoUser1Asset1Balance)

	// Verify osmo user 2 has expected asset 1 balance
	osmoUser2Asset1Balance, err := osmosis.GetBalance(ctx, osmoUser2.FormattedAddress(), osmoIbcAsset1Denom)
	require.NoError(t, err)
	require.Equal(t, AmountExternalToOsmoUser2.Add(AmountGaiaUser1ToOsmoUser2).Int64(), osmoUser2Asset1Balance.Int64())
	fmt.Println("Osmo user2 asset1 coins: ", osmoUser2Asset1Balance)

	// Verify asset 1 cw20 contract has expected final total supply
	var cw20QueryRsp helpers.Cw20WrappedQueryRsp
	cw20QueryReq := helpers.Cw20WrappedQueryMsg{TokenInfo: helpers.Cw20TokenInfo{}}
	wormchain.QueryContract(ctx, cw20Address, cw20QueryReq, &cw20QueryRsp)
	fmt.Println("Asset1 supply: ", cw20QueryRsp.Data.TotalSupply)
	totalSupply, err := strconv.ParseUint(cw20QueryRsp.Data.TotalSupply, 10, 64)
	require.NoError(t, err)
	expectedTotalSupply := AmountExternalToGaiaUser1.Add(AmountExternalToOsmoUser1).Add(AmountExternalToOsmoUser2).Sub(AmountGaiaUser1ToExternalSimple).Sub(AmountGaiaUser1ToExternalCC)
	require.Equal(t, expectedTotalSupply.Uint64(), totalSupply)

	denomsMetadata := helpers.GetDenomsMetadata(t, ctx, wormchain)
	fmt.Println("Denoms metadata: ", denomsMetadata)
}

type QueryMsg struct {
	GuardianSetInfo *struct{} `json:"guardian_set_info,omitempty"`
}

type QueryRsp struct {
	Data *QueryRspObj `json:"data,omitempty"`
}

type QueryRspObj struct {
	GuardianSetIndex uint32                    `json:"guardian_set_index"`
	Addresses        []helpers.GuardianAddress `json:"addresses"`
}
