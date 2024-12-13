package cw_wormhole

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"math/big"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/strangelove-ventures/interchaintest/v4/chain/cosmos"
	"github.com/strangelove-ventures/interchaintest/v4/testutil"

	"github.com/stretchr/testify/require"

	"github.com/wormhole-foundation/wormchain/interchaintest/guardians"
	"github.com/wormhole-foundation/wormchain/interchaintest/helpers"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

type GuardianSetQueryResponse struct {
	Data GuardianSetInfoResponse `json:"data"`
}

type VerifyVAAQueryResponse struct {
	Data ParsedVAA `json:"data"`
}

type GetStateQueryResponse struct {
	Data GetStateResponse `json:"data"`
}

type QueryAddressHexQueryResponse struct {
	Data GetAddressHexResponse `json:"data"`
}

// Custom response type to handle string numbers
type TxResponse struct {
	Code uint32              `json:"code"`
	Logs sdk.ABCIMessageLogs `json:"logs"`
}

// SubmitGuardianSetUpdate submits a VAA to update the guardian set
func SubmitGuardianSetUpdate(
	t *testing.T,
	ctx context.Context,
	wormchain *cosmos.CosmosChain,
	contractAddr string,
	newGuardians *guardians.ValSet,
	newIndex uint32,
	signingGuardians *guardians.ValSet,
) error {
	// Create guardian set update payload
	guardianKeys := make([]common.Address, len(newGuardians.Vals))
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
	executeVAAPayload, err := json.Marshal(ExecuteMsg{
		SubmitVaa: &ExecuteMsg_SubmitVAA{
			Vaa: Binary(encodedVaa),
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

// VerifyGuardianSet verifies the guardian set in the contract state
func VerifyGuardianSet(
	t *testing.T,
	ctx context.Context,
	wormchain *cosmos.CosmosChain,
	contractAddr string,
	expectedGuardians *guardians.ValSet,
	expectedIndex int,
) {
	var guardianSetResp GuardianSetQueryResponse
	err := wormchain.QueryContract(ctx, contractAddr, QueryMsg{
		GuardianSetInfo: &QueryMsg_GuardianSetInfo{},
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

// SubmitContractUpgrade submits a VAA to upgrade the contract code
func SubmitContractUpgrade(
	t *testing.T,
	ctx context.Context,
	guardians *guardians.ValSet,
	wormchain *cosmos.CosmosChain,
	contractAddr string,
	newCodeId string,
) error {
	if err := helpers.MigrateContract(t, ctx, wormchain, "faucet", contractAddr, newCodeId, "{}", guardians); err != nil {
		return err
	}

	// Wait for transaction
	return testutil.WaitForBlocks(ctx, 2, wormchain)
}

func SubmitContractUpgradeWithVaa(
	t *testing.T,
	ctx context.Context,
	guardians *guardians.ValSet,
	guardianSetIndex uint64,
	vaaChainId vaa.ChainID,
	chain *cosmos.CosmosChain,
	contractAddr string,
	newCodeId string,
	keyName string,
) error {
	// convert newCodeId to Uint256
	var newCodeIdBz [32]byte
	newCodeIdInt := new(big.Int)
	newCodeIdInt.SetString(newCodeId, 10)
	newCodeIdInt.FillBytes(newCodeIdBz[:])

	// Create contract upgrade payload
	updateMsg := vaa.BodyContractUpgrade{
		ChainID:     vaaChainId,
		NewContract: vaa.Address(newCodeIdBz),
	}

	payload, err := updateMsg.Serialize()
	require.NoError(t, err)

	// Generate and sign the governance VAA
	govVaa := helpers.GenerateGovernanceVaa(uint32(guardianSetIndex), guardians, payload)
	vaaBz, err := govVaa.Marshal()
	require.NoError(t, err)

	encodedVaa := base64.StdEncoding.EncodeToString(vaaBz)
	executeVAAPayload, err := json.Marshal(ExecuteMsg{
		SubmitVaa: &ExecuteMsg_SubmitVAA{
			Vaa: Binary(encodedVaa),
		},
	})
	require.NoError(t, err)

	// Submit VAA
	_, err = chain.ExecuteContract(ctx, keyName, contractAddr, string(executeVAAPayload))
	return err
}

// SubmitFeeUpdate submits a VAA to update the fee amount
func SubmitFeeUpdate(
	t *testing.T,
	ctx context.Context,
	guardians *guardians.ValSet,
	wormchain *cosmos.CosmosChain,
	contractAddr string,
	amount string,
	replay bool,
) (*TxResponse, error) {
	// Get current guardian set index
	guardianSetIndex := helpers.QueryConsensusGuardianSetIndex(t, wormchain, ctx)

	// Create a fixed 32-byte array for the fee amount
	var amountBytes [32]byte
	amountInt := new(big.Int)
	amountInt.SetString(amount, 10)
	amountInt.FillBytes(amountBytes[:])

	// Create governance VAA payload
	// [32 bytes] Core Module
	// [1 byte]  Action (3 for set fee)
	// [2 bytes] ChainID (0 for universal)
	// [32 bytes] Amount
	buf := new(bytes.Buffer)
	buf.Write(vaa.CoreModule)
	vaa.MustWrite(buf, binary.BigEndian, vaa.ActionCoreSetMessageFee)
	vaa.MustWrite(buf, binary.BigEndian, uint16(0)) // ChainID 0 for universal
	buf.Write(amountBytes[:])

	// Generate and sign governance VAA
	govVaa := helpers.GenerateGovernanceVaa(uint32(guardianSetIndex), guardians, buf.Bytes())
	vaaBz, err := govVaa.Marshal()
	require.NoError(t, err)

	// Rest of the function remains the same...
	encodedVaa := base64.StdEncoding.EncodeToString(vaaBz)
	executeVAAPayload, err := json.Marshal(ExecuteMsg{
		SubmitVaa: &ExecuteMsg_SubmitVAA{
			Vaa: Binary(encodedVaa),
		},
	})
	require.NoError(t, err)

	// Submit VAA
	txHash, err := wormchain.ExecuteContract(ctx, "faucet", contractAddr, string(executeVAAPayload))
	if err != nil {
		return nil, err
	}

	// Wait for transaction
	err = testutil.WaitForBlocks(ctx, 2, wormchain)
	require.NoError(t, err)

	// Replay the transaction
	if replay {
		_, err = wormchain.ExecuteContract(ctx, "faucet", contractAddr, string(executeVAAPayload))
		require.Error(t, err)
		require.Contains(t, err.Error(), "VaaAlreadyExecuted")
	}

	// Query transaction result
	txResult, _, err := wormchain.Validators[0].ExecQuery(ctx, "tx", txHash)
	require.NoError(t, err)

	// Parse response
	var txResponse TxResponse
	err = json.Unmarshal(txResult, &txResponse)
	require.NoError(t, err)

	return &txResponse, nil
}

// VerifyFee verifies the fee amount in the contract state
func VerifyFee(
	t *testing.T,
	ctx context.Context,
	wormchain *cosmos.CosmosChain,
	contractAddr string,
	expectedAmount string,
) {
	var stateResp GetStateQueryResponse
	err := wormchain.QueryContract(ctx, contractAddr, QueryMsg{
		GetState: &QueryMsg_GetState{},
	}, &stateResp)
	require.NoError(t, err)
	require.Equal(t, "uworm", stateResp.Data.Fee.Denom)
	require.Equal(t, Uint128(expectedAmount), stateResp.Data.Fee.Amount)
}

// VerifyEventAttributes verifies the attributes in a wasm tx response
func VerifyEventAttributes(t *testing.T, txResponse *TxResponse, expectedAttributes map[string]string) {
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

	// Verify each expected attribute
	for key, expectedValue := range expectedAttributes {
		actualValue := findAttribute(key)
		require.Equal(t, expectedValue, actualValue,
			"unexpected value for attribute %s", key)
	}
}

// PostMessageWithFee posts a message to the contract with a fee
func PostMessageWithFee(
	t *testing.T,
	ctx context.Context,
	wormchain *cosmos.CosmosChain,
	contractAddr string,
	message string,
	fee int64,
) error {
	messageBase64 := base64.StdEncoding.EncodeToString([]byte(message))
	executeMsg, err := json.Marshal(ExecuteMsg{
		PostMessage: &ExecuteMsg_PostMessage{
			Message: Binary(messageBase64),
			Nonce:   1,
		},
	})
	require.NoError(t, err)

	funds := sdk.Coins{sdk.NewInt64Coin("uworm", fee)}
	_, err = wormchain.ExecuteContractWithAmount(ctx, "faucet", contractAddr, string(executeMsg), funds)
	return err
}

// SubmitTransferFee submits a VAA to transfer fees to an address
func SubmitTransferFee(
	t *testing.T,
	ctx context.Context,
	guardians *guardians.ValSet,
	wormchain *cosmos.CosmosChain,
	contractAddr string,
	addrBytes []byte,
	amount string,
	replay bool,
) (*TxResponse, error) {
	// Created a fixed 32-byte array for the recipient address
	var recipientBytes [32]byte
	copy(recipientBytes[32-len(addrBytes):], addrBytes)

	// Create a fixed 32-byte array for the fee amount
	var amountBytes [32]byte
	amountInt := new(big.Int)
	amountInt.SetString(amount, 10)
	amountInt.FillBytes(amountBytes[:])

	buf := new(bytes.Buffer)
	buf.Write(vaa.CoreModule)
	vaa.MustWrite(buf, binary.BigEndian, vaa.ActionCoreTransferFees)
	vaa.MustWrite(buf, binary.BigEndian, uint16(0))
	buf.Write(recipientBytes[:])
	buf.Write(amountBytes[:])

	guardianSetIndex := helpers.QueryConsensusGuardianSetIndex(t, wormchain, ctx)
	govVaa := helpers.GenerateGovernanceVaa(uint32(guardianSetIndex), guardians, buf.Bytes())
	vaaBz, err := govVaa.Marshal()
	require.NoError(t, err)

	encodedVaa := base64.StdEncoding.EncodeToString(vaaBz)
	executeVAAPayload, err := json.Marshal(ExecuteMsg{
		SubmitVaa: &ExecuteMsg_SubmitVAA{
			Vaa: Binary(encodedVaa),
		},
	})
	require.NoError(t, err)

	txHash, err := wormchain.ExecuteContract(ctx, "faucet", contractAddr, string(executeVAAPayload))
	if err != nil {
		return nil, err
	}

	err = testutil.WaitForBlocks(ctx, 2, wormchain)
	require.NoError(t, err)

	if replay {
		_, err = wormchain.ExecuteContract(ctx, "faucet", contractAddr, string(executeVAAPayload))
		require.Error(t, err)
		require.Contains(t, err.Error(), "VaaAlreadyExecuted")
	}

	txResult, _, err := wormchain.Validators[0].ExecQuery(ctx, "tx", txHash)
	require.NoError(t, err)

	var txResponse TxResponse
	err = json.Unmarshal(txResult, &txResponse)
	require.NoError(t, err)

	return &txResponse, nil
}

// GetUwormBalance returns the balance of uworm tokens for an address
func GetUwormBalance(t *testing.T, ctx context.Context, wormchain *cosmos.CosmosChain, addr string) (int64, error) {
	coins, err := wormchain.GetBalance(ctx, addr, "uworm")
	if err != nil {
		return 0, err
	}

	return coins, nil
}
