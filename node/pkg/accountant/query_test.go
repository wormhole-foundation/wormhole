package accountant

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	cosmossdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
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
	responsesJson := []byte("{\"details\":[{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":3},\"status\":{\"pending\":[{\"digest\":\"65hmAN4IbW9MBnSDzYmgoD/3ze+F8ik9NGeKR/vQ4J4=\",\"tx_hash\":\"CjHx8zExnr4JU8ewAu5/tXM6a5QyslKufGHZNSr0aE8=\",\"signatures\":\"1\",\"guardian_set_index\":0,\"emitter_chain\":2}]}}]}")
	var response BatchTransferStatusResponse
	err := json.Unmarshal(responsesJson, &response)
	require.NoError(t, err)
	require.Equal(t, 1, len(response.Details))

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedDigest, err := hex.DecodeString("eb986600de086d6f4c067483cd89a0a03ff7cdef85f2293d34678a47fbd0e09e")
	require.NoError(t, err)

	expectedTxHash, err := hex.DecodeString("0a31f1f331319ebe0953c7b002ee7fb5733a6b9432b252ae7c61d9352af4684f")
	require.NoError(t, err)

	expectedResult := TransferDetails{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       3,
		},
		Status: &TransferStatus{
			Pending: &[]TransferStatusPending{
				TransferStatusPending{
					Digest:           expectedDigest,
					TxHash:           expectedTxHash,
					Signatures:       "1",
					GuardianSetIndex: 0,
					EmitterChain:     uint16(vaa.ChainIDEthereum),
				},
			},
		},
	}

	// Use DeepEqual() because the response contains pointers.
	assert.True(t, reflect.DeepEqual(expectedResult, response.Details[0]))
}

// BatchTransferStatusQueryConnMock allows us to mock batch_transfer_status by implementing SubmitQuery.
type BatchTransferStatusQueryConnMock struct {
	resp []byte
}

func (qc *BatchTransferStatusQueryConnMock) SubmitQuery(ctx context.Context, contractAddress string, query []byte) ([]byte, error) {
	// Force a failure if the query is much bigger than what we are allowing. This does not have to be exact, since the chunking tests will be using a lot more than that.
	// A json encoded transfer key is about 150 characters.
	if len(query) > 150*maxPendingsPerQuery+1000 {
		return []byte{}, errors.New("query too large")
	}

	return qc.resp, nil
}

// validateBatchTransferStatusResults makes sure the query returned everything expected, and nothing extra.
func validateBatchTransferStatusResults(t *testing.T, keys []TransferKey, transferDetails map[string]*TransferStatus) {
	for _, key := range keys {
		tKey := key.String()
		_, exists := transferDetails[tKey]
		require.Equal(t, true, exists)
		delete(transferDetails, tKey)
	}

	require.Equal(t, 0, len(transferDetails))
}

func TestBatchTransferStatusForExactlyOneTransfer(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	keys, queryResp := createTransferKeysForTestingBatchTransferStatus(t, 1)
	require.Equal(t, 1, len(keys))
	qc := &BatchTransferStatusQueryConnMock{resp: queryResp}

	transferDetails, err := queryBatchTransferStatusWithConn(ctx, logger, qc, "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465", keys)
	require.NoError(t, err)
	require.Equal(t, len(keys), len(transferDetails))
	validateBatchTransferStatusResults(t, keys, transferDetails)
}

func TestBatchTransferStatusForExactlyOneChunk(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	keys, queryResp := createTransferKeysForTestingBatchTransferStatus(t, maxPendingsPerQuery)
	require.Equal(t, maxPendingsPerQuery, len(keys))
	qc := &BatchTransferStatusQueryConnMock{resp: queryResp}

	transferDetails, err := queryBatchTransferStatusWithConn(ctx, logger, qc, "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465", keys)
	require.NoError(t, err)
	require.Equal(t, len(keys), len(transferDetails))
	validateBatchTransferStatusResults(t, keys, transferDetails)
}

func TestBatchTransferStatusForExactlyOneChunkPlus1(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	keys, queryResp := createTransferKeysForTestingBatchTransferStatus(t, maxPendingsPerQuery+1)
	require.Equal(t, maxPendingsPerQuery+1, len(keys))
	qc := &BatchTransferStatusQueryConnMock{resp: queryResp}

	transferDetails, err := queryBatchTransferStatusWithConn(ctx, logger, qc, "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465", keys)
	require.NoError(t, err)
	require.Equal(t, len(keys), len(transferDetails))
	validateBatchTransferStatusResults(t, keys, transferDetails)
}

func TestBatchTransferStatusMultipleChunks(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	keys, queryResp := createTransferKeysForTestingBatchTransferStatus(t, -1)
	require.Less(t, maxPendingsPerQuery, len(keys))
	qc := &BatchTransferStatusQueryConnMock{resp: queryResp}

	transferDetails, err := queryBatchTransferStatusWithConn(ctx, logger, qc, "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465", keys)
	require.NoError(t, err)
	require.Equal(t, len(keys), len(transferDetails))
	validateBatchTransferStatusResults(t, keys, transferDetails)
}
