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

// expectedOutput is the reproducible signature of a replay: the number of published
// messages and the sorted list of their per-message digests (CreateDigest). A missing
// digest that was present before, or an extra one, is a bug.
type expectedOutput struct {
	Count   int      `json:"count"`
	Digests []string `json:"digests"`
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

	tx := b.Transaction
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

// drainReplayOutput collects the published MessagePublication digests. The digest is
// the regression signal: it commits to the msgpub fields that form the VAA body/hash.
//
// wantCount is the expected number of publications, or -1 when unknown. When known, we
// drain exactly that many; when unknown, we collect until the channel is quiet for a
// full settle window so the true count is discovered (needed for subset flows).
func drainReplayOutput(t *testing.T, name string, msgC <-chan *common.MessagePublication, errC <-chan error, wantCount int) (digests []string, replayErrs []error) {
	t.Helper()

	const perMsg = 3 * time.Second
	const settle = 1 * time.Second

	drainErrs := func() []error {
		sort.Strings(digests)
		var errs []error
		for {
			select {
			case err := <-errC:
				errs = append(errs, err)
			default:
				return errs
			}
		}
	}

	if wantCount >= 0 {
		// Known count: drain exactly that many (generous per-message timeout) and
		// return. The digest comparison in the caller is the check.
		for i := 0; i < wantCount; i++ {
			select {
			case msg := <-msgC:
				require.NotNil(t, msg, "nil publication")
				digests = append(digests, msg.CreateDigest())
			case <-time.After(perMsg):
				t.Fatalf("timed out waiting for publication %d/%d for %q", i+1, wantCount, name)
			}
		}
		replayErrs = drainErrs()
		return digests, replayErrs
	}

	// Unknown count (first-time generation): collect until the channel goes quiet for
	// a full settle window so the true count is discovered.
	for {
		select {
		case msg := <-msgC:
			require.NotNil(t, msg, "nil publication")
			digests = append(digests, msg.CreateDigest())
		case <-time.After(settle):
			replayErrs = drainErrs()
			return digests, replayErrs
		}
	}
}

func reobserveTransactionOutput(t *testing.T, b *replayBundle, wantCount int) (digests []string, replayErrs []error) {
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
	return drainReplayOutput(t, b.Name, msgC, s.errC, wantCount)
}

func fetchBlockOutput(t *testing.T, b *replayBundle) (digests []string, replayErrs []error) {
	t.Helper()

	msgC := make(chan *common.MessagePublication, 64)
	s := newReplayWatcher(t, b, msgC)
	m := seedReplayRPCClient(t, b)
	s.rpcClient = m
	// fetchBlock is the normal guardian observation entrypoint: it fetches a block,
	// applies log prefiltering, decodes transactions, and then processes instructions.
	require.True(t, s.fetchBlock(context.Background(), s.logger, b.Slot, 0, false), "fetch block")
	return drainReplayOutput(t, b.Name, msgC, s.errC, -1)
}

// postMessageAccounts returns the message account referenced by each core
// post_message / post_message_unreliable instruction in the bundle (top-level and
// inner), selected exactly as processInstruction does: a core-program instruction whose
// first data byte is the post-message discriminator and whose Accounts[1] is the message
// account. Non-core instructions (wrong-CPI, shim core post, close) are skipped, so only
// the accounts a real regular observation would fetch are returned.
func postMessageAccounts(b *replayBundle) []solana.PublicKey {
	// A failed transaction rolls back on-chain, so its message account is never
	// committed and cannot be fetched — the by-account path has nothing to observe.
	if b.Meta.Err != nil {
		return nil
	}

	keys := b.Transaction.Message.AccountKeys
	coreIdx := -1
	for i, k := range keys {
		if k.Equals(b.Contract) {
			coreIdx = i
			break
		}
	}
	if coreIdx < 0 {
		return nil
	}

	isPost := func(inst solana.CompiledInstruction) bool {
		return int(inst.ProgramIDIndex) == coreIdx &&
			len(inst.Data) > 0 &&
			(inst.Data[0] == postMessageInstructionID || inst.Data[0] == postMessageUnreliableInstructionID) &&
			len(inst.Accounts) >= postMessageInstructionMinNumAccounts
	}

	var accts []solana.PublicKey
	seen := map[solana.PublicKey]bool{}
	collect := func(inst solana.CompiledInstruction) {
		if !isPost(inst) {
			return
		}
		acc := keys[inst.Accounts[1]] // the VAA/message account
		if !seen[acc] {
			seen[acc] = true
			accts = append(accts, acc)
		}
	}
	for _, inst := range b.Transaction.Message.Instructions {
		collect(inst)
	}
	for _, set := range b.Meta.InnerInstructions {
		for _, inst := range set.Instructions {
			collect(inst)
		}
	}
	return accts
}

// reobserveAccountOutput runs each post_message / post_message_unreliable message
// account through the guardian reobservation entrypoint. A valid, core-owned,
// finalized message account emits; anything else emits nothing.
func reobserveAccountOutput(t *testing.T, b *replayBundle) (digests []string, replayErrs []error) {
	t.Helper()

	accts := postMessageAccounts(b)
	if len(accts) == 0 {
		return nil, nil
	}

	msgC := make(chan *common.MessagePublication, 64)
	s := newReplayWatcher(t, b, msgC)
	m := seedReplayRPCClient(t, b)

	for _, acc := range accts {
		// Account-id reobservation exercises the guardian path used when peers ask for a
		// specific posted-message account rather than a full transaction replay.
		_, err := s.handleReobservationRequest(vaa.ChainIDSolana, acc[:], m)
		require.NoError(t, err, "reobserve account")
	}
	return drainReplayOutput(t, b.Name, msgC, s.errC, -1)
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
				reobWant := -1
				if b.Expected != nil {
					reobWant = b.Expected.Count
				}
				reobDigests, reobErrs := reobserveTransactionOutput(t, &b, reobWant)
				assert.Empty(t, reobErrs, "no scissored errors expected (reobservation)")

				// Block observation flow — a subset is allowed, so discover it via the settle
				// window rather than a known count.
				obsDigests, obsErrs := fetchBlockOutput(t, &b)
				assert.Empty(t, obsErrs, "no scissored errors expected (observation)")

				// Account reobservation flow — run each post_message message account through
				// handleReobservationRequest. Like the observation path, any event it emits
				// must be in the recorded hash list; if it emits nothing, that is fine.
				fetchDigests, fetchErrs := reobserveAccountOutput(t, &b)
				assert.Empty(t, fetchErrs, "no scissored errors expected (account reobservation)")

				if b.Expected == nil {
					if !updateFixtures {
						t.Fatalf("%q has no expected output; set %s=1 to record fixtures", b.Name, updateReplayFixturesEnv)
					}
					// Generation: record the reobservation output. Guard against baking
					// in a count we know to be wrong.
					if want, ok := guard[b.Name]; ok {
						require.Equalf(t, want, len(reobDigests),
							"generated reobservation count for %q disagrees with the known-good guard; refusing to record a wrong expectation", b.Name)
					}
					// The observation and account-fetch flows must already be subsets of the
					// reobservation set.
					if _, extra := diffDigests(reobDigests, obsDigests); len(extra) > 0 {
						t.Fatalf("%q: observation published digest(s) not in the reobservation set: %v", b.Name, extra)
					}
					if _, extra := diffDigests(reobDigests, fetchDigests); len(extra) > 0 {
						t.Fatalf("%q: account reobservation published digest(s) not in the transaction reobservation set: %v", b.Name, extra)
					}
					recordedByIndex[i] = &expectedOutput{Count: len(reobDigests), Digests: reobDigests}
					t.Logf("recorded expected (reobservation) for %q: count=%d", b.Name, len(reobDigests))
					return
				}

				want := append([]string(nil), b.Expected.Digests...)
				sort.Strings(want)

				// This digest equality is the core regression check: the replayed
				// MessagePublication values must produce the same VAA body digest as the
				// committed fixture, with no missing or newly introduced publications.
				assert.Equalf(t, b.Expected.Count, len(reobDigests),
					"reobservation message count changed for %q (was %d, now %d)", b.Name, b.Expected.Count, len(reobDigests))
				if !assert.Equalf(t, want, reobDigests, "reobservation digests changed for %q", b.Name) {
					missing, extra := diffDigests(want, reobDigests)
					if len(missing) > 0 {
						t.Errorf("%q reobservation: %d recorded message(s) no longer published (missing): %v", b.Name, len(missing), missing)
					}
					if len(extra) > 0 {
						t.Errorf("%q reobservation: %d unexpected extra message(s) (new digests): %v", b.Name, len(extra), extra)
					}
				}

				// Observation: subset — a missing digest is allowed, but nothing outside
				// the recorded set may appear.
				if _, extra := diffDigests(want, obsDigests); len(extra) > 0 {
					t.Errorf("%q observation: %d message(s) published that are not in the reobservation set: %v", b.Name, len(extra), extra)
				}

				// Account reobservation: subset — every event it emits must be a recorded hash.
				if _, extra := diffDigests(want, fetchDigests); len(extra) > 0 {
					t.Errorf("%q account reobservation: %d event(s) not in the transaction reobservation set: %v", b.Name, len(extra), extra)
				}
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
