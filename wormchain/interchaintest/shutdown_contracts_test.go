package ictest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v7/ibc"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers/cw_wormhole"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// TestShutdownCoreContract tests the endpoints of a contract in its "shutdown" state
func TestShutdownCoreContract(t *testing.T) {
	// Setup chain and contract
	numVals := 1
	oldGuardians := guardians.CreateValSet(t, numVals)
	chain := createSingleNodeCluster(t, *oldGuardians)
	ctx, _ := buildSingleNodeInterchain(t, chain)

	// Chains
	wormchain := chain.(*cosmos.CosmosChain)

	// Users
	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), math.NewInt(1), wormchain)
	user := users[0]

	// Deploy contract to wormhole
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, WormchainConfig, vaa.ChainIDWormchain, oldGuardians)
	contractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/wormhole_core.wasm", "wormhole_core", coreInstantiateMsg, oldGuardians)
	contractAddr := contractInfo.Address

	// Store a new version of the contract to upgrade to
	wormNewCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/shutdown_wormhole_core.wasm", oldGuardians)

	// Submit contract upgrade
	err := cw_wormhole.SubmitContractUpgrade(t, ctx, oldGuardians, wormchain, contractAddr, wormNewCodeId)
	require.NoError(t, err)

	// -----------------------------------

	// Try to post a message - should fail
	message := []byte("test message")
	messageBase64 := base64.StdEncoding.EncodeToString(message)
	nonce := 1

	executeMsg, err := json.Marshal(cw_wormhole.ExecuteMsg{
		PostMessage: &cw_wormhole.ExecuteMsg_PostMessage{
			Message: cw_wormhole.Binary(messageBase64),
			Nonce:   nonce,
		},
	})
	require.NoError(t, err)

	// Execute contract
	_, err = wormchain.ExecuteContract(ctx, "faucet", contractAddr, string(executeMsg))
	require.Error(t, err)

	// -----------------------------------

	// Try to set fees - should fail
	_, err = cw_wormhole.SubmitFeeUpdate(t, ctx, oldGuardians, wormchain, contractAddr, "1000000", false)
	require.Error(t, err)

	// -----------------------------------

	// Try to transfer fees - should fail
	_, err = cw_wormhole.SubmitTransferFee(t, ctx, oldGuardians, wormchain, contractAddr, user.Address(), "10000000000", false)
	require.Error(t, err)

	// -----------------------------------

	// Try to submit a guardian set update - should pass
	initialIndex := int(helpers.QueryConsensusGuardianSetIndex(t, wormchain, ctx))
	signingGuardians := guardians.CreateValSet(t, numVals)

	newGuardians := signingGuardians
	err = cw_wormhole.SubmitGuardianSetUpdate(t, ctx, wormchain, contractAddr, newGuardians, uint32(initialIndex+1), oldGuardians)
	require.NoError(t, err)
	cw_wormhole.VerifyGuardianSet(t, ctx, wormchain, contractAddr, newGuardians, initialIndex+1)

	// -----------------------------------

	// Migrate contract back to original contract id
	err = cw_wormhole.SubmitContractUpgrade(t, ctx, signingGuardians, wormchain, contractAddr, contractInfo.ContractInfo.CodeID)
	require.NoError(t, err)
}

