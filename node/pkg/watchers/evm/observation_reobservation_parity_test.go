package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
)

// This file cross-checks the two ways the EVM watcher turns an on-chain transaction into
// MessagePublications: the live observation path (runMessageProcessor -> postMessage) and the
// reobservation path (runReobservationHandler -> handleReobservationRequest). For the SAME
// transaction they must yield the same messages, because guardians sign a digest that is identical
// whether a message was seen live or recovered via reobservation.
//
// Test cases are a JSON array of geth types.Receipt, read from testdata (regenerate with
// GEN_TESTDATA=1, see TestGenerateObservationReobservationTestdata). Both paths are driven against
// the existing mockConnector, seeded so its RPC calls (TransactionReceipt, TimeOfBlockByHash,
// ParseLogMessagePublished) return the receipt from the JSON instead of hitting a real node.

var parityTestdataPath = filepath.Join("testdata", "generated_receipts.json")

// blockTimeForReceipt derives a deterministic block time for a receipt. The receipt JSON carries no
// timestamp, so both paths must agree on a synthesized value; keying it off the block number makes it
// deterministic and distinct per block.
func blockTimeForReceipt(receipt *types.Receipt) uint64 {
	var bn uint64
	if receipt.BlockNumber != nil {
		bn = receipt.BlockNumber.Uint64()
	}
	return 1_000_000 + bn
}

// seedReceipt wires the mock so both paths resolve the receipt and its block time from the receipt in
// the JSON rather than an RPC. The block time is registered for the receipt's block hash and for
// every log's block hash, so the live path (which looks up by log.BlockHash) and the reobservation
// path (which looks up by receipt.BlockHash) get the same value.
func seedReceipt(mock *mockConnector, receipt *types.Receipt) {
	bt := blockTimeForReceipt(receipt)
	mock.receipts[receipt.TxHash] = receipt
	mock.blockTimes[receipt.BlockHash] = bt
	for _, l := range receipt.Logs {
		if l != nil {
			mock.blockTimes[l.BlockHash] = bt
		}
	}
}

// coreBridgeEmitterFromReceipt returns the address that emitted the LogMessagePublished events in the
// receipt (the core bridge contract). The watcher must be configured with this address or it rejects
// every log as coming from the wrong contract - so tests using real receipts derive the contract from
// the data rather than assuming the default test emitter. Returns false when the receipt has no
// LogMessagePublished logs (which legitimately yields no messages on either path).
//
// Both paths use the returned value, so setting it does not affect message content (the digest's
// EmitterAddress comes from the log's sender topic, not the core bridge address) - it only decides
// which logs are accepted, identically on each side.
func coreBridgeEmitterFromReceipt(receipt *types.Receipt) (eth_common.Address, bool) {
	for _, l := range receipt.Logs {
		if l != nil && len(l.Topics) > 0 && l.Topics[0] == LogMessagePublishedTopic {
			return l.Address, true
		}
	}
	return eth_common.Address{}, false
}

// parseReceiptEvents parses the LogMessagePublished events from a receipt's logs, applying the same
// validation the watcher does. The returned events are exactly the ones the live path will turn into
// messages - the same set MessageEventsForTransaction derives internally on the reobservation side.
func parseReceiptEvents(t *testing.T, mock *mockConnector, receipt *types.Receipt, contract eth_common.Address) []*ethabi.AbiLogMessagePublished {
	t.Helper()
	var events []*ethabi.AbiLogMessagePublished
	for _, l := range receipt.Logs {
		if l == nil || !isValidCoreBridgeMessagePublicationLog(*l, contract) {
			continue
		}
		ev, err := mock.ParseLogMessagePublished(*l)
		require.NoError(t, err)
		events = append(events, ev)
	}
	return events
}

// drainMsgC non-blockingly reads all messages currently buffered on msgC.
func drainMsgC(msgC <-chan *common.MessagePublication) []*common.MessagePublication {
	var out []*common.MessagePublication
	for {
		select {
		case m := <-msgC:
			out = append(out, m)
		default:
			return out
		}
	}
}

// digestMultiset maps each message's VAA signing digest to how many times it appears. Comparing two
// multisets is order-independent, which matters because the two paths may emit messages in different
// orders (map iteration on the live pending set vs. receipt-log order on reobservation).
func digestMultiset(msgs []*common.MessagePublication) map[string]int {
	m := make(map[string]int, len(msgs))
	for _, msg := range msgs {
		m[msg.VAAHash()]++
	}
	return m
}

// runLiveObservation drives runMessageProcessor over the events parsed from the receipt and returns
// every MessagePublication it produces: those published immediately (msgC) plus those queued awaiting
// confirmation (pending map). Queued message objects are byte-for-byte what processNewBlock would
// later publish, so their signing digests are final.
func runLiveObservation(t *testing.T, receipt *types.Receipt) []*common.MessagePublication {
	t.Helper()
	w, mock, _ := newTestWatcher(t)
	msgC := make(chan *common.MessagePublication, 4096)
	w.msgC = msgC
	if emitter, ok := coreBridgeEmitterFromReceipt(receipt); ok {
		w.contract = emitter // match the receipt's core bridge so its logs are accepted
	}
	seedReceipt(mock, receipt)

	events := parseReceiptEvents(t, mock, receipt, w.contract)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	feed := make(chan *ethabi.AbiLogMessagePublished, len(events)+1)
	errC := make(chan error, 1)
	done := make(chan error, 1)
	go func() { done <- w.runMessageProcessor(ctx, errC, newFakeSubscription(), feed) }()

	for _, ev := range events {
		feed <- ev
	}

	// Deterministic wait: keep draining published messages until published + still-pending equals
	// the number of events we fed. No idle timeout, so this is not flaky.
	var published []*common.MessagePublication
	require.Eventually(t, func() bool {
		published = append(published, drainMsgC(msgC)...)
		w.pendingMu.Lock()
		defer w.pendingMu.Unlock()
		return len(published)+len(w.pending) == len(events)
	}, 2*time.Second, 5*time.Millisecond, "live path produced %d/%d messages", len(published), len(events))

	select {
	case err := <-errC:
		t.Fatalf("runMessageProcessor reported an error: %v", err)
	default:
	}

	out := published
	w.pendingMu.Lock()
	for _, pe := range w.pending {
		out = append(out, pe.message)
	}
	w.pendingMu.Unlock()
	return out
}

