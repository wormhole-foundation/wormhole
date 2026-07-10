package solana

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/assert"
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
	Name            string              `json:"name"`
	IsReobservation bool                `json:"isReobservation"`
	Slot            uint64              `json:"slot"`
	Contract        solana.PublicKey    `json:"contract"`
	ShimContract    solana.PublicKey    `json:"shimContract"`
	Transaction     solana.Transaction  `json:"transaction"`
	Meta            rpc.TransactionMeta `json:"meta"`
	Accounts        []replayAccount     `json:"accounts"`
	// Expected is the recorded output of replaying this bundle through
	// processTransaction: how many messages were published and their digests. It is
	// generated on first run (when absent) and asserted on every run thereafter.
	Expected *expectedOutput `json:"expected,omitempty"`
}

type replayAccount struct {
	Pubkey solana.PublicKey `json:"pubkey"`
	Owner  solana.PublicKey `json:"owner"`
	Data   []string         `json:"data"` // [base64Data, "base64"]
}

// expectedOutput is the reproducible signature of a replay: the sorted per-message
// digests (CreateDigest) published by EACH replay flow, recorded separately. Storing
// every flow's own digest set — rather than one "complete" set plus subset checks — lets
// each flow drain its known count: expected publications never wait, and a drain only
// blocks (up to the per-message timeout) when a message the flow was told to expect is
// missing, i.e. a real regression. A missing digest that was present before, or an extra
// one, is a bug.
//
// The four flows:
//   - Reobservation: handleReobservationRequest(txID) — the complete set (includes
//     close-message events that normal observation skips).
//   - Observation:   fetchBlock — the normal guardian block-observation path (a subset).
//   - Polling:       processNewTransactions — the Fogo-style
//     getSignaturesForAddress -> getTransaction path (a subset).
//   - Account:       handleReobservationRequest(accountID) over every served account (a
//     subset: only fetchable, core-owned, finalized message accounts emit).
type expectedOutput struct {
	Reobservation []string `json:"reobservation"`
	Observation   []string `json:"observation"`
	Polling       []string `json:"polling"`
	Account       []string `json:"account"`
}

