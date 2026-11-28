package solana

// This file loads the Solana watcher replay cases and provides their mock RPC client.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// The replay cases are split across two minified JSON files generated as follows:
//
//   - staticBundlesFile: the builder-generated (synthetic) matrix. Regenerate with:
//     go run ./pkg/watchers/solana/testgen/cmd static
//   - liveBundlesFile: live-collected Solana transactions. Regenerate with:
//     go run ./pkg/watchers/solana/testgen/cmd live --rpc "$SOLANA_RPC_URL"

const staticBundlesFile = "testdata/static_bundles.json"
const liveBundlesFile = "testdata/live_bundles.json"

// bundleFiles is every fixture file the replay test loads, in read order.
var bundleFiles = []string{staticBundlesFile, liveBundlesFile}

const updateReplayFixturesEnv = "UPDATE_SOLANA_REPLAY_FIXTURES"

// replayBundle mirrors the JSON emitted by the testgen builder. transaction and meta
// decode straight into the watcher's own input types.
type replayBundle struct {
	Name         string              `json:"name"`
	Slot         uint64              `json:"slot"`
	Contract     solana.PublicKey    `json:"contract"`
	ShimContract solana.PublicKey    `json:"shimContract"`
	Transaction  solana.Transaction  `json:"transaction"`
	Meta         rpc.TransactionMeta `json:"meta"`
	Accounts     []replayAccount     `json:"accounts"`
	Expected     *expectedOutput     `json:"expected,omitempty"`
}

type replayAccount struct {
	Pubkey solana.PublicKey `json:"pubkey"`
	Owner  solana.PublicKey `json:"owner"`
	Data   []string         `json:"data"` // [base64Data, "base64"]
}

// expectedOutput records the VAA signing digests emitted by each replay flow:
//   - Reobservation: handleReobservationRequest(txID).
//   - Observation: fetchBlock, the normal guardian block-observation path.
//   - Polling: processNewTransactions, the getSignaturesForAddress -> getTransaction path.
//   - Account: handleReobservationRequest(accountID) over every served account.
type expectedOutput struct {
	Reobservation []string `json:"reobservation"`
	Observation   []string `json:"observation"`
	Polling       []string `json:"polling"`
	Account       []string `json:"account"`
}

type expectedCount struct {
	count             int
	collectUntilQuiet bool
}

type replayExpectedCounts struct {
	reobservation expectedCount
	observation   expectedCount
	polling       expectedCount
	account       expectedCount
}

// newReplayWatcher builds the minimum watcher state needed to exercise the real Solana
// observation and reobservation entrypoints.
func newReplayWatcher(t *testing.T, b *replayBundle, msgC chan<- *common.MessagePublication) *SolanaWatcher {
	t.Helper()

	s := newTestWatcher(t, vaa.ChainIDSolana, rpc.CommitmentFinalized, msgC)
	s.errC = make(chan error, 64)
	s.ctx = context.Background()
	s.contract = b.Contract
	s.whLogPrefix = fmt.Sprintf("Program %s", b.Contract) // Necessary for the observation flow
	s.shimContractAddr = b.ShimContract
	s.shimContractStr = b.ShimContract.String()
	s.shimSetup()
	return s
}