// runReobservation drives runReobservationHandler for the receipt's transaction and returns the
// published MessagePublications. Both chain heads are set above the receipt's block so messages of
// every consistency level publish (rather than being dropped as "too early"). expectedN is used to
// read a precise number of messages so a count mismatch fails loudly via a recvMsg timeout.
func runReobservation(t *testing.T, receipt *types.Receipt, expectedN int) []*common.MessagePublication {
	t.Helper()
	w, mock, _ := newTestWatcher(t)
	msgC := make(chan *common.MessagePublication, 4096)
	w.msgC = msgC
	if emitter, ok := coreBridgeEmitterFromReceipt(receipt); ok {
		w.contract = emitter // match the receipt's core bridge so its logs are accepted
	}
	seedReceipt(mock, receipt)

	atomic.StoreUint64(&w.latestFinalizedBlockNumber, ^uint64(0))
	atomic.StoreUint64(&w.latestSafeBlockNumber, ^uint64(0))

	reqC := make(chan *gossipv1.ObservationRequest, 1)
	w.obsvReqC = reqC

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = w.runReobservationHandler(ctx) }()

	reqC <- &gossipv1.ObservationRequest{
		ChainId: uint32(w.chainID),
		TxHash:  receipt.TxHash.Bytes(),
	}

	msgs := make([]*common.MessagePublication, 0, expectedN)
	for range expectedN {
		msgs = append(msgs, recvMsg(t, msgC))
	}

	// Nothing more should be published - if it is, the paths disagree on message count.
	select {
	case extra := <-msgC:
		t.Fatalf("reobservation produced more than the expected %d messages (extra msgId=%s)", expectedN, extra.MessageIDString())
	case <-time.After(100 * time.Millisecond):
	}
	return msgs
}

// TestObservationReobservationParity is the core harness: for every receipt (test case) in the JSON
// file it runs both entrypoints and asserts they produce the same messages. Equality is by VAA
// signing digest - the exact bytes guardians sign - so a divergence here means the two paths would
// sign different VAAs for the same transaction.
func TestObservationReobservationParity(t *testing.T) {
	receipts := loadReceiptTestcases(t)
	require.NotEmpty(t, receipts, "no test cases in %s", parityTestdataPath)

	// Guard against a vacuous pass: if every case produced zero messages, the comparison would
	// trivially hold. Require that the suite actually exercised message production overall. Atomic
	// because the cases run in parallel.
	var totalMessages atomic.Int64

	// Cases are independent (each builds its own watcher/mock), so run them in parallel. The group
	// subtest blocks until all parallel children finish, so the guard below sees the final total.
	t.Run("cases", func(t *testing.T) {
		for i, receipt := range receipts {
			t.Run(fmt.Sprintf("case_%d_tx_%s", i, receipt.TxHash.Hex()), func(t *testing.T) {
				t.Parallel()
				// Non-success receipts are handled asymmetrically by design (reobservation rejects the
				// whole tx up front; the live path defers the status check to processNewBlock), so they
				// are out of scope for a direct output comparison.
				require.Equal(t, types.ReceiptStatusSuccessful, receipt.Status,
					"test cases must be successful receipts; got status %d for tx %s", receipt.Status, receipt.TxHash.Hex())

				live := runLiveObservation(t, receipt)
				reobs := runReobservation(t, receipt, len(live))

				// The paths must differ only in IsReobservation, proving we really exercised both.
				for _, m := range live {
					require.False(t, m.IsReobservation, "live observation flagged a message as reobservation (msgId=%s)", m.MessageIDString())
				}
				for _, m := range reobs {
					require.True(t, m.IsReobservation, "reobservation left a message unflagged (msgId=%s)", m.MessageIDString())
				}

				// The invariant: identical signing digests, ignoring IsReobservation (excluded from the
				// VAA). Fail loudly with both sides' message IDs on mismatch.
				require.Equal(t, digestMultiset(live), digestMultiset(reobs),
					"observation vs reobservation produced different messages for tx %s\n live : %v\n reobs: %v",
					receipt.TxHash.Hex(), messageIDs(live), messageIDs(reobs))

				totalMessages.Add(int64(len(live)))
			})
		}
	})

	require.Positive(t, totalMessages.Load(), "no messages were produced across any test case; the harness compared nothing")
}

func messageIDs(msgs []*common.MessagePublication) []string {
	ids := make([]string, len(msgs))
	for i, m := range msgs {
		ids[i] = m.MessageIDString()
	}
	return ids
}

func loadReceiptTestcases(t *testing.T) []*types.Receipt {
	t.Helper()
	data, err := os.ReadFile(parityTestdataPath)
	require.NoError(t, err, "read %s (regenerate with GEN_TESTDATA=1)", parityTestdataPath)
	var receipts []*types.Receipt
	require.NoError(t, json.Unmarshal(data, &receipts), "unmarshal %s", parityTestdataPath)
	return receipts
}
