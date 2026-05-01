package evm

import (
	"context"
	"errors"
	"testing"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/txverifier"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

func TestReobserveSingleMessage(t *testing.T) {
	tests := []struct {
		name             string
		consistencyLevel uint8
	}{
		{"finalized", vaa.ConsistencyLevelFinalized},
		{"safe", vaa.ConsistencyLevelSafe},
		{"instant", vaa.ConsistencyLevelPublishImmediately},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, mock, msgC := newTestWatcher(t)

			log := newValidWormholeLog(t, testBlockNumber, tc.consistencyLevel)
			mock.seedLog(log, 1234)

			numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, log.TxHash.Bytes(), mock, testFinalizedBlockNum, testSafeBlockNum)
			require.NoError(t, err)
			require.Equal(t, uint32(1), numObs)
			require.Equal(t, 1, len(msgC))

			ev, err := mock.ParseLogMessagePublished(*log)
			require.NoError(t, err)
			msg := recvMsg(t, msgC)
			require.True(t, msg.IsReobservation)
			require.Equal(t, common.NotVerified, msg.VerificationState())
			assertMessageMatchesEvent(t, msg, ev, 1234)
		})
	}
}

// TestReobserveSingleMessageEarly covers cases where the log's block number is ahead of the
// current chain head for the message's consistency level, so the reobservation is dropped.
func TestReobserveSingleMessageEarly(t *testing.T) {
	tests := []struct {
		name              string
		consistencyLevel  uint8
		finalizedBlockNum uint64
		safeBlockNum      uint64
	}{
		{"finalized", vaa.ConsistencyLevelFinalized, testBlockNumber - 1, testSafeBlockNum},
		{"safe", vaa.ConsistencyLevelSafe, testFinalizedBlockNum, testBlockNumber - 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, mock, msgC := newTestWatcher(t)

			log := newValidWormholeLog(t, testBlockNumber, tc.consistencyLevel)
			mock.seedLog(log, 1234)

			numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, log.TxHash.Bytes(), mock, tc.finalizedBlockNum, tc.safeBlockNum)
			require.NoError(t, err)
			require.Equal(t, uint32(0), numObs)
			require.Equal(t, 0, len(msgC))
		})
	}
}

func TestReobserveInvalidChainId(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	numObs, err := w.handleReobservationRequest(context.TODO(), vaa.ChainIDSolana, []byte{}, mock, testFinalizedBlockNum, testSafeBlockNum)
	require.Error(t, err)
	require.Equal(t, uint32(0), numObs)
	require.Equal(t, 0, len(msgC))
}

func TestReobserveReceiptError(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	txHash := eth_common.HexToHash("0x1234")
	mock.errors[txHash] = errors.New("rpc failure")

	numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, txHash.Bytes(), mock, testFinalizedBlockNum, testSafeBlockNum)
	require.Error(t, err)
	require.Equal(t, uint32(0), numObs)
	require.Equal(t, 0, len(msgC))
}

func TestReobserveFailedTransactionStatus(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	log := newValidWormholeLog(t, testBlockNumber, vaa.ConsistencyLevelFinalized)
	receipt := newTestReceipt(log.BlockNumber, []*types.Log{log})
	receipt.Status = 0
	mock.receipts[log.TxHash] = receipt

	numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, log.TxHash.Bytes(), mock, testFinalizedBlockNum, testSafeBlockNum)
	require.Error(t, err)
	require.Equal(t, uint32(0), numObs)
	require.Equal(t, 0, len(msgC))
}

