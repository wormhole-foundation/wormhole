package ictest

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/docker/docker/client"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/strangelove-ventures/interchaintest/v4"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/ibc"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"
	"go.uber.org/zap/zaptest"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

func TestCwWormholeHappyPath(t *testing.T) {
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

// TestPostMessage tests the PostMessage function of the cw_wormhole contract
func TestPostMessage(t *testing.T) {
	// Setup chain and contract like in TestCwWormholeHappyPath
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

	// Custom response type to handle string numbers
	type TxResponse struct {
		Code uint32              `json:"code"`
		Logs sdk.ABCIMessageLogs `json:"logs"`
	}

	// Query and parse the response
	txResult, _, err := wormchain.Validators[0].ExecQuery(ctx, "tx", txHash)
	require.NoError(t, err)

	var txResponse TxResponse
	err = json.Unmarshal(txResult, &txResponse)
	require.NoError(t, err)

	require.Equal(t, uint32(0), txResponse.Code, "tx should succeed")

	// Find the wasm event
	var wasmEvent *sdk.StringEvent
	for _, log := range txResponse.Logs {
		for _, event := range log.Events {
			if event.Type == "wasm" {
				wasmEvent = &event
				break
			}
		}
	}
	require.NotNil(t, wasmEvent, "wasm event not found")

	// Helper to find attribute value
	findAttribute := func(key string) string {
		for _, attr := range wasmEvent.Attributes {
			if attr.Key == key {
				return attr.Value
			}
		}
		return ""
	}

	// Verify key attributes
	require.Equal(t, contractAddr, findAttribute("_contract_address"), "incorrect contract address")
	require.Equal(t, hex.EncodeToString(message), findAttribute("message.message"), "incorrect message")
	require.Equal(t, "1", findAttribute("message.nonce"), "incorrect nonce")
	require.Equal(t, "0", findAttribute("message.sequence"), "incorrect sequence")

	// Verify additional attributes exist (values may vary)
	require.NotEmpty(t, findAttribute("message.chain_id"), "chain_id should be present")
	require.NotEmpty(t, findAttribute("message.sender"), "sender should be present")
	require.NotEmpty(t, findAttribute("message.block_time"), "block_time should be present")
}

func TestUpdateGuardianSet(t *testing.T) {
	// Setup chain and contract
	numVals := 2
	oldGuardians := guardians.CreateValSet(t, numVals)
	chains := createWormchainChains(t, "v2.24.2", *oldGuardians)
	ctx, _ := buildMultipleChainsInterchain(t, chains)
	wormchain := chains[0].(*cosmos.CosmosChain)

	// Deploy contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, oldGuardians)
	contractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, oldGuardians)
	contractAddr := contractInfo.Address

	// Helper function to create and submit guardian set update VAA
	submitGuardianSetUpdate := func(newGuardians *guardians.ValSet, newIndex uint32, signingGuardians *guardians.ValSet) error {
		// Create guardian set update payload
		guardianKeys := make([]ethcommon.Address, len(newGuardians.Vals))
		for i, g := range newGuardians.Vals {
			copy(guardianKeys[i][:], g.Addr)
		}

		updateMsg := vaa.BodyGuardianSetUpdate{
			Keys:     guardianKeys,
			NewIndex: newIndex,
		}

		payload, err := updateMsg.Serialize()
		require.NoError(t, err)

		// Generate and sign the governance VAA using the signing guardian set
		guardianSetIndex := helpers.QueryConsensusGuardianSetIndex(t, wormchain, ctx)
		govVaa := helpers.GenerateGovernanceVaa(uint32(guardianSetIndex), signingGuardians, payload)
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
		if err != nil {
			return err
		}

		// Wait for transaction
		return testutil.WaitForBlocks(ctx, 2, wormchain)
	}

	// Helper to verify guardian set state
	verifyGuardianSet := func(expectedGuardians *guardians.ValSet, expectedIndex int) {
		var guardianSetResp cw_wormhole.GuardianSetQueryResponse
		err := wormchain.QueryContract(ctx, contractAddr, cw_wormhole.QueryMsg{
			GuardianSetInfo: &cw_wormhole.QueryMsg_GuardianSetInfo{},
		}, &guardianSetResp)
		require.NoError(t, err)

		require.Equal(t, len(expectedGuardians.Vals), len(guardianSetResp.Data.Addresses), "unexpected number of guardians")
		require.Equal(t, expectedIndex, guardianSetResp.Data.GuardianSetIndex, "unexpected guardian set index")

		for i, val := range expectedGuardians.Vals {
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
			require.True(t, found, "guardian %d not found in guardian set", i)
		}
	}

	// Get initial guardian set index
	initialIndex := int(helpers.QueryConsensusGuardianSetIndex(t, wormchain, ctx))

	signingGuardians := guardians.CreateValSet(t, numVals+1)

	t.Run("successful update", func(t *testing.T) {
		newGuardians := signingGuardians
		err := submitGuardianSetUpdate(newGuardians, uint32(initialIndex+1), oldGuardians)
		require.NoError(t, err)
		verifyGuardianSet(newGuardians, initialIndex+1)
	})

	t.Run("invalid guardian set index", func(t *testing.T) {
		// Try to update with non-sequential index
		newGuardians := guardians.CreateValSet(t, numVals+1)
		err := submitGuardianSetUpdate(newGuardians, uint32(initialIndex+3), signingGuardians)
		require.Error(t, err)

		// Try to update with same index
		err = submitGuardianSetUpdate(newGuardians, uint32(initialIndex), signingGuardians)
		require.Error(t, err)
	})

	t.Run("empty guardian set", func(t *testing.T) {
		emptyGuardians := guardians.CreateValSet(t, 0)
		err := submitGuardianSetUpdate(emptyGuardians, uint32(initialIndex+1), signingGuardians)
		require.Error(t, err)
		require.Contains(t, err.Error(), "GuardianSignatureError")
	})

	t.Run("duplicate guardians", func(t *testing.T) {
		// Create guardian set with duplicate addresses
		dupGuardians := guardians.CreateValSet(t, 1)
		dupGuardians.Vals = append(dupGuardians.Vals, dupGuardians.Vals[0])
		dupGuardians.Total = 2

		err := submitGuardianSetUpdate(dupGuardians, uint32(initialIndex+1), signingGuardians)
		require.Error(t, err)
		require.Contains(t, err.Error(), "GuardianSignatureError")
	})

	t.Run("wrong signing guardian set", func(t *testing.T) {
		// Create new guardians and try to use them to sign the update
		wrongSigners := guardians.CreateValSet(t, numVals)
		newGuardians := guardians.CreateValSet(t, numVals+1)

		err := submitGuardianSetUpdate(newGuardians, uint32(initialIndex+1), wrongSigners)
		require.Error(t, err)
		require.Contains(t, err.Error(), "GuardianSignatureError")
	})

	t.Run("insufficient signatures", func(t *testing.T) {
		// Create a guardian set with only one signer (below quorum)
		insufficientSigners := guardians.CreateValSet(t, 1)
		newGuardians := guardians.CreateValSet(t, numVals+1)

		err := submitGuardianSetUpdate(newGuardians, uint32(initialIndex+1), insufficientSigners)
		require.Error(t, err)
		require.Contains(t, err.Error(), "NoQuorum")
	})

	// Verify latest guardian set is unchanged
	verifyGuardianSet(signingGuardians, initialIndex+1)
}

