package evm

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	gossipv1 "github.com/certusone/wormhole/node/pkg/proto/gossip/v1"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Each fixture is exercised through both the live observation and reobservation paths. The core
// bridge addresses are pinned to the networks from which the fixtures were captured.
var (
	generatedReceiptContract = eth_common.HexToAddress("0x4a8bc80ed5a4067f1ccf107057b8270e0cc11a78")
	realReceiptContract      = eth_common.HexToAddress("0x98f3c9e6e3face36baad05fe09d375ef1464288b")
)

var parityTestdataFiles = []struct {
	path     string
	contract eth_common.Address
}{
	{
		path:     filepath.Join("testdata", "generated_receipts.json"),
		contract: generatedReceiptContract,
	},
	{
		path:     filepath.Join("testdata", "real_receipts.json"),
		contract: realReceiptContract,
	},
}

// Both paths request time by block hash, but live observation uses the log hash while
// reobservation uses the receipt hash. Fixtures can contain either representation.
func seedReceipt(mock *mockConnector, receipt *types.Receipt, blockTime uint64) {
	mock.receipts[receipt.TxHash] = receipt
	mock.blockTimes[receipt.BlockHash] = blockTime
	for _, l := range receipt.Logs {
		if l != nil {
			mock.blockTimes[l.BlockHash] = blockTime
		}
	}
}

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

// digestMultiset maps each message's CreateDigest hash to how many times it appears. Comparing two
// multisets is order-independent, which matters because the two paths may emit messages in different
// orders (map iteration on the live pending set vs. receipt-log order on reobservation).
func digestMultiset(t *testing.T, msgs []*common.MessagePublication) map[string]int {
	t.Helper()

	m := make(map[string]int, len(msgs))
	for _, msg := range msgs {
		digest := msg.CreateDigest()
		require.Equal(t, digest, msg.VAAHash(),
			"CreateDigest and VAAHash must stay equivalent for msgId=%s", msg.MessageIDString())
		m[digest]++
	}
	return m
}

func expectedDigestMultiset(expected []receiptExpectedMessage) map[string]int {
	m := make(map[string]int, len(expected))
	for _, item := range expected {
		m[normalizeDigest(item.Hash)]++
	}
	return m
}

func normalizeDigest(hash string) string {
	return strings.TrimPrefix(strings.ToLower(hash), "0x")
}