// TestShutdownTokenBridge tests the endpoints of a contract in its "shutdown" state
// The shutdown contract only allows the following: Upgrading the Contract & Registering a new chain.
func TestShutdownTokenBridge(t *testing.T) {
	// Setup chain and contract
	numVals := 1
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

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(10_000_000_000), gaia, osmosis, osmosis)
	gaiaUser := users[0]
	osmoUser1 := users[1]
	osmoUser2 := users[2]

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

	// Store a new version of the token bridge to upgrade to
	tbNewCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/shutdown_token_bridge.wasm", guardians)

	// Submit contract upgrade
	err = cw_wormhole.SubmitContractUpgrade(t, ctx, guardians, wormchain, tbContractAddr, tbNewCodeId)
	require.NoError(t, err)

	// -----------------------------------

	// Registering a new chain - should pass
	tbRegisterChainMsg := helpers.TbRegisterChainMsg(t, ExternalChainId, ExternalChainEmitterAddr, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", tbContractAddr, string(tbRegisterChainMsg))
	require.NoError(t, err)

	// -----------------------------------

	// Registering a new token - should fail
	tbRegisterForeignAssetMsg := helpers.TbRegisterForeignAsset(t, Asset1ContractAddr, Asset1ChainID, ExternalChainEmitterAddr, Asset1Decimals, Asset1Symbol, Asset1Name, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", tbContractAddr, string(tbRegisterForeignAssetMsg))
	require.Error(t, err)
	require.ErrorContains(t, err, "InvalidVAAAction")

	// -----------------------------------

	// Upgrade contract back to original contract id - should pass
	err = cw_wormhole.SubmitContractUpgrade(t, ctx, guardians, wormchain, tbContractAddr, tbContractCodeId)
	require.NoError(t, err)

	// -----------------------------------

	// Setup chains in "full" mode in preparation for transfer vaas

	// Now register a new token - should pass on original contract
	tbRegisterForeignAssetMsg = helpers.TbRegisterForeignAsset(t, Asset1ContractAddr, Asset1ChainID, ExternalChainEmitterAddr, Asset1Decimals, Asset1Symbol, Asset1Name, guardians)
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

	ibcHooksCodeId, err := osmosis.StoreContract(ctx, osmoUser1.KeyName(), "./contracts/ibc_hooks.wasm")
	require.NoError(t, err)
	fmt.Println("IBC hooks code id: ", ibcHooksCodeId)

	ibcHooksContractAddr, err := osmosis.InstantiateContract(ctx, osmoUser1.KeyName(), ibcHooksCodeId, "{}", true)
	require.NoError(t, err)
	fmt.Println("IBC hooks contract addr: ", ibcHooksContractAddr)

	// -----------------------------------

	// Upgrade contract to shutdown contract
	err = cw_wormhole.SubmitContractUpgrade(t, ctx, guardians, wormchain, tbContractAddr, tbNewCodeId)
	require.NoError(t, err)

	// -----------------------------------

	// Send transfer vaas - all should fail

	// Create and process a simple ibc payload3: Transfers 10.000_018 of asset1 from external chain through wormchain to gaia user
	simplePayload := helpers.CreateGatewayIbcTokenBridgePayloadTransfer(t, GaiaChainID, gaiaUser.FormattedAddress(), 0, 1)
	externalSender := []byte{1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8, 1, 2, 3, 4, 5, 6, 7, 8}
	payload3 := helpers.CreatePayload3(wormchain.Config(), AmountExternalToGaiaUser1.Uint64(), Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg := helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.Error(t, err)
	require.ErrorContains(t, err, "Invalid during shutdown mode")

	// Create and process a simple ibc payload3: Transfers 1.000_001 of asset1 from external chain through wormchain to osmo user1
	simplePayload = helpers.CreateGatewayIbcTokenBridgePayloadTransfer(t, OsmoChainID, osmoUser1.FormattedAddress(), 0, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), AmountExternalToOsmoUser1.Uint64(), Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, simplePayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.Error(t, err)
	require.ErrorContains(t, err, "Invalid during shutdown mode")

	// Create and process a contract controlled ibc payload3
	// Transfers 1.000_002 of asset1 from external chain through wormchain to ibc hooks contract addr
	// IBC hooks is used to route the contract controlled payload to a test contract which forwards tokens to osmo user2
	ibcHooksPayload := helpers.CreateIbcHooksMsg(t, ibcHooksContractAddr, osmoUser2.FormattedAddress())
	contractControlledPayload := helpers.CreateGatewayIbcTokenBridgePayloadTransferWithPayload(t, OsmoChainID, ibcHooksContractAddr, ibcHooksPayload, 1)
	payload3 = helpers.CreatePayload3(wormchain.Config(), AmountExternalToOsmoUser2.Uint64(), Asset1ContractAddr, Asset1ChainID, ibcTranslatorContractAddr, uint16(vaa.ChainIDWormchain), externalSender, contractControlledPayload)
	completeTransferAndConvertMsg = helpers.IbcTranslatorCompleteTransferAndConvertMsg(t, ExternalChainId, ExternalChainEmitterAddr, payload3, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", ibcTranslatorContractAddr, completeTransferAndConvertMsg)
	require.Error(t, err)
	require.ErrorContains(t, err, "Invalid during shutdown mode")
}