// knownPublicationCounts is a generation-time guard, NOT the source of truth for the
// checks. When a bundle's `expected` block is first recorded, if the bundle's name
// appears here and the freshly observed count disagrees, generation fails loudly so a
// known-wrong output is never baked into a fixture. It captures the counts we reasoned
// about when authoring the matrices; the digests themselves are recorded fresh.
func knownPublicationCounts() map[string]int {
	m := map[string]int{
		// Curated matrix (testdata/bundles).
		"regular_outer_reliable":              1,
		"regular_outer_unreliable":            1,
		"regular_inner_reliable":              1,
		"shim_direct":                         1,
		"shim_integrator":                     1,
		"shim_bad_ordering_event_before_core": 0, // event-before-core: shimProcessRest errors, nothing published
		"close_outer":                         1,
		"close_inner":                         1,
		"regular_wrong_owner":                 0, // owner != core bridge: fetch rejected
		"regular_not_finalized":               0, // MessageStatus != 0: account not finalized
		"regular_bad_prefix":                  0, // prefix not msg/msu: NewMessageAccountData rejects
		"mixed_multi":                         4, // two regulars + shim + close
	}

	// Boundary matrix (testdata/bundles_boundary): nonce / sequence / timestamp at
	// 0 / middle / max, for every type and location. Every case is an otherwise-valid
	// finalized message, and none of these fields are validated by the watcher, so
	// each must produce exactly one publication.
	for _, ty := range []string{"postmessage", "postmessageunreliable", "shim", "close"} {
		for _, loc := range []string{"outer", "inner"} {
			for _, field := range []string{"nonce", "sequence", "timestamp"} {
				for _, level := range []string{"zero", "mid", "max"} {
					m[fmt.Sprintf("%s_%s_%s_%s", ty, loc, field, level)] = 1
				}
			}
		}
	}

	// Situations matrix (testdata/bundles_situations): edge cases per type/location.
	allTypes := []string{"postmessage", "postmessageunreliable", "shim", "close"}
	locs := []string{"outer", "inner"}
	for _, ty := range allTypes {
		for _, loc := range locs {
			// Empty payload publishes, EXCEPT unreliable+empty is dropped (client.go:1238).
			empty := 1
			if ty == "postmessageunreliable" {
				empty = 0
			}
			m[fmt.Sprintf("payload_empty_%s_%s", ty, loc)] = empty
			m[fmt.Sprintf("payload_large_%s_%s", ty, loc)] = 1
			m[fmt.Sprintf("payload_allbytes_%s_%s", ty, loc)] = 1
			m[fmt.Sprintf("largeaccounts_%s_%s", ty, loc)] = 1
			m[fmt.Sprintf("diffindex_%s_%s", ty, loc)] = 1
		}
	}
	for _, ty := range []string{"postmessage", "postmessageunreliable"} {
		for _, loc := range locs {
			m[fmt.Sprintf("wrongowner_%s_%s", ty, loc)] = 0
		}
	}
	for _, ty := range []string{"postmessage", "postmessageunreliable", "close"} {
		for _, loc := range locs {
			m[fmt.Sprintf("badprefix_%s_%s", ty, loc)] = 0
		}
	}
	for _, ty := range []string{"postmessage", "postmessageunreliable", "close"} {
		for _, loc := range locs {
			m[fmt.Sprintf("wrongcpi_%s_%s", ty, loc)] = 0
		}
	}
	for _, tag := range []string{"core", "event"} {
		for _, loc := range locs {
			m[fmt.Sprintf("wrongcpi_shim_%s_%s", tag, loc)] = 0
		}
	}

	// Triplets matrix (testdata/bundles_triplets): 40 bundles, each executing three
	// distinct (type, location) variants in one transaction. Every message is valid, so
	// each triplet publishes exactly three. Names mirror cmd/main.go's tripletScenarios
	// (first 40 lexicographic 3-combinations of the 8 variants).
	variantCodes := []string{"pmo", "pmi", "pmuo", "pmui", "sho", "shi", "clo", "cli"}
	idx := 0
	for i := 0; i < 8 && idx < 40; i++ {
		for j := i + 1; j < 8 && idx < 40; j++ {
			for k := j + 1; k < 8 && idx < 40; k++ {
				m[fmt.Sprintf("triplet_%02d_%s_%s_%s", idx, variantCodes[i], variantCodes[j], variantCodes[k])] = 3
				idx++
			}
		}
	}

	// Tx-status matrix (testdata bundles named txfailed_*): the transaction is marked
	// failed on-chain (meta.Err set), so validateTransactionMeta rejects it before any
	// instruction runs — nothing is published, for every type and location.
	for _, ty := range allTypes {
		for _, loc := range locs {
			m[fmt.Sprintf("txfailed_%s_%s", ty, loc)] = 0
		}
	}
	return m
}

// newReplayWatcher builds the minimum watcher state needed to exercise the real Solana
// observation and reobservation entrypoints. The bundle supplies the active core and
// shim program IDs so the replay follows the same program-matching decisions as a live
// watcher.
func newReplayWatcher(t *testing.T, b *replayBundle, msgC chan<- *common.MessagePublication) *SolanaWatcher {
	t.Helper()

	s := newTestWatcher(t, vaa.ChainIDSolana, rpc.CommitmentFinalized, msgC)
	s.errC = make(chan error, 64)
	s.ctx = context.Background()
	s.contract = b.Contract
	s.whLogPrefix = fmt.Sprintf("Program %s", b.Contract)
	// The shim is always enabled on Solana, so wire it up for every bundle.
	s.shimContractAddr = b.ShimContract
	s.shimContractStr = b.ShimContract.String()
	s.shimSetup()
	return s
}

