package solana

// This file contains the main regression test for the Solana watcher backtesting fixtures.
// This helps keep the invariant that the Solana watcher is idompotent now, and for all changes in the future.
// Add 'UPDATE_SOLANA_REPLAY_FIXTURES=1' to write back the digests at the end of execution.

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Regression tests that uses generated test cases, and real txs from mainnet.
func TestReplayGeneratedBundles(t *testing.T) {
	updateFixtures := os.Getenv(updateReplayFixturesEnv) == "1"

	sources := loadBundleSources(t)
	if updateFixtures {
		warnRerecord(t, sources)
	} else {
		requireRecordedExpectations(t, sources)
	}

	t.Run("replay", func(t *testing.T) {
		for _, src := range sources {
			src := src
			for i := range src.raw {
				i := i
				var b replayBundle
				require.NoErrorf(t, json.Unmarshal(src.raw[i], &b), "decode bundle %d in %s", i, src.path)

				t.Run(b.Name, func(t *testing.T) {
					t.Parallel()

					// An explicit re-record run discards the previous digests and regenerates every bundle's output
					expected := b.Expected
					if updateFixtures {
						expected = nil
					}

					reobExpect, obsExpect, pollExpect, acctExpect := expectedCounts(expected)
					reobDigests, reobErrs := reobserveTransactionOutput(t, &b, reobExpect)
					obsDigests, obsErrs := fetchBlockOutput(t, &b, obsExpect)
					pollDigests, pollErrs := processNewTransactionsOutput(t, &b, pollExpect)
					acctDigests, acctErrs := reobserveAccountOutput(t, &b, acctExpect)

					assert.Empty(t, reobErrs, "no scissored errors expected (reobservation)")
					assert.Empty(t, obsErrs, "no scissored errors expected (observation)")
					assert.Empty(t, pollErrs, "no scissored errors expected (transaction polling)")
					assert.Empty(t, acctErrs, "no scissored errors expected (account reobservation)")

					if expected == nil {
						recordExpected(t, b.Name, i, src.recorded, reobDigests, obsDigests, pollDigests, acctDigests)
						return
					}

					flows := []struct {
						name string
						want []string
						got  []string
					}{
						{name: "reobservation", want: expected.Reobservation, got: reobDigests},
						{name: "observation", want: expected.Observation, got: obsDigests},
						{name: "transaction polling", want: expected.Polling, got: pollDigests},
						{name: "account reobservation", want: expected.Account, got: acctDigests},
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
		require.NoErrorf(t, err, "read %s; regenerate fixtures with: "+
			"go run ./pkg/watchers/solana/testgen/cmd static (or `live --rpc ...`)", path)

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

// probeBundle decodes just the identifying fields of a raw bundle, without paying for the
// transaction/meta/account payload.
func probeBundle(t *testing.T, raw json.RawMessage, path string, idx int) (name string, recorded bool) {
	t.Helper()
	var probe struct {
		Name     string          `json:"name"`
		Expected *expectedOutput `json:"expected"`
	}
	require.NoErrorf(t, json.Unmarshal(raw, &probe), "decode bundle %d in %s", idx, path)
	name = probe.Name
	if name == "" {
		name = fmt.Sprintf("#%d", idx)
	}
	return name, probe.Expected != nil
}

// warnRerecord announces on stderr that this run will discard every recorded baseline and
// replace it with whatever the current code emits.
func warnRerecord(t *testing.T, sources []*bundleSource) {
	t.Helper()
	const rule = "========================================================================"
	fmt.Fprintf(os.Stderr, "\n%s\n%39s\n%s\n\n", rule, "*** WARNING ***", rule)
	for _, src := range sources {
		n := 0
		for i, raw := range src.raw {
			if _, recorded := probeBundle(t, raw, src.path, i); recorded {
				n++
			}
		}
		fmt.Fprintf(os.Stderr, "  %s: discarding %d of %d recorded `expected` block(s)\n", src.path, n, len(src.raw))
	}
	fmt.Fprintf(os.Stderr, "\n  %s=1 re-records every bundle from CURRENT behavior.\n", updateReplayFixturesEnv)
	fmt.Fprintf(os.Stderr, "  Review the resulting digest diff before committing.\n\n%s\n\n", rule)
}

// requireRecordedExpectations fails the whole replay test up front unless every bundle
// carries a recorded `expected` block.
// Add `UPDATE_SOLANA_REPLAY_FIXTURES=1` to generate the expected output data.
func requireRecordedExpectations(t *testing.T, sources []*bundleSource) {
	t.Helper()

	var missing []string
	total := 0
	for _, src := range sources {
		for i, raw := range src.raw {
			total++
			name, recorded := probeBundle(t, raw, src.path, i)
			if recorded {
				continue
			}
			missing = append(missing, src.path+":"+name)
		}
	}
	if len(missing) == 0 {
		return
	}

	const show = 10
	numMissing := len(missing)
	elided := ""
	if numMissing > show {
		elided = fmt.Sprintf(" (and %d more)", numMissing-show)
		missing = missing[:show]
	}
	t.Fatalf("%d of %d bundle(s) have no recorded expected output: %v%s\n"+
		"A bundle without a baseline asserts nothing. Record them with %s=1, then review the "+
		"recorded digests before committing.",
		numMissing, total, missing, elided, updateReplayFixturesEnv)
}

func expectedCounts(exp *expectedOutput) (reob, obs, poll, acct int) {
	if exp == nil {
		return -1, -1, -1, -1
	}
	return len(exp.Reobservation), len(exp.Observation), len(exp.Polling), len(exp.Account)
}

// recordExpected stages a bundle's freshly replayed digests for write-back. It is only ever
// reached on an UPDATE_SOLANA_REPLAY_FIXTURES=1 run: without the flag, requireRecordedExpectations
// has already failed the test if any bundle lacks a baseline.
func recordExpected(
	t *testing.T,
	name string,
	idx int,
	recordedByIndex []*expectedOutput,
	reobDigests []string,
	obsDigests []string,
	pollDigests []string,
	acctDigests []string,
) {
	t.Helper()
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
