package ictest

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/btcsuite/btcd/btcutil/base58"
	sdk "github.com/cosmos/cosmos-sdk/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/strangelove-ventures/interchaintest/v7/testutil"
	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// TestUpgradeTest upgrades from v2.18.1 -> v2.18.1.1 -> V2.23.0 and:
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
func TestUpgrade(t *testing.T) {
	// Base setup
	numVals := 3
	guardians := guardians.CreateValSet(t, numVals)

	chains := CreateChain(t, *guardians, ibc.DockerImage{
		Repository: WormchainRemoteRepo,
		Version:    "v2.24.3.2",
		UidGid:     WormchainImage.UidGid,
	})

	_, ctx, r, eRep, client, _ := BuildInterchain(t, chains)

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

	// ******************************************************************************************
	// ********* Upgrade to new version of wormchain (non-state breaking upgrade) ***************
	// ******************************************************************************************

	blocksAfterUpgrade := int64(5)

	haltHeight, err := wormchain.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade1")

	err = wormchain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	// upgrade version on all nodes
	wormchain.UpgradeVersion(ctx, client, WormchainRemoteRepo, "v2.24.9")

	err = wormchain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx, timeoutCtxCancel := context.WithTimeout(ctx, time.Second*20)
	defer timeoutCtxCancel()

	err = testutil.WaitForBlocks(timeoutCtx, int(blocksAfterUpgrade), wormchain)
	require.NoError(t, err, "chain did not produce blocks after upgrade1")

	height, err := wormchain.Height(ctx)
	require.NoError(t, err, "error fetching height after upgrade1")
	fmt.Println("Checked height: ", height)

	require.GreaterOrEqual(t, height, haltHeight+blocksAfterUpgrade, "height did not increment enough after upgrade1")

	fmt.Println("***** PASS upgrade #1 **********")

	// --------------------------------------------------------------------------------------
	// upgrade version on all nodes
	blocksAfterUpgrade = int64(10)

	height, err = wormchain.Height(ctx)
	require.NoError(t, err, "error fetching height before upgrade2")
	fmt.Println("Height at sending schedule upgrade: ", height)

	haltHeight = height + blocksAfterUpgrade
	fmt.Println("Height for scheduled upgrade2: ", haltHeight)

	helpers.ScheduleUpgrade(t, ctx, wormchain, "faucet", "v3.0.0", uint64(haltHeight), guardians)

	timeoutCtx3, timeoutCtxCancel3 := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel3()

	// this should timeout due to chain halt at upgrade height.
	testutil.WaitForBlocks(timeoutCtx3, int(blocksAfterUpgrade)+2, wormchain)

	height, err = wormchain.Height(ctx)
	require.NoError(t, err, "error fetching height after chain should have halted")
	fmt.Println("Height when chains should have halted: ", height)

	require.Equal(t, haltHeight, height, "height is not equal to halt height")

	// bring down nodes to prepare for upgrade
	err = wormchain.StopAllNodes(ctx)
	require.NoError(t, err, "error stopping node(s)")

	// upgrade version on all nodes
	wormchain.UpgradeVersion(ctx, client, WormchainLocalRepo, WormchainLocalVersion)

	// start all nodes back up.
	// validators reach consensus on first block after upgrade height
	// and chain block production resumes.
	err = wormchain.StartAllNodes(ctx)
	require.NoError(t, err, "error starting upgraded node(s)")

	timeoutCtx4, timeoutCtxCancel4 := context.WithTimeout(ctx, time.Second*45)
	defer timeoutCtxCancel4()

	err = testutil.WaitForBlocks(timeoutCtx4, int(blocksAfterUpgrade)+1, wormchain)
	require.NoError(t, err, "chain did not produce blocks after upgrade")

	height, err = wormchain.Height(ctx)
	require.NoError(t, err, "error fetching height after upgrade")
	fmt.Println("Height after upgrade >10 blocks after scheduled halt: ", height)

	require.GreaterOrEqual(t, height, haltHeight+blocksAfterUpgrade, "height did not increment enough after upgrade")
	fmt.Println("***** PASS 2nd upgrade **********")

	// *************************************************************
	// ******************* Continue with test **********************
	// *************************************************************

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
}