// TestReobserveLogSkipped covers receipt logs that should not produce a MessagePublication.
// Each subtest starts with a fully valid log and mutates a single field so the log is rejected
// by by_transaction.go. All cases should return numObs=0 with no error.
func TestReobserveLogSkipped(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*types.Log)
	}{
		{
			name: "wrong_contract_address",
			mutate: func(l *types.Log) {
				l.Address = eth_common.HexToAddress("0x396343362be2A4dA1cE0C1C210945346fb82Aa49")
			},
		},
		{
			name: "wrong_event_topic",
			mutate: func(l *types.Log) {
				l.Topics[0] = eth_common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
			},
		},
		{
			name: "removed_flag_set",
			mutate: func(l *types.Log) {
				l.Removed = true
			},
		},
		{
			name: "empty_topics",
			mutate: func(l *types.Log) {
				l.Topics = []eth_common.Hash{}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, mock, msgC := newTestWatcher(t)

			log := newValidWormholeLog(t, testBlockNumber, vaa.ConsistencyLevelFinalized)
			tc.mutate(log)
			mock.seedLog(log, 1234)

			numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, log.TxHash.Bytes(), mock, testFinalizedBlockNum, testSafeBlockNum)
			require.NoError(t, err)
			require.Equal(t, uint32(0), numObs)
			require.Equal(t, 0, len(msgC))
		})
	}
}

// TestReobserveDeterministicOrdering verifies processing the same message
// always leads to the same result. This test is meant to catch non-determinism issues
// that get added to this code path.
func TestReobserveDeterministicOrdering(t *testing.T) {
	const numEvents = 20
	const numRuns = 5

	var firstRun []uint64
	for run := 0; run < numRuns; run++ {
		w, mock, _ := newTestWatcher(t)
		msgC := make(chan *common.MessagePublication, numEvents)
		w.msgC = msgC

		logs := make([]*types.Log, numEvents)
		for i := 0; i < numEvents; i++ {
			log := newTestLog(t, testLogParams{
				sender:           testTokenBridge,
				contractAddr:     testEmitter,
				sequence:         uint64(i),
				consistencyLevel: vaa.ConsistencyLevelFinalized,
				blockNumber:      testBlockNumber,
			})
			logs[i] = &log
		}

		receipt := newTestReceipt(testBlockNumber, logs)
		mock.receipts[logs[0].TxHash] = receipt
		mock.blockTimes[receipt.BlockHash] = 1234

		numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, logs[0].TxHash.Bytes(), mock, testFinalizedBlockNum, testSafeBlockNum)
		require.NoError(t, err)
		require.Equal(t, uint32(numEvents), numObs)

		seqs := make([]uint64, numEvents)
		for i := 0; i < numEvents; i++ {
			seqs[i] = recvMsg(t, msgC).Sequence
		}

		if run == 0 {
			firstRun = seqs
			continue
		}
		require.Equal(t, firstRun, seqs, "run %d produced a different ordering than run 0", run)
	}
}

// TestReobserve1KEventsInReceipt verifies that a receipt containing 1000 LogMessagePublished
// events produces 1000 published MessagePublications with sequences preserved in order.
func TestReobserve1KEventsInReceipt(t *testing.T) {
	const numEvents = 1000

	w, mock, _ := newTestWatcher(t)
	msgC := make(chan *common.MessagePublication, numEvents)
	w.msgC = msgC

	logs := make([]*types.Log, numEvents)
	for i := 0; i < numEvents; i++ {
		log := newTestLog(t, testLogParams{
			sender:           testTokenBridge,
			contractAddr:     testEmitter,
			sequence:         uint64(i),
			consistencyLevel: vaa.ConsistencyLevelFinalized,
			blockNumber:      testBlockNumber,
		})
		logs[i] = &log
	}

	receipt := newTestReceipt(testBlockNumber, logs)
	mock.receipts[logs[0].TxHash] = receipt
	mock.blockTimes[receipt.BlockHash] = 1234

	numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, logs[0].TxHash.Bytes(), mock, testFinalizedBlockNum, testSafeBlockNum)
	require.NoError(t, err)
	require.Equal(t, uint32(numEvents), numObs)
	require.Equal(t, numEvents, len(msgC))

	for i := 0; i < numEvents; i++ {
		msg := recvMsg(t, msgC)
		require.True(t, msg.IsReobservation)
		require.Equal(t, uint64(i), msg.Sequence)
	}
}