// seedReplayRPCClient converts a generated bundle into the RPC responses consumed by
// the high-level watcher paths: getAccountInfo, getBlock, and getTransaction.
func seedReplayRPCClient(t *testing.T, b *replayBundle) *mockSolanaRPCClient {
	t.Helper()

	m := newMockSolanaRPCClient()
	for _, acc := range b.Accounts {
		require.NotEmpty(t, acc.Data, "account %s has no data", acc.Pubkey)
		decoded, err := base64.StdEncoding.DecodeString(acc.Data[0])
		require.NoError(t, err, "decode account data")
		m.SetAccount(acc.Pubkey, acc.Owner.String(), decoded)
	}

	tx := b.Transaction // addressable copy
	if len(tx.Message.AddressTableLookups) > 0 {
		tx.Message.SetAddressTableLookups(tx.Message.AddressTableLookups)
	}
	txBytes, err := tx.MarshalBinary()
	require.NoError(t, err, "marshal bundle transaction")
	meta := b.Meta
	if meta.Err == nil && len(meta.LogMessages) == 0 {
		// fetchBlock filters transactions by Wormhole-looking logs before parsing
		// instructions. Generated bundles model the transaction and account data, so
		// synthesize the minimal logs needed to reach the watcher parsing logic.
		meta.LogMessages = []string{
			fmt.Sprintf("Program %s invoke [1]", b.Contract),
			"Program log: Sequence: 1",
		}
	}

	txWithMeta := rpc.TransactionWithMeta{
		Slot:        b.Slot,
		Transaction: rpc.DataBytesOrJSONFromBytes(txBytes),
		Meta:        &meta,
		Version:     rpc.LegacyTransactionVersion,
	}
	m.blocks[b.Slot] = &rpc.GetBlockResult{Transactions: []rpc.TransactionWithMeta{txWithMeta}}

	if len(tx.Signatures) > 0 {
		m.transactions[tx.Signatures[0]] = makeGetTransactionResult(t, b.Slot, txBytes, &meta)
	}
	return m
}

// makeGetTransactionResult round-trips through JSON so the private solana-go envelope
// fields are populated exactly as they are for a real base64 getTransaction response.
func makeGetTransactionResult(t *testing.T, slot uint64, txBytes []byte, meta *rpc.TransactionMeta) *rpc.GetTransactionResult {
	t.Helper()

	raw := map[string]interface{}{
		"slot":        slot,
		"transaction": []string{base64.StdEncoding.EncodeToString(txBytes), "base64"},
		"meta":        meta,
		"version":     "legacy",
	}
	encoded, err := json.Marshal(raw)
	require.NoError(t, err, "marshal getTransaction fixture")

	var result rpc.GetTransactionResult
	require.NoError(t, json.Unmarshal(encoded, &result), "decode getTransaction fixture")
	return &result
}

// drain collects the VAA signing digests published by one replay flow. The caller
// compares the sorted digest list with the recorded regression baseline.
func drain(t *testing.T, msgC <-chan *common.MessagePublication, errC <-chan error, expect expectedCount) (digests []string, replayErrs []error) {
	t.Helper()
	if expect.collectUntilQuiet {
		return collectUntilQuiet(t, msgC, errC)
	}
	return collectExpected(t, msgC, errC, expect.count)
}

// drainPolling collects output from processNewTransactions, which returns before
// processTransactionWithRetry finishes. Zero-output fixtures must wait for the
// channel to stay quiet, otherwise a delayed unexpected publication can be missed.
// Future refactor: split the watcher async boundaries so RunWithScissors wrappers
// spawn deterministic serial workers that replay tests can call directly.
func drainPolling(t *testing.T, msgC <-chan *common.MessagePublication, errC <-chan error, expect expectedCount) (digests []string, replayErrs []error) {
	t.Helper()
	if expect.collectUntilQuiet || expect.count == 0 {
		return collectUntilQuiet(t, msgC, errC)
	}
	return collectExpected(t, msgC, errC, expect.count)
}

func collectExpected(t *testing.T, msgC <-chan *common.MessagePublication, errC <-chan error, expect int) (digests []string, replayErrs []error) {
	t.Helper()
	const perMsg = 3 * time.Second
	for i := 0; i < expect; i++ {
		select {
		case msg := <-msgC:
			require.NotNil(t, msg, "nil publication")
			digests = append(digests, msg.CreateDigest())
		case <-time.After(perMsg):
			return finishDrain(digests, msgC, errC)
		}
	}
	return finishDrain(digests, msgC, errC)
}

// collectUntilQuiet is used during fixture generation, when a flow's count is not
// yet known, and for zero-output polling fixtures where processNewTransactions
// returns before its worker goroutine has necessarily finished.
func collectUntilQuiet(t *testing.T, msgC <-chan *common.MessagePublication, errC <-chan error) (digests []string, replayErrs []error) {
	t.Helper()
	const settle = 1 * time.Second
	for {
		select {
		case msg := <-msgC:
			require.NotNil(t, msg, "nil publication")
			digests = append(digests, msg.CreateDigest())
		case <-time.After(settle):
			return finishDrain(digests, msgC, errC)
		}
	}
}