// seedReplayRPCClient converts a generated bundle into the RPC responses consumed by
// the high-level watcher paths: getAccountInfo, getBlock, and getTransaction. This keeps
// replay deterministic while still testing the same RPC interface calls used in normal
// operation.
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
	// UnmarshalJSON), so IsVersioned() would be false and the watcher would skip ALT
	// resolution. Re-mark it as versioned; the ALT accounts are in b.Accounts and served
	// to the mock above, so processTransaction's populateLookupTableAccounts resolves them.
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
//
// expect is the number of publications the flow is known to produce, or -1 when it is not
// yet known (first-time fixture generation). When known we drain exactly that many and
// return the instant the last one arrives — no settle wait — so a correctly behaving flow
// never sleeps; the only wait is the per-message timeout when an expected message is
// missing (a dropped publication). When unknown we collect until the channel is quiet for
// a full settle window so the true set is discovered.
func drain(t *testing.T, msgC <-chan *common.MessagePublication, errC <-chan error, expect int) (digests []string, replayErrs []error) {
	t.Helper()
	if expect < 0 {
		return collectUntilQuiet(t, msgC, errC)
	}
	return collectExpected(t, msgC, errC, expect)
}

// collectExpected drains exactly `expect` publications (3s per-message timeout, stopping
// early if one never arrives) then finishes. No settle in the happy path.
func collectExpected(t *testing.T, msgC <-chan *common.MessagePublication, errC <-chan error, expect int) (digests []string, replayErrs []error) {
	t.Helper()
	const perMsg = 3 * time.Second
	for i := 0; i < expect; i++ {
		select {
		case msg := <-msgC:
			require.NotNil(t, msg, "nil publication")
			digests = append(digests, msg.CreateDigest())
		case <-time.After(perMsg):
			// A message the flow expects never arrived; stop waiting and let the caller
			// report exactly which digests dropped.
			return finishDrain(digests, msgC, errC)
		}
	}
	return finishDrain(digests, msgC, errC)
}

// collectUntilQuiet drains publications until the channel is idle for a full settle
// window. Used only during fixture generation, when the per-flow count is not yet known.
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

// finishDrain non-blockingly sweeps any already-queued surplus publications (so an extra
// beyond the expected count is still caught) and any scissored errors, then sorts the
// digests for order-independent comparison.
//
// Note the surplus sweep is best-effort for the ASYNC flows (reobservation, observation,
// polling): a spurious extra that has not yet been sent by its goroutine when we finish is
// not waited for — matching the design goal that a correct flow never sleeps. Such an
// extra is still guarded by the reobservation flow's exact match over the superset. The
// account flow is synchronous, so every publication is already queued here and its
// extra-detection is complete.
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
	// Transaction reobservation is the canonical complete path: it replays the full
	// transaction and includes close-message events that normal observation skips.
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
	// fetchBlock is the normal guardian observation entrypoint: it fetches a block,
	// applies log prefiltering, decodes transactions, and then processes instructions.
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

	// processNewTransactions is the poll-for-transactions path (used by Fogo). It
	// discovers the signature via getSignaturesForAddress, fetches it with getTransaction,
	// and then feeds the result into the normal transaction processor asynchronously.
	require.NoError(t, s.processNewTransactions(), "process new transactions")
	return drain(t, msgC, s.errC, expect)
}

// reobserveAccountOutput runs every served account through the guardian by-account
// reobservation entrypoint. This mirrors a peer asking to reobserve a specific posted
// message account. Only a fetchable, core-owned, finalized message account emits; any
// other account (an ALT table, a wrong-owner or non-finalized account, ...) yields
// nothing — the account-id path discards the fetch error and returns (0, nil) — so we can
// feed it every account without parsing the transaction to pick out the message accounts.
// The path is fully synchronous, so all publications are queued before we drain.
func reobserveAccountOutput(t *testing.T, b *replayBundle, expect int) (digests []string, replayErrs []error) {
	t.Helper()

	msgC := make(chan *common.MessagePublication, 64)
	s := newReplayWatcher(t, b, msgC)
	m := seedReplayRPCClient(t, b)

	seen := map[solana.PublicKey]bool{}
	for _, acc := range b.Accounts {
		if seen[acc.Pubkey] {
			continue // a duplicate account would double-count its publication
		}
		seen[acc.Pubkey] = true
		_, err := s.handleReobservationRequest(vaa.ChainIDSolana, acc.Pubkey[:], m)
		require.NoError(t, err, "reobserve account %s", acc.Pubkey)
	}
	return drain(t, msgC, s.errC, expect)
}

