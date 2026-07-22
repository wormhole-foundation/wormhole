package evm

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

const generatedReceiptGoldenPath = "generated_receipts.json"

// TestGeneratedReceiptGoldenVectors verifies that receipt parsing reproduces the independently
// generated digest pinned by every vector.
func TestGeneratedReceiptGoldenVectors(t *testing.T) {
	vectors := loadGeneratedReceiptGoldenVectors(t)
	require.NotEmpty(t, vectors)

	seenNames := make(map[string]struct{}, len(vectors))
	for _, vec := range vectors {
		require.NotEmpty(t, vec.Name)
		require.NotContains(t, seenNames, vec.Name, "duplicate vector name")
		seenNames[vec.Name] = struct{}{}

		t.Run(vec.Name, func(t *testing.T) {
			require.NotEmpty(t, vec.Comment)
			require.Contains(t, []vaa.ChainID{vaa.ChainIDEthereum, 4004}, vec.WormholeChainID)
			require.NotZero(t, vec.BlockTime)
			require.NotNil(t, vec.Receipt)
			require.NotEmpty(t, vec.Expected)

			msgs, logs := processGeneratedReceiptGoldenVector(t, vec, vec.Receipt)
			require.Len(t, msgs, len(vec.Expected))

			for i, msg := range msgs {
				assertGeneratedReceiptGoldenMessage(t, vec, vec.Expected[i], logs[i], msg)
			}
		})
	}
}

