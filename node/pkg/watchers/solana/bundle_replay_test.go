package solana

// This file contains the main deterministic replay regression test for Solana watcher
// backtesting fixtures. It intentionally keeps the test body focused on the durable
// behavior we care about: for each committed bundle, replay the same Solana inputs
// through each high-level watcher entrypoint and compare the resulting
// MessagePublication digest sets against the recorded fixture.
//
// The helper/setup code that turns bundle JSON into mock RPC responses lives in
// bundle_replay_setup_test.go. That split keeps this file oriented around adding new
// replay flows and assertions, rather than fixture decoding and watcher bootstrapping.
//
// General algorithm:
//   - Load the committed matrix from testdata/generated_bundles.json (synthetic) and
//     testdata/real_bundles.json (live-collected).
//   - For each bundle, run transaction reobservation, block observation, transaction
//     polling, and by-account reobservation.
//   - Assert each flow's calculated digests exactly match its recorded digest list.
//   - If a bundle has no expected output, fail by default; only
//     UPDATE_SOLANA_REPLAY_FIXTURES=1 records new expected digest lists.

import (
	"encoding/json"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReplayGeneratedBundles reads the committed bundle matrix and replays each bundle
// through the high-level watcher entrypoints. The main regression signal is the
// per-flow MessagePublication digest set: any dropped digest or new digest means the
// watcher would produce different VAA body hashes for the same Solana inputs.
func TestReplayGeneratedBundles(t *testing.T) {
	updateFixtures := os.Getenv(updateReplayFixturesEnv) == "1"

	sources := loadBundleSources(t)

	t.Run("replay", func(t *testing.T) {
		for _, src := range sources {
			src := src
			for i := range src.raw {
				i := i
				var b replayBundle
				require.NoErrorf(t, json.Unmarshal(src.raw[i], &b), "decode bundle %d in %s", i, src.path)

				t.Run(b.Name, func(t *testing.T) {
					t.Parallel()

					reobExpect, obsExpect, pollExpect, acctExpect := expectedCounts(b.Expected)
					reobDigests, reobErrs := reobserveTransactionOutput(t, &b, reobExpect)
					obsDigests, obsErrs := fetchBlockOutput(t, &b, obsExpect)
					pollDigests, pollErrs := processNewTransactionsOutput(t, &b, pollExpect)
					acctDigests, acctErrs := reobserveAccountOutput(t, &b, acctExpect)

					assert.Empty(t, reobErrs, "no scissored errors expected (reobservation)")
					assert.Empty(t, obsErrs, "no scissored errors expected (observation)")
					assert.Empty(t, pollErrs, "no scissored errors expected (transaction polling)")
					assert.Empty(t, acctErrs, "no scissored errors expected (account reobservation)")

					if b.Expected == nil {
						recordExpected(t, updateFixtures, b.Name, i, src.recorded, reobDigests, obsDigests, pollDigests, acctDigests)
						return
					}

					flows := []struct {
						name string
						want []string
						got  []string
					}{
						{name: "reobservation", want: b.Expected.Reobservation, got: reobDigests},
						{name: "observation", want: b.Expected.Observation, got: obsDigests},
						{name: "transaction polling", want: b.Expected.Polling, got: pollDigests},
						{name: "account reobservation", want: b.Expected.Account, got: acctDigests},
					}
					for _, flow := range flows {
						assertFlow(t, b.Name, flow.name, flow.want, flow.got)
					}
				})
			}
		}
	})

	for _, src := range sources {
		writeRecordedExpectations(t, src.path, src.raw, src.recorded)
	}
}

// bundleSource is one loaded fixture file: its raw bundle bytes plus a per-index slot for
// any expected output recorded this run (written back into this same file).
type bundleSource struct {
	path     string
	raw      []json.RawMessage
	recorded []*expectedOutput
}

// loadBundleSources reads every fixture file in bundleFiles into its own bundleSource so
// recorded expectations round-trip back to the file each bundle came from.
func loadBundleSources(t *testing.T) []*bundleSource {
	t.Helper()
	sources := make([]*bundleSource, 0, len(bundleFiles))
	total := 0
	for _, path := range bundleFiles {
		raw, err := os.ReadFile(path)
		require.NoErrorf(t, err, "read %s; generate the synthetic matrix with: "+
			"go run ./pkg/watchers/solana/testgen/cmd --out ./pkg/watchers/solana/%s", path, generatedBundlesFile)

		var rawBundles []json.RawMessage
		require.NoErrorf(t, json.Unmarshal(raw, &rawBundles), "decode bundle array in %s", path)
		sources = append(sources, &bundleSource{
			path:     path,
			raw:      rawBundles,
			recorded: make([]*expectedOutput, len(rawBundles)),
		})
		total += len(rawBundles)
	}
	require.NotZero(t, total, "no bundles in %v", bundleFiles)
	return sources
}

func expectedCounts(exp *expectedOutput) (reob, obs, poll, acct int) {
	if exp == nil {
		return -1, -1, -1, -1
	}
	return len(exp.Reobservation), len(exp.Observation), len(exp.Polling), len(exp.Account)
}

func recordExpected(
	t *testing.T,
	updateFixtures bool,
	name string,
	idx int,
	recordedByIndex []*expectedOutput,
	reobDigests []string,
	obsDigests []string,
	pollDigests []string,
	acctDigests []string,
) {
	t.Helper()
	if !updateFixtures {
		t.Fatalf("%q has no expected output; set %s=1 to record fixtures", name, updateReplayFixturesEnv)
	}
	recordedByIndex[idx] = &expectedOutput{
		Reobservation: reobDigests,
		Observation:   obsDigests,
		Polling:       pollDigests,
		Account:       acctDigests,
	}
	t.Logf("recorded expected for %q: reob=%d obs=%d poll=%d acct=%d",
		name, len(reobDigests), len(obsDigests), len(pollDigests), len(acctDigests))
}

func writeRecordedExpectations(t *testing.T, path string, rawBundles []json.RawMessage, recordedByIndex []*expectedOutput) {
	t.Helper()
	changed := false
	for i := range recordedByIndex {
		if recordedByIndex[i] == nil {
			continue
		}
		newRaw, err := insertExpected(rawBundles[i], recordedByIndex[i])
		require.NoErrorf(t, err, "insert expected for bundle %d in %s", i, path)
		rawBundles[i] = newRaw
		changed = true
	}
	if !changed {
		return
	}
	// Minified to keep the fixture file small; the decoder ignores whitespace on read.
	out, err := json.Marshal(rawBundles)
	require.NoError(t, err, "marshal bundle array")
	require.NoError(t, os.WriteFile(path, append(out, '\n'), 0644), "write %s", path)
	t.Logf("recorded expected output into %s; re-run to assert against them", path)
}

// assertFlow checks that a single flow reproduced exactly its recorded digest set. `got`
// is already sorted by the drain; a mismatch is broken down into the specific missing
// and extra digests for a legible failure.
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
