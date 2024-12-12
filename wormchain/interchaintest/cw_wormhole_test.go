package ictest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/docker/docker/client"
	"github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"
	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers/cw_wormhole"

	vaa "github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func createWormchainChains(t *testing.T, wormchainVersion string, guardians guardians.ValSet) []ibc.Chain {
	numWormchainVals := len(guardians.Vals)
	numFullNodes := 0

	wormchainConfig.Images[0].Version = wormchainVersion
	wormchainConfig.ModifyGenesis = ModifyGenesis(votingPeriod, maxDepositPeriod, guardians, true)

	cf := interchaintest.NewBuiltinChainFactory(zaptest.NewLogger(t), []*interchaintest.ChainSpec{
		{
			ChainName:     "wormchain",
			ChainConfig:   wormchainConfig,
			NumValidators: &numWormchainVals,
			NumFullNodes:  &numFullNodes,
		},
	})

	// Get chains from the chain factory
	chains, err := cf.Chains(t.Name())
	require.NoError(t, err)

	return chains
}

func buildMultipleChainsInterchain(t *testing.T, chains []ibc.Chain) (context.Context, *client.Client) {
	ic := interchaintest.NewInterchain()

	for _, chain := range chains {
		ic.AddChain(chain)
	}

	ctx := context.Background()
	client, network := interchaintest.DockerSetup(t)

	err := ic.Build(ctx, nil, interchaintest.InterchainBuildOptions{
		TestName:         t.Name(),
		Client:           client,
		NetworkID:        network,
		SkipPathCreation: true,
	})
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = ic.Close()
	})

	return ctx, client
}

// TestCWWormholeQueries tests the query functions of the cw_wormhole contract
func TestCWWormholeQueries(t *testing.T) {
	// Base setup
	numVals := 1
	guardians := guardians.CreateValSet(t, numVals)

	chains := createWormchainChains(t, "v2.24.2", *guardians)
	ctx, _ := buildMultipleChainsInterchain(t, chains)

	wormchain := chains[0].(*cosmos.CosmosChain)

	// Instantiate the cw_wormhole contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	wormchainCoreContractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, guardians)
	contractAddr := wormchainCoreContractInfo.Address

	// Query the contract to check that the guardian set is correct
	var guardianSetResp cw_wormhole.GuardianSetQueryResponse
	err := wormchain.QueryContract(ctx, contractAddr, cw_wormhole.QueryMsg{
		GuardianSetInfo: &cw_wormhole.QueryMsg_GuardianSetInfo{},
	}, &guardianSetResp)
	require.NoError(t, err)
	require.Equal(t, numVals, len(guardianSetResp.Data.Addresses), "guardian set should have the correct number of guardians")
	// Check that all the guardians from the query are the ones in the running valset
	for _, val := range guardians.Vals {
		found := false
		for _, guardian := range guardianSetResp.Data.Addresses {
			decoded, err := base64.StdEncoding.DecodeString(string(guardian.Bytes))
			require.NoError(t, err)
			guardianDecodedBytes := []byte(decoded)
			if bytes.Equal(val.Addr, guardianDecodedBytes) {
				found = true
				break
			}
		}
		require.True(t, found, "guardian not found in guardian set")
	}

	// Check that the core contract fee is set to 0uworm
	var stateResp cw_wormhole.GetStateQueryResponse
	err = wormchain.QueryContract(ctx, contractAddr, cw_wormhole.QueryMsg{
		GetState: &cw_wormhole.QueryMsg_GetState{},
	}, &stateResp)
	require.NoError(t, err)
	require.Equal(t, "uworm", stateResp.Data.Fee.Denom, "core contract fee should be in uworm")
	require.Equal(t, cw_wormhole.Uint128("0"), stateResp.Data.Fee.Amount, "core contract fee should be 0")

	// Check that hex addresse are able to be queried
	var hexAddressResp cw_wormhole.QueryAddressHexQueryResponse
	err = wormchain.QueryContract(ctx, contractAddr, cw_wormhole.QueryMsg{
		QueryAddressHex: &cw_wormhole.QueryMsg_QueryAddressHex{
			Address: "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465",
		},
	}, &hexAddressResp)
	require.NoError(t, err)
	require.IsType(t, "", hexAddressResp.Data.Hex, "hex address should be a string")

	// Check that the core contract can properly verify a VAA
	guardianSetIndex := helpers.QueryConsensusGuardianSetIndex(t, wormchain, ctx)
	vaa := helpers.GenerateGovernanceVaa(uint32(guardianSetIndex), guardians, []byte("test"))
	vaaBz, err := vaa.Marshal()
	require.NoError(t, err)
	encodedVaa := base64.StdEncoding.EncodeToString(vaaBz)
	vaaBinary := cw_wormhole.Binary(encodedVaa)

	currentWormchainBlock, err := wormchain.Height(ctx)
	require.NoError(t, err)

	var parsedVaaResponse cw_wormhole.VerifyVAAQueryResponse
	err = wormchain.QueryContract(ctx, contractAddr, cw_wormhole.QueryMsg{
		VerifyVaa: &cw_wormhole.QueryMsg_VerifyVAA{
			BlockTime: int(currentWormchainBlock),
			Vaa:       vaaBinary,
		},
	}, &parsedVaaResponse)
	require.NoError(t, err)
	require.NotNil(t, parsedVaaResponse.Data, "VAA should be verified")
	require.Equal(t, "test", string(parsedVaaResponse.Data.Payload), "VAA payload should be what we passed in")
}

