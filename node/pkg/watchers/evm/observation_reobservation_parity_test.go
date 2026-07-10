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
// Test cases come from the JSON files in parityTestdataFiles - each a flattened geth types.Receipt
// plus messageSent/error fields - captured against a specific core bridge contract. Both paths are
// driven against the existing mockConnector, seeded so its RPC calls (TransactionReceipt,
// TimeOfBlockByHash, ParseLogMessagePublished) return the receipt from the JSON instead of hitting a
// real node.

// parityTestdataFiles lists the JSON test-case files and the core bridge contract each was captured
// against. The watcher is configured with a single contract per chain, so every case - including
// failures whose logs come from the wrong address - is validated against its file's contract
// (deriving it per-receipt would wrongly accept a log from any address). Contracts are pinned here
// rather than read from the mutable chain config so they stay matched to the data.
var parityTestdataFiles = []struct {
	path     string
	contract eth_common.Address
}{
	// Sepolia core bridge (testnetChainConfig[vaa.ChainIDEthereum].ContractAddr).
	{filepath.Join("testdata", "generated_receipts.json"), eth_common.HexToAddress("0x4a8bc80ed5a4067f1ccf107057b8270e0cc11a78")},
	// Ethereum mainnet core bridge.
	{filepath.Join("testdata", "real_receipts.json"), eth_common.HexToAddress("0x98f3c9e6e3face36baad05fe09d375ef1464288b")},
}

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
func runLiveObservation(t *testing.T, receipt *types.Receipt, contract eth_common.Address) []*common.MessagePublication {
	t.Helper()
	w, mock, _ := newTestWatcher(t)
	msgC := make(chan *common.MessagePublication, 4096)
	w.msgC = msgC
	w.contract = contract // the watcher's single configured core bridge; invalid logs are rejected against it
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
func runReobservation(t *testing.T, receipt *types.Receipt, contract eth_common.Address, expectedN int) []*common.MessagePublication {
	t.Helper()
	w, mock, _ := newTestWatcher(t)
	msgC := make(chan *common.MessagePublication, 4096)
	w.msgC = msgC
	w.contract = contract // the watcher's single configured core bridge; invalid logs are rejected against it
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

// TestObservationReobservationParity is the core harness: for every test case in the JSON file it
// runs both entrypoints (live observation and reobservation) and asserts they agree.
//
// For a case expected to succeed (messageSent=true) the two paths must produce the same messages by
// VAA signing digest - the exact bytes guardians sign - so a divergence means they would sign
// different VAAs for the same transaction. For a case expected to fail (messageSent=false) neither
// path may publish a message; we only assert nothing is emitted (the specific rejection reason is
// not checked).
func TestObservationReobservationParity(t *testing.T) {
	cases := loadTestCases(t)
	require.NotEmpty(t, cases, "no test cases loaded")

	// Guard against a vacuous pass: if no case produced messages, the comparison would trivially hold.
	// Require that the suite actually exercised message production overall. Atomic because the cases
	// run in parallel.
	var totalMessages atomic.Int64

	// Cases are independent (each builds its own watcher/mock), so run them in parallel. The group
	// subtest blocks until all parallel children finish, so the guard below sees the final total.
	t.Run("cases", func(t *testing.T) {
		for i, tc := range cases {
			receipt := tc.Receipt
			t.Run(fmt.Sprintf("%s_case_%d_tx_%s", tc.Source, i, receipt.TxHash.Hex()), func(t *testing.T) {
				t.Parallel()

				live := runLiveObservation(t, receipt, tc.Contract)
				reobs := runReobservation(t, receipt, tc.Contract, len(live))

				if !tc.MessageSent {
					// Expected failure: a bad log (reorg-removed, wrong emitter, or wrong topic) must
					// not be published by either path.
					require.Empty(t, live, "messageSent=false but live path published/queued messages for tx %s: %v", receipt.TxHash.Hex(), messageIDs(live))
					require.Empty(t, reobs, "messageSent=false but reobservation published messages for tx %s: %v", receipt.TxHash.Hex(), messageIDs(reobs))
					return
				}

				require.NotEmpty(t, live, "messageSent=true but live path produced no messages for tx %s", receipt.TxHash.Hex())

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

	require.Positive(t, totalMessages.Load(), "no successful messages were produced across any test case; the harness compared nothing")
}

func messageIDs(msgs []*common.MessagePublication) []string {
	ids := make([]string, len(msgs))
	for i, m := range msgs {
		ids[i] = m.MessageIDString()
	}
	return ids
}

// parityTestCase is one entry in a JSON test file: a geth receipt with two extra flattened fields.
// messageSent records whether the transaction is expected to produce a MessagePublication. error
// documents the reason a failure case is rejected; its exact text is not asserted (we only require
// that a failing message is not published). Contract and Source are populated at load time (not from
// JSON) from the file the case came from.
type parityTestCase struct {
	Receipt     *types.Receipt
	MessageSent bool
	Error       string

	Contract eth_common.Address // core bridge the receipt was captured against
	Source   string             // testdata file basename, for identifying the case
}

// UnmarshalJSON reads the flattened form: the messageSent/error keys sit alongside the receipt's own
// fields in the same object. The receipt's UnmarshalJSON ignores the two extra keys, and the metadata
// struct ignores the receipt's keys, so each side decodes only what it owns.
func (tc *parityTestCase) UnmarshalJSON(data []byte) error {
	var meta struct {
		MessageSent bool   `json:"messageSent"`
		Error       string `json:"error"`
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return err
	}
	tc.MessageSent = meta.MessageSent
	tc.Error = meta.Error
	tc.Receipt = new(types.Receipt)
	return tc.Receipt.UnmarshalJSON(data)
}

func loadTestCases(t *testing.T) []*parityTestCase {
	t.Helper()
	var all []*parityTestCase
	for _, f := range parityTestdataFiles {
		data, err := os.ReadFile(f.path)
		require.NoError(t, err, "read %s", f.path)
		var cases []*parityTestCase
		require.NoError(t, json.Unmarshal(data, &cases), "unmarshal %s", f.path)
		source := filepath.Base(f.path)
		for _, tc := range cases {
			tc.Contract = f.contract
			tc.Source = source
		}
		all = append(all, cases...)
	}
	return all
}