// insertExpected returns a copy of a single bundle's raw JSON with a freshly recorded
// `expected` block appended before its closing brace. Every original field is preserved
// byte-for-byte; the value is inserted compactly and re-indented when the whole array is
// marshaled with json.MarshalIndent.
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

// TestReplayGeneratedBundles reads the single committed bundle file and replays every
// bundle through high-level watcher entrypoints. The recorded `expected` block is the
// transaction reobservation output — the complete set of publications.
//
//   - Reobservation flow: every recorded digest must appear and none extra (exact match).
//   - Block observation flow: a subset is allowed — a recorded digest may legitimately
//     not appear (e.g. close-message events are only processed on reobservation) — but
//     no digest outside the recorded set may appear.
//   - Transaction polling flow follows the same subset rule, but reaches processing via
//     getSignaturesForAddress and getTransaction rather than getBlock.
//
// If a bundle has no `expected` block yet, set UPDATE_SOLANA_REPLAY_FIXTURES=1 to
// record the reobservation output back into the same file.
func TestReplayGeneratedBundles(t *testing.T) {
	guard := knownPublicationCounts()
	updateFixtures := os.Getenv(updateReplayFixturesEnv) == "1"

	raw, err := os.ReadFile(bundlesFile)
	require.NoErrorf(t, err, "read %s; generate it with: go run ./pkg/watchers/solana/testgen/cmd --matrix all --out ./pkg/watchers/solana/%s", bundlesFile, bundlesFile)

	var rawBundles []json.RawMessage
	require.NoError(t, json.Unmarshal(raw, &rawBundles), "decode bundle array")
	require.NotEmpty(t, rawBundles, "no bundles in %s", bundlesFile)

	// Subtests run in parallel; each stages its recorded output at its own index so the
	// file is written once after they all complete (below the group subtest).
	recordedByIndex := make([]*expectedOutput, len(rawBundles))

	t.Run("replay", func(t *testing.T) {
		for i := range rawBundles {
			i := i
			var b replayBundle
			require.NoErrorf(t, json.Unmarshal(rawBundles[i], &b), "decode bundle %d", i)

			t.Run(b.Name, func(t *testing.T) {
				t.Parallel()

				// Transaction reobservation flow — the complete set.
				reobExpect, obsExpect, pollExpect, acctExpect := -1, -1, -1, -1
				if b.Expected != nil {
					reobExpect = len(b.Expected.Reobservation)
					obsExpect = len(b.Expected.Observation)
					pollExpect = len(b.Expected.Polling)
					acctExpect = len(b.Expected.Account)
				}
				reobDigests, reobErrs := reobserveTransactionOutput(t, &b, reobExpect)
				assert.Empty(t, reobErrs, "no scissored errors expected (reobservation)")

				// Block observation flow — a subset is allowed, so discover it via the settle
				// window rather than a known count.
				obsDigests, obsErrs := fetchBlockOutput(t, &b, obsExpect)
				assert.Empty(t, obsErrs, "no scissored errors expected (observation)")

				// Transaction polling flow — also a normal-observation subset, but reached
				// through getSignaturesForAddress -> getTransaction -> processTransaction.
				pollDigests, pollErrs := processNewTransactionsOutput(t, &b, pollExpect)
				assert.Empty(t, pollErrs, "no scissored errors expected (transaction polling)")

				// Account reobservation flow — run each post_message message account through
				// handleReobservationRequest. Like the observation path, any event it emits
				// must be in the recorded hash list; if it emits nothing, that is fine.
				fetchDigests, fetchErrs := reobserveAccountOutput(t, &b, acctExpect)
				assert.Empty(t, fetchErrs, "no scissored errors expected (account reobservation)")

				if b.Expected == nil {
					if !updateFixtures {
						t.Fatalf("%q has no expected output; set %s=1 to record fixtures", b.Name, updateReplayFixturesEnv)
					}
					// Generation: record each flow's output. Guard the complete
					// (reobservation) count against a value we know to be wrong.
					if want, ok := guard[b.Name]; ok {
						require.Equalf(t, want, len(reobDigests),
							"generated reobservation count for %q disagrees with the known-good guard; refusing to record a wrong expectation", b.Name)
					}
					recordedByIndex[i] = &expectedOutput{
						Reobservation: reobDigests,
						Observation:   obsDigests,
						Polling:       pollDigests,
						Account:       fetchDigests,
					}
					t.Logf("recorded expected for %q: reob=%d obs=%d poll=%d acct=%d",
						b.Name, len(reobDigests), len(obsDigests), len(pollDigests), len(fetchDigests))
					return
				}

				// Each flow must reproduce exactly its own recorded digest set: no missing
				// (dropped) and no extra (newly introduced) publications. The digests commit
				// to the msgpub fields that form the VAA body/hash.
				assertFlow(t, b.Name, "reobservation", b.Expected.Reobservation, reobDigests)
				assertFlow(t, b.Name, "observation", b.Expected.Observation, obsDigests)
				assertFlow(t, b.Name, "transaction polling", b.Expected.Polling, pollDigests)
				assertFlow(t, b.Name, "account reobservation", b.Expected.Account, fetchDigests)
			})
		}
	})

	// All parallel subtests have completed. Write any newly recorded expectations once.
	changed := false
	for i := range recordedByIndex {
		if recordedByIndex[i] == nil {
			continue
		}
		newRaw, err := insertExpected(rawBundles[i], recordedByIndex[i])
		require.NoErrorf(t, err, "insert expected for bundle %d", i)
		rawBundles[i] = newRaw
		changed = true
	}
	if changed {
		out, err := json.MarshalIndent(rawBundles, "", "  ")
		require.NoError(t, err, "marshal bundle array")
		require.NoError(t, os.WriteFile(bundlesFile, append(out, '\n'), 0644), "write %s", bundlesFile)
		t.Logf("recorded expected output into %s; re-run to assert against them", bundlesFile)
	}
}