func TestContractUpgrade(t *testing.T) {
	// Setup chain and contract
	numVals := 2
	guardians := guardians.CreateValSet(t, numVals)
	chains := createWormchainChains(t, "v2.24.2", *guardians)
	ctx, _ := buildMultipleChainsInterchain(t, chains)
	wormchain := chains[0].(*cosmos.CosmosChain)

	// Deploy contract
	coreInstantiateMsg := helpers.CoreContractInstantiateMsg(t, wormchainConfig, guardians)
	contractInfo := helpers.StoreAndInstantiateWormholeContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", "wormhole_core", coreInstantiateMsg, guardians)
	contractAddr := contractInfo.Address

	// Helper function to submit contract upgrade VAA
	submitContractUpgrade := func(codeId string, expectErr bool) error {
		helpers.MigrateContract(t, ctx, wormchain, "faucet", contractAddr, codeId, "{}", guardians, expectErr)

		// Wait for transaction
		return testutil.WaitForBlocks(ctx, 2, wormchain)
	}

	// Store a new version of the contract to upgrade to
	newCodeId := helpers.StoreContract(t, ctx, wormchain, "faucet", "./contracts/cw_wormhole.wasm", guardians)

	t.Run("successful upgrade", func(t *testing.T) {
		err := submitContractUpgrade(newCodeId, false)
		require.NoError(t, err)

		contractInfo = helpers.QueryContractInfo(t, wormchain, ctx, contractAddr)
		require.NoError(t, err)
		require.Equal(t, newCodeId, contractInfo.ContractInfo.CodeID)
	})

	t.Run("invalid code id", func(t *testing.T) {
		// Try to upgrade to a non-existent code ID
		err := submitContractUpgrade("999999", true)
		require.NoError(t, err)
	})

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
