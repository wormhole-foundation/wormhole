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

func TestHasGuardianSigned(t *testing.T) {
	tests := []struct {
		name           string
		signatures     string
		guardianIndex  int
		expectedResult bool
	}{
		{
			name:           "guardian 0 signed (bit 0 set)",
			signatures:     "1",
			guardianIndex:  0,
			expectedResult: true,
		},
		{
			name:           "guardian 0 not signed (bit 0 not set)",
			signatures:     "2",
			guardianIndex:  0,
			expectedResult: false,
		},
		{
			name:           "guardian 1 signed (bit 1 set)",
			signatures:     "2",
			guardianIndex:  1,
			expectedResult: true,
		},
		{
			name:           "guardian 0 and 1 signed (bits 0 and 1 set)",
			signatures:     "3",
			guardianIndex:  0,
			expectedResult: true,
		},
		{
			name:           "guardian 0 and 1 signed, check guardian 1",
			signatures:     "3",
			guardianIndex:  1,
			expectedResult: true,
		},
		{
			name:           "guardian 0 and 1 signed, check guardian 2",
			signatures:     "3",
			guardianIndex:  2,
			expectedResult: false,
		},
		{
			name:           "all 19 guardians signed",
			signatures:     "524287", // 2^19 - 1 = 0x7FFFF
			guardianIndex:  18,
			expectedResult: true,
		},
		{
			name:           "all 19 guardians signed, check guardian 19",
			signatures:     "524287",
			guardianIndex:  19,
			expectedResult: false,
		},
		{
			name:           "large signature value with guardian 63 signed",
			signatures:     "9223372036854775808", // 2^63
			guardianIndex:  63,
			expectedResult: true,
		},
		{
			name:           "empty signatures string",
			signatures:     "",
			guardianIndex:  0,
			expectedResult: false,
		},
		{
			name:           "invalid signatures string",
			signatures:     "not-a-number",
			guardianIndex:  0,
			expectedResult: false,
		},
		{
			name:           "zero signatures",
			signatures:     "0",
			guardianIndex:  0,
			expectedResult: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := hasGuardianSigned(tc.signatures, tc.guardianIndex)
			assert.Equal(t, tc.expectedResult, result)
		})
	}
}

func TestParseAllPendingTransfersResponse(t *testing.T) {
	responsesJson := []byte(`{"pending":[{"key":{"emitter_chain":2,"emitter_address":"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16","sequence":1674568234},"data":[{"digest":"1nbbff/7/ai9GJUs4h2JymFuO4+XcasC6t05glXc99M=","tx_hash":"CjHx8zExnr4JU8ewAu5/tXM6a5QyslKufGHZNSr0aE8=","signatures":"3","guardian_set_index":0,"emitter_chain":2}]}]}`)
	var response AllPendingTransfersResponse
	err := json.Unmarshal(responsesJson, &response)
	require.NoError(t, err)
	require.Equal(t, 1, len(response.Pending))

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedDigest, err := hex.DecodeString("d676db7dfffbfda8bd18952ce21d89ca616e3b8f9771ab02eadd398255dcf7d3")
	require.NoError(t, err)

	expectedTxHash, err := hex.DecodeString("0a31f1f331319ebe0953c7b002ee7fb5733a6b9432b252ae7c61d9352af4684f")
	require.NoError(t, err)

	expectedResult := PendingTransfer{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674568234,
		},
		Data: []PendingTransferData{
			{
				Digest:           expectedDigest,
				TxHash:           expectedTxHash,
				Signatures:       "3",
				GuardianSetIndex: 0,
				EmitterChain:     uint16(vaa.ChainIDEthereum),
			},
		},
	}

	assert.True(t, reflect.DeepEqual(expectedResult, response.Pending[0]))
}

// AllPendingTransfersQueryConnMock allows us to mock all_pending_transfers by implementing SubmitQuery.
type AllPendingTransfersQueryConnMock struct {
	// pages holds the response data for each page, keyed by page index
	pages     [][]byte
	pageIndex int
	err       error
}