func finishDrain(digests []string, msgC <-chan *common.MessagePublication, errC <-chan error) ([]string, []error) {
	for {
		select {
		case msg := <-msgC:
			if msg != nil {
				digests = append(digests, msg.CreateDigest())
			}
		default:
			var errs []error
			for {
				select {
				case err := <-errC:
					errs = append(errs, err)
				default:
					sort.Strings(digests)
					return digests, errs
				}
			}
		}
	}
}

func reobserveTransactionOutput(t *testing.T, b *replayBundle, expect expectedCount) (digests []string, replayErrs []error) {
	t.Helper()

	msgC := make(chan *common.MessagePublication, 64)
	s := newReplayWatcher(t, b, msgC)
	m := seedReplayRPCClient(t, b)
	require.NotEmpty(t, b.Transaction.Signatures, "bundle has no transaction signature")
	_, err := s.handleReobservationRequest(vaa.ChainIDSolana, b.Transaction.Signatures[0][:], m)
	if b.Meta.Err == nil {
		require.NoError(t, err, "reobserve transaction")
	} else {
		require.Error(t, err, "failed transaction reobservation should return an error")
	}
	return drain(t, msgC, s.errC, expect)
}

func fetchBlockOutput(t *testing.T, b *replayBundle, expect expectedCount) (digests []string, replayErrs []error) {
	t.Helper()

	msgC := make(chan *common.MessagePublication, 64)
	s := newReplayWatcher(t, b, msgC)
	m := seedReplayRPCClient(t, b)
	s.rpcClient = m
	require.True(t, s.fetchBlock(context.Background(), s.logger, b.Slot, 0, false), "fetch block")
	return drain(t, msgC, s.errC, expect)
}

func processNewTransactionsOutput(t *testing.T, b *replayBundle, expect expectedCount) (digests []string, replayErrs []error) {
	t.Helper()

	msgC := make(chan *common.MessagePublication, 64)
	s := newReplayWatcher(t, b, msgC)
	m := seedReplayRPCClient(t, b)
	require.NotEmpty(t, b.Transaction.Signatures, "bundle has no transaction signature")
	m.signatures[b.Contract] = []*rpc.TransactionSignature{{Signature: b.Transaction.Signatures[0]}}
	s.rpcClient = m

	require.NoError(t, s.processNewTransactions(), "process new transactions")
	return drainPolling(t, msgC, s.errC, expect)
}

// reobserveAccountOutput feeds every served account through the by-account reobservation path.
func reobserveAccountOutput(t *testing.T, b *replayBundle, expect expectedCount) (digests []string, replayErrs []error) {
	t.Helper()

	msgC := make(chan *common.MessagePublication, 64)
	s := newReplayWatcher(t, b, msgC)
	m := seedReplayRPCClient(t, b)

	seen := map[solana.PublicKey]bool{}
	for _, acc := range b.Accounts {
		if seen[acc.Pubkey] {
			continue
		}
		seen[acc.Pubkey] = true
		_, err := s.handleReobservationRequest(vaa.ChainIDSolana, acc.Pubkey[:], m)
		require.NoError(t, err, "reobserve account %s", acc.Pubkey)
	}
	return drain(t, msgC, s.errC, expect)
}

// insertExpected sets the "expected" field on a bundle's JSON object. It decodes the
// bundle into a map of raw fields so the existing transaction/meta/account values are
// preserved verbatim, adds (or replaces) the expected entry, and re-encodes.
func insertExpected(raw json.RawMessage, exp *expectedOutput) (json.RawMessage, error) {
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("decode bundle object: %w", err)
	}
	expJSON, err := json.Marshal(exp)
	if err != nil {
		return nil, fmt.Errorf("marshal expected: %w", err)
	}
	fields["expected"] = expJSON
	out, err := json.Marshal(fields)
	if err != nil {
		return nil, fmt.Errorf("encode bundle object: %w", err)
	}
	return out, nil
}
