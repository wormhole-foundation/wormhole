package ictest

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/strangelove-ventures/interchaintest/v7"
	"github.com/strangelove-ventures/interchaintest/v7/chain/cosmos"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	wasmdante "github.com/wormhole-foundation/wormchain/x/wormhole/ante"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestWasmdAnteDecorator(t *testing.T) {
	// Base setup
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)
	chains := CreateLocalChain(t, *guardians)
	_, ctx, _, _, _, _ := BuildInterchain(t, chains)

	// Chains
	wormchain := chains[0].(*cosmos.CosmosChain)

	users := interchaintest.GetAndFundTestUsers(t, ctx, "default", math.NewInt(10_000_000_000), wormchain)
	user := users[0]

	// === PART #1 ===
	// Store contract via wasm (fails)
	//
	_, err := wormchain.StoreContract(ctx, "faucet", "./contracts/ibc_hooks.wasm")
	require.Error(t, err)
	require.Contains(t, err.Error(), wasmdante.ErrNotSupported().Error())

	// === PART #2 ===
	// Store wormhole core contract via Wormhole (pass)
	//
	coreContractCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/wormhole_core.wasm", guardians)
	fmt.Println("Core contract code id: ", coreContractCodeId)

	// === PART #3 ===
	// Instantiate contract via wasm (fails)
	_, err = wormchain.InstantiateContract(ctx, "faucet", coreContractCodeId, "{}", true)
	require.Error(t, err)
	require.Contains(t, err.Error(), wasmdante.ErrNotSupported().Error())

	// === PART #4 ===
	// Instantiate contract via Wormhole (pass)
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, WormchainConfig, vaa.ChainIDWormchain, guardians)
	coreContractAddr := helpers.InstantiateContract(t, ctx, wormchain, "faucet", coreContractCodeId, "wormhole_core", coreInstantiateMsg, guardians)
	fmt.Println("Core contract address: ", coreContractAddr)

	// === PART #5 ===
	// Add helper contracts for executing wormchain core contract

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

	// === PART #6 ===
	// Execute contract via wasm (pass)
	tbRegisterChainMsg := helpers.TbRegisterChainMsg(t, ExternalChainId, ExternalChainEmitterAddr, guardians)
	_, err = wormchain.ExecuteContract(ctx, "faucet", tbContractAddr, string(tbRegisterChainMsg))
	require.NoError(t, err)

	// === PART #7 ===
	// Test misc wasm messages (all should fail)
	_, err = wormchain.MigrateContract(ctx, "faucet", coreContractAddr, coreContractCodeId, "{}")
	require.Error(t, err)
	require.Contains(t, err.Error(), wasmdante.ErrNotSupported().Error())

	node := wormchain.FullNodes[0]

	// Clear contract admin (fails)
	cmd := []string{"wasm", "clear-contract-admin", coreContractAddr}
	_, err = node.ExecTx(ctx, user.KeyName(), cmd...)
	require.Error(t, err)
	require.Contains(t, err.Error(), wasmdante.ErrNotSupported().Error())

	faucetBz, err := wormchain.GetAddress(ctx, "faucet")
	require.NoError(t, err)
	faucetAddr := sdk.MustBech32ifyAddressBytes(wormchain.Config().Bech32Prefix, faucetBz)
	fmt.Println("Wormchain faucet addr: ", faucetAddr)

	// Set contract admin (fails)
	cmd = []string{"wasm", "set-contract-admin", coreContractAddr, faucetAddr}
	_, err = node.ExecTx(ctx, user.KeyName(), cmd...)
	require.Error(t, err)
	require.Contains(t, err.Error(), wasmdante.ErrNotSupported().Error())

	// Set contract label (fails)
	cmd = []string{"wasm", "set-contract-label", coreContractAddr, "label"}
	_, err = node.ExecTx(ctx, user.KeyName(), cmd...)
	require.Error(t, err)
	require.Contains(t, err.Error(), wasmdante.ErrNotSupported().Error())
}