func TestReobserveTwoValidEvents(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	log1 := newValidWormholeLog(t, testBlockNumber, vaa.ConsistencyLevelFinalized)
	mock.seedLog(log1, 1234)

	log2 := newTestLog(t, testLogParams{
		sender:           testTokenBridge,
		contractAddr:     testEmitter,
		sequence:         2,
		consistencyLevel: vaa.ConsistencyLevelFinalized,
		blockNumber:      testBlockNumber,
	})

	receipt := newTestReceipt(log1.BlockNumber, []*types.Log{log1, &log2})
	mock.receipts[log1.TxHash] = receipt
	mock.blockTimes[receipt.BlockHash] = 1234

	numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, log1.TxHash.Bytes(), mock, testFinalizedBlockNum, testSafeBlockNum)
	require.NoError(t, err)
	require.Equal(t, uint32(2), numObs)
	require.Equal(t, 2, len(msgC))

	ev, err := mock.ParseLogMessagePublished(*log1)
	require.NoError(t, err)
	msg := recvMsg(t, msgC)
	require.True(t, msg.IsReobservation)
	assertMessageMatchesEvent(t, msg, ev, 1234)

	ev, err = mock.ParseLogMessagePublished(log2)
	require.NoError(t, err)
	msg = recvMsg(t, msgC)
	require.True(t, msg.IsReobservation)
	assertMessageMatchesEvent(t, msg, ev, 1234)
}

func TestReobserveValidAndInvalid(t *testing.T) {
	w, mock, msgC := newTestWatcher(t)

	log1 := newValidWormholeLog(t, testBlockNumber, vaa.ConsistencyLevelFinalized)
	mock.seedLog(log1, 1234)

	log2 := newTestLog(t, testLogParams{
		sender:           testTokenBridge,
		contractAddr:     testEmitter,
		sequence:         2,
		consistencyLevel: vaa.ConsistencyLevelFinalized,
		blockNumber:      testBlockNumber,
	})

	log2.Topics[0] = eth_common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")

	receipt := newTestReceipt(log1.BlockNumber, []*types.Log{log1, &log2})
	mock.receipts[log1.TxHash] = receipt
	mock.blockTimes[receipt.BlockHash] = 1234

	numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, log1.TxHash.Bytes(), mock, testFinalizedBlockNum, testSafeBlockNum)
	require.NoError(t, err)
	require.Equal(t, uint32(1), numObs)
	require.Equal(t, 1, len(msgC))

	ev, err := mock.ParseLogMessagePublished(*log1)
	require.NoError(t, err)
	msg := recvMsg(t, msgC)
	require.True(t, msg.IsReobservation)
	assertMessageMatchesEvent(t, msg, ev, 1234)
}

func TestReobserveTxVerifierIntegration(t *testing.T) {
	otherSender := eth_common.HexToAddress("0x000000000000000000000000000000000000dEaD")

	tests := []struct {
		name        string
		verifier    txverifier.TransferVerifierInterface
		sender      eth_common.Address
		expectState common.VerificationState
	}{
		{"success", &MockTransferVerifier[ethclient.Client, connectors.Connector]{success: true}, testTokenBridge, common.Valid},
		{"failure", &MockTransferVerifier[ethclient.Client, connectors.Connector]{success: false}, testTokenBridge, common.Rejected},
		{"notapplicable", &MockTransferVerifier[ethclient.Client, connectors.Connector]{success: false}, otherSender, common.NotApplicable},
		{"notverified", nil, testTokenBridge, common.NotVerified},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			w, mock, msgC := newTestWatcher(t)
			w.txVerifier = tc.verifier

			log := newTestLog(t, testLogParams{
				sender:           tc.sender,
				contractAddr:     testEmitter,
				sequence:         1,
				consistencyLevel: vaa.ConsistencyLevelFinalized,
				blockNumber:      testBlockNumber,
			})
			mock.seedLog(&log, 1234)

			numObs, err := w.handleReobservationRequest(context.TODO(), w.chainID, log.TxHash.Bytes(), mock, testFinalizedBlockNum, testSafeBlockNum)
			require.NoError(t, err)
			require.Equal(t, uint32(1), numObs)
			require.Equal(t, 1, len(msgC))

			ev, err := mock.ParseLogMessagePublished(log)
			require.NoError(t, err)
			msg := recvMsg(t, msgC)
			require.True(t, msg.IsReobservation)
			require.Equal(t, tc.expectState, msg.VerificationState())
			assertMessageMatchesEvent(t, msg, ev, 1234)
		})
	}
}
