package accountant

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseMissingObservationsResponse(t *testing.T) {
	responsesJson := []byte("{\"missing\":[{\"chain_id\":2,\"tx_hash\":\"y1E+jwgozWKEVbHt9dIeErgNXlHnntQwdzymYSCmBEA=\"},{\"chain_id\":4,\"tx_hash\":\"FZyF7xR5bIwtvdBIlIrDEZc+mrCkN/ixjGazgJdJdQQ=\"}]}")
	var response MissingObservationsResponse
	err := json.Unmarshal(responsesJson, &response)
	require.NoError(t, err)
	require.Equal(t, 2, len(response.Missing))

	expectedTxHash0, err := hex.DecodeString("cb513e8f0828cd628455b1edf5d21e12b80d5e51e79ed430773ca66120a60440")
	require.NoError(t, err)

	expectedTxHash1, err := hex.DecodeString("159c85ef14796c8c2dbdd048948ac311973e9ab0a437f8b18c66b38097497504")
	require.NoError(t, err)

	expectedResult := MissingObservationsResponse{
		Missing: []MissingObservation{
			MissingObservation{
				ChainId: uint16(vaa.ChainIDEthereum),
				TxHash:  expectedTxHash0,
			},
			MissingObservation{
				ChainId: uint16(vaa.ChainIDBSC),
				TxHash:  expectedTxHash1,
			},
		},
	}

	assert.Equal(t, expectedResult, response)
}

func TestParseBatchTransferStatusCommittedResponse(t *testing.T) {
	responsesJson := []byte("{\"details\":[{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1674568234},\"status\":{\"committed\":{\"data\":{\"amount\":\"1000000000000000000\",\"token_chain\":2,\"token_address\":\"0000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a\",\"recipient_chain\":4},\"digest\":\"1nbbff/7/ai9GJUs4h2JymFuO4+XcasC6t05glXc99M=\"}}}]}")
	var response BatchTransferStatusResponse
	err := json.Unmarshal(responsesJson, &response)
	require.NoError(t, err)
	require.Equal(t, 1, len(response.Details))

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedTokenAddress, err := vaa.StringToAddress("0000000000000000000000002d8be6bf0baa74e0a907016679cae9190e80dd0a")
	require.NoError(t, err)

	expectedAmount := cosmossdk.NewInt(1000000000000000000)

	expectedDigest, err := hex.DecodeString("d676db7dfffbfda8bd18952ce21d89ca616e3b8f9771ab02eadd398255dcf7d3")
	require.NoError(t, err)

	expectedResult := TransferDetails{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674568234,
		},
		Status: &TransferStatus{
			Committed: &TransferStatusCommitted{
				Data: TransferData{
					Amount:         &expectedAmount,
					TokenChain:     uint16(vaa.ChainIDEthereum),
					TokenAddress:   expectedTokenAddress,
					RecipientChain: uint16(vaa.ChainIDBSC),
				},
				Digest: expectedDigest,
			},
		},
	}

	// Use DeepEqual() because the response contains pointers.
	assert.True(t, reflect.DeepEqual(expectedResult, response.Details[0]))
}

func TestParseBatchTransferStatusNotFoundResponse(t *testing.T) {
	responsesJson := []byte("{\"details\":[{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1674484597},\"status\":null}]}")
	var response BatchTransferStatusResponse
	err := json.Unmarshal(responsesJson, &response)
	require.NoError(t, err)
	require.Equal(t, 1, len(response.Details))

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedResult := TransferDetails{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674484597,
		},
		Status: nil,
	}

	// Use DeepEqual() because the response contains pointers.
	assert.True(t, reflect.DeepEqual(expectedResult, response.Details[0]))
}

func TestParseBatchTransferStatusPendingResponse(t *testing.T) {
	//TODO: Write this test once we get a sample response.
}