func (qc *AllPendingTransfersQueryConnMock) SubmitQuery(ctx context.Context, contractAddress string, query []byte) ([]byte, error) {
	if qc.err != nil {
		return nil, qc.err
	}

	if qc.pageIndex >= len(qc.pages) {
		// Return empty response if we've exhausted pages
		return []byte(`{"pending":[]}`), nil
	}

	resp := qc.pages[qc.pageIndex]
	qc.pageIndex++
	return resp, nil
}

func TestQueryAllPendingTransfersPage(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	queryResp := createPendingTransfersForTest(t, 3, 0, "1")
	qc := &AllPendingTransfersQueryConnMock{pages: [][]byte{queryResp}}

	pending, err := queryAllPendingTransfersPage(ctx, logger, qc, "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465", nil, 500)
	require.NoError(t, err)
	require.Equal(t, 3, len(pending))
}

func TestQueryAllPendingTransfersEmpty(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	qc := &AllPendingTransfersQueryConnMock{pages: [][]byte{[]byte(`{"pending":[]}`)}}

	pending, err := queryAllPendingTransfersWithConn(ctx, logger, qc, "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465", 500)
	require.NoError(t, err)
	require.Equal(t, 0, len(pending))
}

func TestQueryAllPendingTransfersPagination(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	page1 := createPendingTransfersForTest(t, 3, 0, "1")
	page2 := createPendingTransfersForTest(t, 2, 0, "1")

	qc := &AllPendingTransfersQueryConnMock{pages: [][]byte{page1, page2}}

	// Use pageSize=3 so first page is "full" and triggers fetching second page
	pending, err := queryAllPendingTransfersWithConn(ctx, logger, qc, "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465", 3)
	require.NoError(t, err)
	require.Equal(t, 5, len(pending)) // 3 from page1 + 2 from page2
}

func TestQueryAllPendingTransfersPaginationMultipleFullPages(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	page1 := createPendingTransfersForTest(t, 3, 0, "1")
	page2 := createPendingTransfersForTest(t, 3, 0, "1")
	emptyPage := []byte(`{"pending":[]}`)

	qc := &AllPendingTransfersQueryConnMock{pages: [][]byte{page1, page2, emptyPage}}

	// Use pageSize=3 so first two pages are "full" and trigger fetching next page
	// Third page is empty which terminates pagination
	pending, err := queryAllPendingTransfersWithConn(ctx, logger, qc, "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465", 3)
	require.NoError(t, err)
	require.Equal(t, 6, len(pending)) // 3 from page1 + 3 from page2 + 0 from empty page
}

func TestQueryAllPendingTransfersQueryError(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	qc := &AllPendingTransfersQueryConnMock{err: errors.New("query failed")}

	_, err := queryAllPendingTransfersWithConn(ctx, logger, qc, "wormhole14hj2tavq8fpesdwxxcu44rty3hh90vhujrvcmstl4zr3txmfvw9srrg465", 500)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "query failed")
}

func TestQueryAllPendingTransfersMultipleDataEntries(t *testing.T) {
	// Test parsing a response where a single pending transfer has multiple data entries
	// (which can happen when the same transfer has observations from different guardian sets)
	responsesJson := []byte(`{"pending":[{"key":{"emitter_chain":2,"emitter_address":"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16","sequence":100},"data":[{"digest":"AAAA","tx_hash":"BBBB","signatures":"1","guardian_set_index":0,"emitter_chain":2},{"digest":"CCCC","tx_hash":"DDDD","signatures":"2","guardian_set_index":1,"emitter_chain":2}]}]}`)

	var response AllPendingTransfersResponse
	err := json.Unmarshal(responsesJson, &response)
	require.NoError(t, err)
	require.Equal(t, 1, len(response.Pending))
	require.Equal(t, 2, len(response.Pending[0].Data))

	assert.Equal(t, uint32(0), response.Pending[0].Data[0].GuardianSetIndex)
	assert.Equal(t, "1", response.Pending[0].Data[0].Signatures)
	assert.Equal(t, uint32(1), response.Pending[0].Data[1].GuardianSetIndex)
	assert.Equal(t, "2", response.Pending[0].Data[1].Signatures)
}