// TestCWWormholePostMessage tests the PostMessage function of the cw_wormhole contract
func TestCWWormholePostMessage(t *testing.T) {
	numVals := 1
	guardians := guardians.CreateValSet(t, numVals)
	chains := createWormchainChains(t, "v2.24.2", *guardians)
	ctx, _ := buildMultipleChainsInterchain(t, chains)
	wormchain := chains[0].(*cosmos.CosmosChain)

	// Instantiate contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	contractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, guardians)
	contractAddr := contractInfo.Address

	// Create message and encode to base64
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
	txHash, err := wormchain.ExecuteContract(ctx, "faucet", contractAddr, string(executeMsg))
	require.NoError(t, err)

	// Wait 2 blocks
	err = testutil.WaitForBlocks(ctx, 2, wormchain)
	require.NoError(t, err)

	// Query and parse the response
	txResult, _, err := wormchain.Validators[0].ExecQuery(ctx, "tx", txHash)
	require.NoError(t, err)

	var txResponse cw_wormhole.TxResponse
	err = json.Unmarshal(txResult, &txResponse)
	require.NoError(t, err)

	// Verify event attributes
	cw_wormhole.VerifyEventAttributes(t, &txResponse, map[string]string{
		"_contract_address": contractAddr,
		"message.message":   hex.EncodeToString(message),
		"message.nonce":     "1",
		"message.sequence":  "0",
	})
}