// Receipt metadata identifies where a message was observed but is not part of the VAA body. This
// test changes every relevant metadata field while keeping the event data and expected digest fixed.
func TestGeneratedReceiptGoldenVectorMetadataIndependence(t *testing.T) {
	vec := findGeneratedReceiptGoldenVector(t, "short-payload-seq-1")
	mutated := cloneReceipt(t, vec.Receipt)

	newTxHash := eth_common.HexToHash("0xaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	newBlockHash := eth_common.HexToHash("0xbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb")
	newBlockNumber := uint64(29_999_999)
	newTxIndex := uint(77)

	mutated.Type = types.LegacyTxType
	mutated.TxHash = newTxHash
	mutated.BlockHash = newBlockHash
	mutated.BlockNumber = new(big.Int).SetUint64(newBlockNumber)
	mutated.TransactionIndex = newTxIndex
	mutated.GasUsed = 123_456
	mutated.CumulativeGasUsed = 234_567
	mutated.Bloom = types.Bloom{0xaa, 0xbb, 0xcc}

	for _, log := range mutated.Logs {
		require.NotNil(t, log)
		log.TxHash = newTxHash
		log.BlockHash = newBlockHash
		log.BlockNumber = newBlockNumber
		log.TxIndex = uint(newTxIndex)
		log.Index += 100
	}

	mutatedVector := vec
	mutatedVector.Expected = append([]receiptExpectedMessage(nil), vec.Expected...)
	// Keep log selection strict after changing the fixture's log indices.
	for i := range mutatedVector.Expected {
		mutatedVector.Expected[i].LogIndex += 100
	}

	msgs, logs := processGeneratedReceiptGoldenVector(t, mutatedVector, mutated)
	require.Len(t, msgs, 1)
	require.Equal(t, newTxHash.Bytes(), msgs[0].TxID)
	assertGeneratedReceiptGoldenMessage(t, mutatedVector, mutatedVector.Expected[0], logs[0], msgs[0])
}

// TestGeneratedReceiptGoldenVectorsCoverage prevents regeneration from silently dropping the
// boundary cases that give this corpus its regression value.
func TestGeneratedReceiptGoldenVectorsCoverage(t *testing.T) {
	vectors := loadGeneratedReceiptGoldenVectors(t)
	require.GreaterOrEqual(t, len(vectors), 200)

	seenNames := map[string]bool{}
	seenSequences := map[uint64]bool{}
	payloadLengths := map[int]int{}
	chainIDs := map[vaa.ChainID]int{}
	consistencyLevels := map[uint8]int{}
	var sawNonceMax bool
	var sawNonRoundTimestamp bool
	var sawHighBitTimestamp bool
	var sawThirtyTwoZeroPayload bool
	var sawThirtyThreeBinaryZeroTail bool
	var sawLeadingZeroEmitter bool
	var sawTwoMessagesInOneReceipt bool

	for _, vec := range vectors {
		seenNames[vec.Name] = true
		chainIDs[vec.WormholeChainID]++
		msgs, _ := processGeneratedReceiptGoldenVector(t, vec, vec.Receipt)
		if len(msgs) == 2 {
			sawTwoMessagesInOneReceipt = true
			require.Less(t, vec.Expected[0].LogIndex, vec.Expected[1].LogIndex)
			require.Less(t, msgs[0].Sequence, msgs[1].Sequence)
		}

		for _, msg := range msgs {
			seenSequences[msg.Sequence] = true
			payloadLengths[len(msg.Payload)]++
			consistencyLevels[msg.ConsistencyLevel]++
			sawNonceMax = sawNonceMax || msg.Nonce == 0xffffffff
			sawNonRoundTimestamp = sawNonRoundTimestamp || vec.BlockTime == 0x65a1b2c3
			sawHighBitTimestamp = sawHighBitTimestamp || vec.BlockTime == 0x80000001
			sawThirtyTwoZeroPayload = sawThirtyTwoZeroPayload || bytes.Equal(msg.Payload, bytes.Repeat([]byte{0x00}, 32))
			sawThirtyThreeBinaryZeroTail = sawThirtyThreeBinaryZeroTail || (len(msg.Payload) == 33 && msg.Payload[7] == 0x00 && msg.Payload[32] == 0x00 && msg.Payload[0] >= 0x80)
			sawLeadingZeroEmitter = sawLeadingZeroEmitter || msg.EmitterAddress.String() == "00000000000000000000000000000000000000000000000000000000000000ab"
		}
	}

	requiredVectors := []string{
		"empty-payload-seq-0",
		"short-payload-seq-1",
		"payload-1-byte",
		"binary-payload-with-nuls",
		"payload-31-bytes",
		"payload-32-normal",
		"payload-32-zero",
		"payload-33-binary-zero-tail",
		"large-payload-4096",
		"leading-zero-emitter-ab",
		"multi-log-select-third",
		"multi-log-two-core-events",
		"sequence-max-minus-one",
		"sequence-uint64-max-serializer-range",
		"sequence-js-unsafe-2pow53-plus-one",
		"chain-id-4004",
		"consistency-level-203-custom",
		"consistency-level-1-finalized",
		"consistency-level-15-finalized",
	}
	for _, name := range requiredVectors {
		require.Truef(t, seenNames[name], "missing required vector %q", name)
	}

	require.True(t, seenSequences[0])
	require.True(t, seenSequences[1])
	require.True(t, seenSequences[0xfffffffffffffffe])
	require.True(t, seenSequences[0xffffffffffffffff])
	require.True(t, seenSequences[9007199254740993])
	require.True(t, sawNonceMax)
	require.True(t, sawNonRoundTimestamp)
	require.True(t, sawHighBitTimestamp)
	require.True(t, sawThirtyTwoZeroPayload)
	require.True(t, sawThirtyThreeBinaryZeroTail)
	require.True(t, sawLeadingZeroEmitter)
	require.True(t, sawTwoMessagesInOneReceipt)
	require.Positive(t, chainIDs[vaa.ChainIDEthereum])
	require.Equal(t, 1, chainIDs[4004])
	require.Positive(t, consistencyLevels[vaa.ConsistencyLevelPublishImmediately])
	require.Positive(t, consistencyLevels[vaa.ConsistencyLevelSafe])
	require.Positive(t, consistencyLevels[vaa.ConsistencyLevelFinalized])
	require.Positive(t, consistencyLevels[vaa.ConsistencyLevelCustom])
	require.Positive(t, consistencyLevels[1])
	require.Positive(t, consistencyLevels[15])

	require.Positive(t, payloadLengths[1])
	require.Positive(t, payloadLengths[31])
	require.Positive(t, payloadLengths[32])
	require.Positive(t, payloadLengths[33])
	require.Positive(t, payloadLengths[4096])
}

func loadGeneratedReceiptGoldenVectors(t *testing.T) []receiptGoldenVector {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("testdata", generatedReceiptGoldenPath))
	require.NoError(t, err)

	var vectors []receiptGoldenVector
	require.NoError(t, json.Unmarshal(data, &vectors))
	return vectors
}