// assertFlow checks that a single flow reproduced exactly its recorded digest set. `got`
// is already sorted by the drain; a mismatch is broken down into the specific missing
// (dropped) and extra (newly introduced) digests for a legible failure.
func assertFlow(t *testing.T, name, flow string, want, got []string) {
	t.Helper()
	w := append([]string(nil), want...)
	sort.Strings(w)
	if assert.Equalf(t, w, got, "%q %s digests changed", name, flow) {
		return
	}
	missing, extra := diffDigests(w, got)
	if len(missing) > 0 {
		t.Errorf("%q %s: %d recorded message(s) no longer published (missing): %v", name, flow, len(missing), missing)
	}
	if len(extra) > 0 {
		t.Errorf("%q %s: %d unexpected extra message(s) (new digests): %v", name, flow, len(extra), extra)
	}
}

// diffDigests returns the digests present in want but not got (missing), and in got but
// not want (extra), treating each as a multiset.
func diffDigests(want, got []string) (missing, extra []string) {
	wc := map[string]int{}
	for _, d := range want {
		wc[d]++
	}
	gc := map[string]int{}
	for _, d := range got {
		gc[d]++
	}
	for d, n := range wc {
		for i := gc[d]; i < n; i++ {
			missing = append(missing, d)
		}
	}
	for d, n := range gc {
		for i := wc[d]; i < n; i++ {
			extra = append(extra, d)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return missing, extra
}