// TestCWWormholeUpdateGuardianSet tests the UpdateGuardianSet function of the cw_wormhole contract
func TestCWWormholeUpdateGuardianSet(t *testing.T) {
	// Setup chain and contract
	numVals := 1
	oldGuardians := guardians.CreateValSet(t, numVals)
	chains := createWormchainChains(t, "v2.24.2", *oldGuardians)
	ctx, _ := buildMultipleChainsInterchain(t, chains)
	wormchain := chains[0].(*cosmos.CosmosChain)

	// Deploy contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, oldGuardians)
	contractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, oldGuardians)
	contractAddr := contractInfo.Address

	// Get initial guardian set index
	initialIndex := int(helpers.QueryConsensusGuardianSetIndex(t, wormchain, ctx))
	signingGuardians := guardians.CreateValSet(t, numVals+1)

	t.Run("successful update", func(t *testing.T) {
		newGuardians := signingGuardians
		err := cw_wormhole.SubmitGuardianSetUpdate(t, ctx, wormchain, contractAddr, newGuardians, uint32(initialIndex+1), oldGuardians)
		require.NoError(t, err)
		cw_wormhole.VerifyGuardianSet(t, ctx, wormchain, contractAddr, newGuardians, initialIndex+1)
	})

	t.Run("invalid guardian set index", func(t *testing.T) {
		// Try to update with non-sequential index
		newGuardians := guardians.CreateValSet(t, numVals+1)
		err := cw_wormhole.SubmitGuardianSetUpdate(t, ctx, wormchain, contractAddr, newGuardians, uint32(initialIndex+3), signingGuardians)
		require.Error(t, err)

		// Try to update with same index
		err = cw_wormhole.SubmitGuardianSetUpdate(t, ctx, wormchain, contractAddr, newGuardians, uint32(initialIndex), signingGuardians)
		require.Error(t, err)
	})

	t.Run("empty guardian set", func(t *testing.T) {
		emptyGuardians := guardians.CreateValSet(t, 0)
		err := cw_wormhole.SubmitGuardianSetUpdate(t, ctx, wormchain, contractAddr, emptyGuardians, uint32(initialIndex+1), signingGuardians)
		require.Error(t, err)
		require.Contains(t, err.Error(), "GuardianSignatureError")
	})

	t.Run("duplicate guardians", func(t *testing.T) {
		// Create guardian set with duplicate addresses
		dupGuardians := guardians.CreateValSet(t, 1)
		dupGuardians.Vals = append(dupGuardians.Vals, dupGuardians.Vals[0])
		dupGuardians.Total = 2

		err := cw_wormhole.SubmitGuardianSetUpdate(t, ctx, wormchain, contractAddr, dupGuardians, uint32(initialIndex+1), signingGuardians)
		require.Error(t, err)
		require.Contains(t, err.Error(), "GuardianSignatureError")
	})

	t.Run("wrong signing guardian set", func(t *testing.T) {
		// Create new guardians and try to use them to sign the update
		wrongSigners := guardians.CreateValSet(t, numVals)
		newGuardians := guardians.CreateValSet(t, numVals+1)

		err := cw_wormhole.SubmitGuardianSetUpdate(t, ctx, wormchain, contractAddr, newGuardians, uint32(initialIndex+1), wrongSigners)
		require.Error(t, err)
		require.Contains(t, err.Error(), "governance VAAs must be signed by the current guardian set")
	})

	t.Run("insufficient signatures", func(t *testing.T) {
		// Create a guardian set with only one signer (below quorum)
		insufficientSigners := guardians.CreateValSet(t, 1)
		newGuardians := guardians.CreateValSet(t, numVals+1)

		err := cw_wormhole.SubmitGuardianSetUpdate(t, ctx, wormchain, contractAddr, newGuardians, uint32(initialIndex+1), insufficientSigners)
		require.Error(t, err)
		require.Contains(t, err.Error(), "governance VAAs must be signed by the current guardian set")

		// too many signatures
		insufficientSigners = guardians.CreateValSet(t, numVals+2)
		err = cw_wormhole.SubmitGuardianSetUpdate(t, ctx, wormchain, contractAddr, newGuardians, uint32(initialIndex+1), insufficientSigners)
		require.Error(t, err)
		require.Contains(t, err.Error(), "GuardianSignatureError")
	})

	// Verify signing validators did not change
	cw_wormhole.VerifyGuardianSet(t, ctx, wormchain, contractAddr, signingGuardians, initialIndex+1)
}

