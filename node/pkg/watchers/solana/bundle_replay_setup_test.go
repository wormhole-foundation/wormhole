package solana

// This file contains the bootstrap machinery for the deterministic Solana replay
// tests. The committed bundle JSON already describes transactions, metadata, and
// account data; these helpers translate that data into the in-memory solanaRPCClient
// responses consumed by the real watcher entrypoints.
//
// The setup deliberately avoids a JSON-RPC HTTP server. Each test flow talks to the
// same solanaRPCClient interface used by production code, but all getAccountInfo,
// getBlock, getTransaction, and getSignaturesForAddress responses are served from the
// bundle. This keeps tests deterministic and fast while still exercising the watcher at
// high-level boundaries.
//
// Non-obvious replay details handled here:
//   - Bundles decoded from JSON lose solana-go's private version marker, so ALT-backed
//     transactions are re-marked versioned before being serialized into mock RPC data.
//   - fetchBlock filters transactions by Wormhole-looking logs before parsing
//     instructions, so generated bundles without logs receive minimal synthetic logs.
//   - getTransaction fixtures round-trip through JSON to populate solana-go's private
//     transaction envelope fields exactly like a real base64 RPC response.
//   - Normal test runs drain known per-flow publication counts without sleeping; the
//     quiet-window drain is reserved for explicit first-time fixture generation.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// bundlesFile is the single JSON file holding the entire generated bundle matrix (an
// array of bundles). Regenerate it with:
//
//	go run ./pkg/watchers/solana/testgen/cmd --matrix all --out ./pkg/watchers/solana/testdata/bundles.json
const bundlesFile = "testdata/bundles.json"

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
	// Expected is the recorded output of replaying this bundle through the four
	// high-level flows. It is generated in explicit fixture-update mode and asserted
	// on every normal run.
	Expected *expectedOutput `json:"expected,omitempty"`
}

type replayAccount struct {
	Pubkey solana.PublicKey `json:"pubkey"`
	Owner  solana.PublicKey `json:"owner"`
	Data   []string         `json:"data"` // [base64Data, "base64"]
}

// expectedOutput is the reproducible signature of a replay: the sorted per-message
// digests (CreateDigest) published by each replay flow, recorded separately. Each flow
// is asserted against its own exact digest list; this preserves intentional differences
// between normal observation, transaction polling, and reobservation behavior.
//
// The four flows:
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

// newReplayWatcher builds the minimum watcher state needed to exercise the real Solana
// observation and reobservation entrypoints.
func newReplayWatcher(t *testing.T, b *replayBundle, msgC chan<- *common.MessagePublication) *SolanaWatcher {
	t.Helper()

	s := newTestWatcher(t, vaa.ChainIDSolana, rpc.CommitmentFinalized, msgC)
	s.errC = make(chan error, 64)
	s.ctx = context.Background()
	s.contract = b.Contract
	s.whLogPrefix = fmt.Sprintf("Program %s", b.Contract)
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
	// A bundle loaded from JSON loses the message version (a private field with no custom
	// UnmarshalJSON), so IsVersioned would be false and the watcher would skip ALT
	// resolution. Re-mark it as versioned before serializing it into RPC responses.
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

// drain collects the published MessagePublication digests for one flow. The digest is
// the regression signal: it commits to the msgpub fields that form the VAA body/hash.
func drain(t *testing.T, msgC <-chan *common.MessagePublication, errC <-chan error, expect int) (digests []string, replayErrs []error) {
	t.Helper()
	if expect < 0 {
		return collectUntilQuiet(t, msgC, errC)
	}
	return collectExpected(t, msgC, errC, expect)
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

// collectUntilQuiet is used only during fixture generation, when a flow's count is not
// yet known. Normal runs use collectExpected and do not wait in the happy path.
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

func reobserveTransactionOutput(t *testing.T, b *replayBundle, expect int) (digests []string, replayErrs []error) {
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

func fetchBlockOutput(t *testing.T, b *replayBundle, expect int) (digests []string, replayErrs []error) {
	t.Helper()

	msgC := make(chan *common.MessagePublication, 64)
	s := newReplayWatcher(t, b, msgC)
	m := seedReplayRPCClient(t, b)
	s.rpcClient = m
	require.True(t, s.fetchBlock(context.Background(), s.logger, b.Slot, 0, false), "fetch block")
	return drain(t, msgC, s.errC, expect)
}

func processNewTransactionsOutput(t *testing.T, b *replayBundle, expect int) (digests []string, replayErrs []error) {
	t.Helper()

	msgC := make(chan *common.MessagePublication, 64)
	s := newReplayWatcher(t, b, msgC)
	m := seedReplayRPCClient(t, b)
	require.NotEmpty(t, b.Transaction.Signatures, "bundle has no transaction signature")
	m.signatures[b.Contract] = []*rpc.TransactionSignature{{Signature: b.Transaction.Signatures[0]}}
	s.rpcClient = m

	require.NoError(t, s.processNewTransactions(), "process new transactions")
	return drain(t, msgC, s.errC, expect)
}

// reobserveAccountOutput feeds every served account through the by-account reobservation
// path. Non-message accounts simply produce no publication, so no transaction parsing is
// needed here to identify message-account keys.
func reobserveAccountOutput(t *testing.T, b *replayBundle, expect int) (digests []string, replayErrs []error) {
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

// insertExpected returns a copy of a single bundle's raw JSON with a freshly recorded
// expected block appended before its closing brace. Every original field is preserved
// byte-for-byte; the value is inserted compactly and re-indented with the whole array.
func insertExpected(raw json.RawMessage, exp *expectedOutput) (json.RawMessage, error) {
	expJSON, err := json.Marshal(exp)
	if err != nil {
		return nil, fmt.Errorf("marshal expected: %w", err)
	}
	trimmed := strings.TrimRight(string(raw), " \t\r\n")
	idx := strings.LastIndex(trimmed, "}")
	if idx < 0 {
		return nil, fmt.Errorf("bundle is not a JSON object")
	}
	body := strings.TrimRight(trimmed[:idx], " \t\r\n")
	return json.RawMessage(body + `,"expected":` + string(expJSON) + `}`), nil
}