func findGeneratedReceiptGoldenVector(t *testing.T, name string) receiptGoldenVector {
	t.Helper()

	for _, vec := range loadGeneratedReceiptGoldenVectors(t) {
		if vec.Name == name {
			return vec
		}
	}
	t.Fatalf("missing golden vector %q", name)
	return receiptGoldenVector{}
}

func processGeneratedReceiptGoldenVector(t *testing.T, vec receiptGoldenVector, receipt *types.Receipt) ([]*common.MessagePublication, []*types.Log) {
	t.Helper()
	require.NotNil(t, receipt)

	mock := newMockConnector(t)
	mock.receipts[receipt.TxHash] = receipt
	mock.blockTimes[receipt.BlockHash] = vec.BlockTime

	logs := validCoreBridgeLogs(t, receipt, generatedReceiptContract)
	require.Equal(t, expectedMessageLogIndices(vec.Expected), logIndices(logs))

	_, _, msgs, err := MessageEventsForTransaction(
		context.Background(),
		mock,
		generatedReceiptContract,
		vec.WormholeChainID,
		receipt.TxHash,
		true,
	)
	require.NoError(t, err)
	require.Len(t, msgs, len(logs))
	return msgs, logs
}

func validCoreBridgeLogs(t *testing.T, receipt *types.Receipt, contract eth_common.Address) []*types.Log {
	t.Helper()

	mock := newMockConnector(t)
	var logs []*types.Log
	for _, log := range receipt.Logs {
		if log == nil || !isValidCoreBridgeMessagePublicationLog(*log, contract) {
			continue
		}
		_, err := mock.ParseLogMessagePublished(*log)
		require.NoError(t, err)
		logs = append(logs, log)
	}
	return logs
}

func assertGeneratedReceiptGoldenMessage(
	t *testing.T,
	vec receiptGoldenVector,
	expected receiptExpectedMessage,
	selectedLog *types.Log,
	msg *common.MessagePublication,
) {
	t.Helper()

	require.Equal(t, vec.WormholeChainID, msg.EmitterChain)
	require.Equal(t, int64(vec.BlockTime), msg.Timestamp.Unix())
	require.Equal(t, expected.LogIndex, selectedLog.Index)
	require.GreaterOrEqual(t, len(selectedLog.Topics), 2)
	require.Equal(t, selectedLog.Topics[1].Bytes(), msg.EmitterAddress.Bytes(),
		"emitter must come from topics[1], not the core contract log address")

	expectedHash := strings.TrimPrefix(strings.ToLower(expected.Hash), "0x")
	require.Equal(t, expectedHash, msg.CreateDigest())
	require.Equal(t, expectedHash, msg.VAAHash())
}

func logIndices(logs []*types.Log) []uint {
	out := make([]uint, 0, len(logs))
	for _, log := range logs {
		out = append(out, log.Index)
	}
	return out
}

func cloneReceipt(t *testing.T, receipt *types.Receipt) *types.Receipt {
	t.Helper()

	data, err := json.Marshal(receipt)
	require.NoError(t, err)

	var cloned types.Receipt
	require.NoError(t, json.Unmarshal(data, &cloned))
	return &cloned
}