// TestCWWormholeContractUpgrade tests the SubmitContractUpgrade function of the cw_wormhole contract
func TestCWWormholeContractUpgrade(t *testing.T) {
	// Setup chain and contract
	numVals := 1
	guardians := guardians.CreateValSet(t, numVals)
	chains := createWormchainChains(t, "v2.24.2", *guardians)
	ctx, _ := buildMultipleChainsInterchain(t, chains)
	wormchain := chains[0].(*cosmos.CosmosChain)

	// Deploy contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	contractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, guardians)
	contractAddr := contractInfo.Address

	// Store a new version of the contract to upgrade to
	newCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", guardians)

	t.Run("successful upgrade", func(t *testing.T) {
		err := cw_wormhole.SubmitContractUpgrade(t, ctx, guardians, wormchain, contractAddr, newCodeId)
		require.NoError(t, err)

		contractInfo = helpers.QueryContractInfo(t, wormchain, ctx, contractAddr)
		require.NoError(t, err)
		require.Equal(t, newCodeId, contractInfo.ContractInfo.CodeID)
	})

	t.Run("invalid code id", func(t *testing.T) {
		// Try to upgrade to a non-existent code ID
		err := cw_wormhole.SubmitContractUpgrade(t, ctx, guardians, wormchain, contractAddr, "999999")
		require.Error(t, err)
	})

	// VAA payload to upgrade contract is not allowed on Wormchain, must use the wormhole module
	t.Run("invalid: use x/wormhole", func(t *testing.T) {
		// Left pad code ID to 32 bytes
		paddedBuf, err := vaa.LeftPadBytes(newCodeId, 32)
		require.NoError(t, err)

		var newContract vaa.Address
		copy(newContract[:], paddedBuf.Bytes())

		// Create contract upgrade payload
		updateMsg := vaa.BodyContractUpgrade{
			ChainID:     vaa.ChainIDWormchain,
			NewContract: newContract,
		}

		payload, err := updateMsg.Serialize()
		require.NoError(t, err)

		// Generate and sign the governance VAA
		guardianSetIndex := helpers.QueryConsensusGuardianSetIndex(t, wormchain, ctx)
		govVaa := helpers.GenerateGovernanceVaa(uint32(guardianSetIndex), guardians, payload)
		vaaBz, err := govVaa.Marshal()
		require.NoError(t, err)

		encodedVaa := base64.StdEncoding.EncodeToString(vaaBz)
		executeVAAPayload, err := json.Marshal(cw_wormhole.ExecuteMsg{
			SubmitVaa: &cw_wormhole.ExecuteMsg_SubmitVAA{
				Vaa: cw_wormhole.Binary(encodedVaa),
			},
		})
		require.NoError(t, err)

		// Submit VAA
		_, err = wormchain.ExecuteContract(ctx, "faucet", contractAddr, string(executeVAAPayload))
		require.Error(t, err)
		require.Contains(t, err.Error(), "must use x/wormhole")
	})
}

// TestCWWormholeSetFee tests the SetFee function of the cw_wormhole contract
func TestCWWormholeSetFee(t *testing.T) {
	// Setup chain and contract
	numVals := 1
	guardians := guardians.CreateValSet(t, numVals)
	chains := createWormchainChains(t, "v2.24.2", *guardians)
	ctx, _ := buildMultipleChainsInterchain(t, chains)
	wormchain := chains[0].(*cosmos.CosmosChain)

	// Deploy contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	contractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, guardians)
	contractAddr := contractInfo.Address

	// wrapper around helper function for cleaner test code
	submitFeeUpdate := func(amount string, replay bool) (*cw_wormhole.TxResponse, error) {
		return cw_wormhole.SubmitFeeUpdate(t, ctx, guardians, wormchain, contractAddr, amount, replay)
	}

	t.Run("successful fee update", func(t *testing.T) {
		txResponse, err := submitFeeUpdate("1000000", true) // Set fee to 1 WORM (1000000 uworm)
		require.NoError(t, err)
		cw_wormhole.VerifyEventAttributes(t, txResponse, map[string]string{
			"action":         "fee_change",
			"new_fee.amount": "1000000",
			"new_fee.denom":  "uworm",
		})
		cw_wormhole.VerifyFee(t, ctx, wormchain, contractAddr, "1000000")
	})

	t.Run("zero fee", func(t *testing.T) {
		txResponse, err := submitFeeUpdate("0", false)
		require.NoError(t, err)
		cw_wormhole.VerifyEventAttributes(t, txResponse, map[string]string{
			"action":         "fee_change",
			"new_fee.amount": "0",
			"new_fee.denom":  "uworm",
		})
		cw_wormhole.VerifyFee(t, ctx, wormchain, contractAddr, "0")
	})

	t.Run("very large fee", func(t *testing.T) {
		txResponse, err := submitFeeUpdate("1000000000000", false) // 1M WORM
		require.NoError(t, err)
		cw_wormhole.VerifyEventAttributes(t, txResponse, map[string]string{
			"action":         "fee_change",
			"new_fee.amount": "1000000000000",
			"new_fee.denom":  "uworm",
		})
		cw_wormhole.VerifyFee(t, ctx, wormchain, contractAddr, "1000000000000")
	})
}

