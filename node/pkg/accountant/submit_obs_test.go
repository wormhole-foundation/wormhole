package accountant

import (
	// "encoding/hex"
	"encoding/json"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseObservationResponseDataKey(t *testing.T) {
	dataJson := []byte("{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1673978163}")

	var key TransferKey
	err := json.Unmarshal(dataJson, &key)
	require.NoError(t, err)

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedResult := TransferKey{
		EmitterChain:   uint16(vaa.ChainIDEthereum),
		EmitterAddress: expectedEmitterAddress,
		Sequence:       1673978163,
	}
	assert.Equal(t, expectedResult, key)
}

func TestParseObservationResponseData(t *testing.T) {
	responsesJson := []byte("[{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1674061268},\"status\":{\"type\":\"committed\"}},{\"key\":{\"emitter_chain\":2,\"emitter_address\":\"0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16\",\"sequence\":1674061267},\"status\":{\"type\":\"error\",\"data\":\"digest mismatch for processed message\"}}]")
	var responses ObservationResponses
	err := json.Unmarshal(responsesJson, &responses)
	require.NoError(t, err)
	require.Equal(t, 2, len(responses))

	expectedEmitterAddress, err := vaa.StringToAddress("0000000000000000000000000290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)

	expectedResult0 := ObservationResponse{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674061268,
		},
		Status: ObservationResponseStatus{
			Type: "committed",
		},
	}

	expectedResult1 := ObservationResponse{
		Key: TransferKey{
			EmitterChain:   uint16(vaa.ChainIDEthereum),
			EmitterAddress: expectedEmitterAddress,
			Sequence:       1674061267,
		},
		Status: ObservationResponseStatus{
			Type: "error",
			Data: "digest mismatch for processed message",
		},
	}

	assert.Equal(t, expectedResult0, responses[0])
	assert.Equal(t, expectedResult1, responses[1])
}

func makeMsgForPackingTest(t *testing.T, sequence uint64, payloadLen int) *common.MessagePublication {
	t.Helper()
	emitterAddr, err := vaa.StringToAddress("0x0290fb167208af455bb137780163b7b7a9a10c16")
	require.NoError(t, err)
	return &common.MessagePublication{
		TxID:           []byte("0123456789abcdef0123456789abcdef"),
		Timestamp:      time.Unix(1654543099, 0),
		Nonce:          123456,
		Sequence:       sequence,
		EmitterChain:   vaa.ChainIDEthereum,
		EmitterAddress: emitterAddr,
		Payload:        make([]byte, payloadLen),
	}
}

// repeatedPayloadLens returns num copies of payloadLen, for building a batch of identical messages.
func repeatedPayloadLens(payloadLen int, num int) []int {
	lens := make([]int, num)
	for i := range lens {
		lens[i] = payloadLen
	}
	return lens
}

// indexRange returns the indices from start (inclusive) to end (exclusive).
func indexRange(start int, end int) []int {
	indices := make([]int, 0, end-start)
	for i := start; i < end; i++ {
		indices = append(indices, i)
	}
	return indices
}

// marshaledMsgSizeForBatch builds the submit_observations message for a batch the same way SubmitObservationsToContract does,
// with worst-case values for the fields other than the observations, and returns its marshaled size.
func marshaledMsgSizeForBatch(t *testing.T, batch []*common.MessagePublication) int {
	t.Helper()
	obs := make([]Observation, len(batch))
	for idx, msg := range batch {
		obs[idx] = makeObservation(msg)
	}
	obsBytes, err := json.Marshal(obs)
	require.NoError(t, err)

	sig := make(SignatureBytes, 65)
	for idx := range sig {
		sig[idx] = math.MaxUint8
	}
	msgBytes, err := json.Marshal(SubmitObservationsMsg{
		Params: SubmitObservationsParams{
			Observations:     obsBytes,
			GuardianSetIndex: math.MaxUint32,
			Signature:        SignatureType{Index: math.MaxUint32, Signature: sig},
		},
	})
	require.NoError(t, err)
	return len(msgBytes)
}

func TestSubmitObservationsMsgSizeMatchesMarshaledMsg(t *testing.T) {
	// The size the packing logic computes for a batch should exactly match the size of the real marshaled message.
	batch := []*common.MessagePublication{
		makeMsgForPackingTest(t, 1, 0),
		makeMsgForPackingTest(t, 12345678, 100),
		makeMsgForPackingTest(t, math.MaxUint64, 4000),
	}
	obs := make([]Observation, len(batch))
	for idx, msg := range batch {
		obs[idx] = makeObservation(msg)
	}
	obsBytes, err := json.Marshal(obs)
	require.NoError(t, err)

	assert.Equal(t, marshaledMsgSizeForBatch(t, batch), submitObservationsMsgSize(len(obsBytes)))
}

func TestPackObservationBatches(t *testing.T) {
	testCases := []struct {
		name        string
		payloadLens []int // One message per entry; the message's sequence number is its index.

		// maxBatchCount is the batch count limit for the test. Zero means the default limit.
		maxBatchCount int

		// maxMsgSize returns the batch size limit for the test, given the test messages. Nil means the production limit.
		maxMsgSize func(msgs []*common.MessagePublication) int

		// expBatches are the expected batches as indices into the messages. Nil means the batch composition is not asserted,
		// only the shared invariants (used when the composition would depend on hand-computed observation sizes).
		expBatches   [][]int
		expOversized []int
		minBatches   int
	}{
		{
			name:        "returns no batches for no messages",
			payloadLens: []int{},
			expBatches:  [][]int{},
		},
		{
			name:        "keeps small messages together in one batch",
			payloadLens: repeatedPayloadLens(100, 100),
			expBatches:  [][]int{indexRange(0, 100)},
		},
		{
			name:        "respects count limit",
			payloadLens: repeatedPayloadLens(100, 205),
			expBatches:  [][]int{indexRange(0, 100), indexRange(100, 200), indexRange(200, 205)},
		},
		{
			name:          "respects configured count limit",
			payloadLens:   repeatedPayloadLens(100, 5),
			maxBatchCount: 2,
			expBatches:    [][]int{{0, 1}, {2, 3}, {4}},
		},
		{
			name:        "splits on size",
			payloadLens: repeatedPayloadLens(10*1024, 10),
			minBatches:  2,
		},
		{
			name:        "defers large message to a later batch",
			payloadLens: []int{50, 2000, 50},
			maxMsgSize: func(msgs []*common.MessagePublication) int {
				// Fits the large message on its own (with a little headroom), but not together with a small one.
				return submitObservationsMsgSize(marshaledObservationSize(msgs[1]) + 10)
			},
			expBatches: [][]int{{0, 2}, {1}},
		},
		{
			name:        "isolates oversized message in its own batch",
			payloadLens: []int{50, 5000, 50},
			maxMsgSize: func(msgs []*common.MessagePublication) int {
				// Cannot fit the huge message even on its own.
				return submitObservationsMsgSize(marshaledObservationSize(msgs[1]) - 10)
			},
			expBatches:   [][]int{{0, 2}, {1}},
			expOversized: []int{1},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			msgs := make([]*common.MessagePublication, len(tc.payloadLens))
			for idx, payloadLen := range tc.payloadLens {
				msgs[idx] = makeMsgForPackingTest(t, uint64(idx), payloadLen) // #nosec G115 -- test values are small
			}
			maxBatchCount := tc.maxBatchCount
			if maxBatchCount == 0 {
				maxBatchCount = DefaultSubmitObservationBatchSize
			}
			maxMsgSize := maxSubmitObservationsMsgSize
			if tc.maxMsgSize != nil {
				maxMsgSize = tc.maxMsgSize(msgs)
			}
			expOversized := make([]*common.MessagePublication, 0, len(tc.expOversized))
			for _, idx := range tc.expOversized {
				expOversized = append(expOversized, msgs[idx])
			}

			batches, oversized := packObservationBatches(msgs, maxBatchCount, maxMsgSize)

			assert.ElementsMatch(t, expOversized, oversized)
			assert.GreaterOrEqual(t, len(batches), tc.minBatches)

			// Shared invariants: every message lands in exactly one batch, and every batch respects the count limit and,
			// unless it holds an oversized message (which always gets a batch to itself), the size limit.
			var packed []*common.MessagePublication
			for _, batch := range batches {
				require.NotEmpty(t, batch)
				assert.LessOrEqual(t, len(batch), maxBatchCount)
				if !slices.Contains(expOversized, batch[0]) {
					assert.LessOrEqual(t, marshaledMsgSizeForBatch(t, batch), maxMsgSize)
				} else {
					require.Len(t, batch, 1)
				}
				packed = append(packed, batch...)
			}
			assert.ElementsMatch(t, msgs, packed)

			if tc.expBatches != nil {
				require.Len(t, batches, len(tc.expBatches))
				for batchIdx, expIndices := range tc.expBatches {
					expBatch := make([]*common.MessagePublication, 0, len(expIndices))
					for _, idx := range expIndices {
						expBatch = append(expBatch, msgs[idx])
					}
					assert.Equal(t, expBatch, batches[batchIdx], "batch %d", batchIdx)
				}
			}
		})
	}
}