// runLiveObservation drives runMessageProcessor over the events parsed from the receipt and returns
// every MessagePublication it produces: those published immediately (msgC) plus those queued awaiting
// confirmation (pending map). Queued message objects are byte-for-byte what processNewBlock would
// later publish, so their signing digests are final.
func runLiveObservation(t *testing.T, tc *parityTestCase) []*common.MessagePublication {
	t.Helper()
	w, mock, _ := newTestWatcher(t)
	msgC := make(chan *common.MessagePublication, 4096)
	w.msgC = msgC
	w.contract = tc.Contract // the watcher's single configured core bridge; invalid logs are rejected against it
	w.chainID = tc.WormholeChainID
	seedReceipt(mock, tc.Receipt, tc.BlockTime)

	events := parseReceiptEvents(t, mock, tc.Receipt, w.contract)

	ctx, cancel := context.WithCancel(context.Background())
	feed := make(chan *ethabi.AbiLogMessagePublished, len(events)+1)
	errC := make(chan error, 1)
	done := make(chan error, 1)
	go func() { done <- w.runMessageProcessor(ctx, errC, newFakeSubscription(), feed) }()
	defer func() {
		cancel()
		require.NoError(t, <-done)
	}()

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
func runReobservation(t *testing.T, tc *parityTestCase, expectedN int) []*common.MessagePublication {
	t.Helper()
	w, mock, _ := newTestWatcher(t)
	msgC := make(chan *common.MessagePublication, 4096)
	w.msgC = msgC
	w.contract = tc.Contract // the watcher's single configured core bridge; invalid logs are rejected against it
	w.chainID = tc.WormholeChainID
	seedReceipt(mock, tc.Receipt, tc.BlockTime)

	atomic.StoreUint64(&w.latestFinalizedBlockNumber, ^uint64(0))
	atomic.StoreUint64(&w.latestSafeBlockNumber, ^uint64(0))

	reqC := make(chan *gossipv1.ObservationRequest, 1)
	w.obsvReqC = reqC

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- w.runReobservationHandler(ctx) }()
	defer func() {
		cancel()
		require.NoError(t, <-done)
	}()

	reqC <- &gossipv1.ObservationRequest{
		ChainId: uint32(w.chainID),
		TxHash:  tc.Receipt.TxHash.Bytes(),
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

// TestObservationReobservationParity pins the digest produced by both watcher paths to hashes
// generated independently of MessagePublication.
func TestObservationReobservationParity(t *testing.T) {
	cases := loadTestCases(t)
	require.NotEmpty(t, cases, "no test cases loaded")

	for i, tc := range cases {
		receipt := tc.Receipt
		t.Run(parityTestName(tc, i), func(t *testing.T) {
			t.Parallel()

			live := runLiveObservation(t, tc)
			reobs := runReobservation(t, tc, len(live))
			require.NotEmpty(t, live, "live path produced no messages for tx %s", receipt.TxHash.Hex())

			for _, m := range live {
				require.False(t, m.IsReobservation, "live observation flagged a message as reobservation (msgId=%s)", m.MessageIDString())
			}
			for _, m := range reobs {
				require.True(t, m.IsReobservation, "reobservation left a message unflagged (msgId=%s)", m.MessageIDString())
			}

			liveDigests := digestMultiset(t, live)
			reobservationDigests := digestMultiset(t, reobs)
			require.Equal(t, liveDigests, reobservationDigests,
				"observation vs reobservation produced different messages for tx %s\n live : %v\n reobs: %v",
				receipt.TxHash.Hex(), messageIDs(live), messageIDs(reobs))

			require.Equal(t, expectedDigestMultiset(tc.Expected), liveDigests,
				"live observation hash does not match fixture for tx %s", receipt.TxHash.Hex())
		})
	}
}

func parityTestName(tc *parityTestCase, index int) string {
	name := fmt.Sprintf("%s_%03d", strings.TrimSuffix(tc.Source, filepath.Ext(tc.Source)), index)
	if tc.Name != "" {
		name += "_" + tc.Name
	}
	return name
}

func messageIDs(msgs []*common.MessagePublication) []string {
	ids := make([]string, len(msgs))
	for i, m := range msgs {
		ids[i] = m.MessageIDString()
	}
	return ids
}

type parityTestCase struct {
	*receiptGoldenVector
	Contract eth_common.Address
	Source   string
}

// receiptGoldenVector is the canonical fixture format shared by the parity and generated-vector tests.
type receiptGoldenVector struct {
	Name            string                   `json:"name"`
	Comment         string                   `json:"comment"`
	WormholeChainID vaa.ChainID              `json:"wormholeChainId"`
	BlockTime       uint64                   `json:"blockTime"`
	Receipt         *types.Receipt           `json:"receipt"`
	Expected        []receiptExpectedMessage `json:"expectedMessages"`
}

type receiptExpectedMessage struct {
	LogIndex uint   `json:"logIndex"`
	Hash     string `json:"hash"`
}

func loadTestCases(t *testing.T) []*parityTestCase {
	t.Helper()
	var all []*parityTestCase
	for _, f := range parityTestdataFiles {
		data, err := os.ReadFile(f.path)
		require.NoError(t, err, "read %s", f.path)
		var vectors []*receiptGoldenVector
		require.NoError(t, json.Unmarshal(data, &vectors), "unmarshal %s", f.path)
		source := filepath.Base(f.path)
		for i, vector := range vectors {
			require.NotNil(t, vector, "%s case %d", source, i)
			tc := &parityTestCase{
				receiptGoldenVector: vector,
				Contract:            f.contract,
				Source:              source,
			}
			requireGoldenFixture(t, tc, i)
			all = append(all, tc)
		}
	}
	return all
}

func requireGoldenFixture(t *testing.T, tc *parityTestCase, index int) {
	t.Helper()
	require.NotEmpty(t, tc.Name, "%s case %d name", tc.Source, index)
	require.NotZero(t, tc.BlockTime, "%s case %d block time", tc.Source, index)
	require.NotZero(t, tc.WormholeChainID, "%s case %d chain ID", tc.Source, index)
	require.NotNil(t, tc.Receipt, "%s case %d receipt", tc.Source, index)
	require.NotNil(t, tc.Receipt.BlockNumber, "%s case %d block number", tc.Source, index)
	require.NotEmpty(t, tc.Expected, "%s case %d expected messages", tc.Source, index)

	actualLogIndices := make([]uint, 0, len(tc.Expected))
	for _, log := range tc.Receipt.Logs {
		if log != nil && isValidCoreBridgeMessagePublicationLog(*log, tc.Contract) {
			actualLogIndices = append(actualLogIndices, log.Index)
		}
	}
	require.Equal(t, expectedMessageLogIndices(tc.Expected), actualLogIndices,
		"%s case %d selected logs", tc.Source, index)

	for messageIndex, expected := range tc.Expected {
		digest, err := hex.DecodeString(normalizeDigest(expected.Hash))
		require.NoError(t, err, "%s case %d message %d hash", tc.Source, index, messageIndex)
		require.Len(t, digest, 32, "%s case %d message %d hash", tc.Source, index, messageIndex)
	}
}

func expectedMessageLogIndices(expected []receiptExpectedMessage) []uint {
	indices := make([]uint, 0, len(expected))
	for _, message := range expected {
		indices = append(indices, message.LogIndex)
	}
	return indices
}