// TestCWWormholeTransferFees tests transferring the accumulated fees to the core contract
func TestCWWormholeTransferFees(t *testing.T) {
	// Setup chain and contract
	numVals := 1
	guardians := guardians.CreateValSet(t, numVals)
	chains := createWormchainChains(t, "v2.24.2", *guardians)
	ctx, _ := buildMultipleChainsInterchain(t, chains)
	wormchain := chains[0].(*cosmos.CosmosChain)

	// Deploy contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	contractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, guardians)
	contractAddr := contractInfo.Address

	users := interchaintest.GetAndFundTestUsers(t, ctx, t.Name(), 1, wormchain)
	user := users[0]
	userAddr := user.Bech32Address("wormhole")

	t.Run("successful fee transfer", func(t *testing.T) {
		// Set fee to 1000000 uworm
		_, err := cw_wormhole.SubmitFeeUpdate(t, ctx, guardians, wormchain, contractAddr, "1000000", false)
		require.NoError(t, err)

		// Post some messages with fees to build up balance
		err = cw_wormhole.PostMessageWithFee(t, ctx, wormchain, contractAddr, "message1", 1000000)
		require.NoError(t, err)
		err = cw_wormhole.PostMessageWithFee(t, ctx, wormchain, contractAddr, "message2", 1000000)
		require.NoError(t, err)

		// Get recipient's initial balance
		initialBalance, err := cw_wormhole.GetUwormBalance(t, ctx, wormchain, userAddr)
		require.NoError(t, err)

		// Transfer 1500000 uworm
		_, err = cw_wormhole.SubmitTransferFee(t, ctx, guardians, wormchain, contractAddr, []byte(user.Address), "1500000", true)
		require.NoError(t, err)

		// Verify successful transfer
		finalBalance, err := cw_wormhole.GetUwormBalance(t, ctx, wormchain, userAddr)
		require.NoError(t, err)
		require.Equal(t, initialBalance+1500000, finalBalance)
	})

	t.Run("transfer more than balance", func(t *testing.T) {
		_, err := cw_wormhole.SubmitTransferFee(t, ctx, guardians, wormchain, contractAddr, []byte(user.Address), "10000000000", false)
		require.Error(t, err)
		require.Contains(t, err.Error(), "insufficient funds")
	})

	t.Run("invalid recipient", func(t *testing.T) {
		_, err := cw_wormhole.SubmitTransferFee(t, ctx, guardians, wormchain, contractAddr, []byte("invalid"), "1000000", false)
		require.Error(t, err)
	})

	t.Run("zero amount - invalid coins", func(t *testing.T) {
		_, err := cw_wormhole.SubmitTransferFee(t, ctx, guardians, wormchain, contractAddr, []byte(user.Address), "0", false)
		require.Error(t, err)
	})
}
