// Command testgen emits a matrix of Solana watcher processTransaction() input
// bundles as a single JSON file holding an array of bundles. Each bundle's
// "transaction"/"meta" decode straight into solana.Transaction / rpc.TransactionMeta,
// and "accounts" is the list of getAccountInfo responses to feed a mock.
//
//	go run ./pkg/watchers/solana/testgen/cmd --matrix all --out ./bundles.json
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/certusone/wormhole/node/pkg/watchers/solana/testgen"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// key builds a deterministic pubkey from a role tag (mirrors the builder's internal
// scheme without depending on its unexported helper).
func key(tag byte) solana.PublicKey {
	var raw [32]byte
	for i := range raw {
		raw[i] = tag
	}
	return solana.PublicKeyFromBytes(raw[:])
}

var (
	core    = key(0xAA)
	shim    = key(0xDD)
	emitter = [32]byte{0x04, 0x1c, 0x65, 0x7e, 0x84, 0x5d, 0x65, 0xd0, 0x09, 0xd5, 0x9c, 0xee, 0xb1, 0xdd, 0xa1, 0x72, 0xbd, 0x6b, 0xc9, 0xe7, 0xee, 0x5a, 0x19, 0xe5, 0x65, 0x73, 0x19, 0x7c, 0xf7, 0xfd, 0xff, 0xde}
)

func baseCfg(name string) testgen.Config {
	return testgen.Config{
		Contract:          core,
		ShimContract:      shim,
		WatcherCommitment: rpc.CommitmentFinalized,
		Slot:              42,
		Name:              name,
	}
}

func fields(nonce uint32, seq uint64, payload string) testgen.WormholeFields {
	return testgen.WormholeFields{
		Nonce:          nonce,
		Payload:        []byte(payload),
		Sequence:       seq,
		EmitterChain:   uint16(vaa.ChainIDSolana),
		EmitterAddress: emitter,
		Timestamp:      1736530812,
	}
}

// scenario builds one named bundle.
type scenario struct {
	name  string
	build func() (*testgen.Bundle, error)
}

func scenarios() []scenario {
	return []scenario{
		// --- Regular (account-based), outer vs inner, reliable vs unreliable ---
		{"regular_outer_reliable", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("regular_outer_reliable")).
				AddRegular(testgen.RegularSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(1, 1, "hello")}).Build()
		}},
		{"regular_outer_unreliable", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("regular_outer_unreliable")).
				AddRegular(testgen.RegularSpec{Location: testgen.Outer, Kind: testgen.PostMessageUnreliable, Msg: fields(2, 2, "payload")}).Build()
		}},
		{"regular_inner_reliable", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("regular_inner_reliable")).
				AddRegular(testgen.RegularSpec{Location: testgen.Inner, Kind: testgen.PostMessage, Msg: fields(3, 3, "cpi-message")}).Build()
		}},

		// --- Shim, direct vs integrator ---
		{"shim_direct", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("shim_direct")).
				AddShim(testgen.ShimSpec{Topology: testgen.Direct, Msg: fields(4, 4, "shim-direct")}).Build()
		}},
		{"shim_integrator", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("shim_integrator")).
				AddShim(testgen.ShimSpec{Topology: testgen.Integrator, Msg: fields(5, 5, "shim-integrator")}).Build()
		}},
		{"shim_bad_ordering_event_before_core", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("shim_bad_ordering_event_before_core")).
				AddShim(testgen.ShimSpec{
					Topology: testgen.Integrator,
					Ordering: []testgen.ShimPart{testgen.ShimPostPart, testgen.ShimEventPart, testgen.CorePostPart},
					Msg:      fields(6, 6, "bad-order"),
				}).Build()
		}},

		// --- Close (reobservation), outer vs inner ---
		{"close_outer", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("close_outer")).
				AddClose(testgen.CloseSpec{Location: testgen.Outer, Msg: fields(7, 7, "closed")}).Build()
		}},
		{"close_inner", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("close_inner")).
				AddClose(testgen.CloseSpec{Location: testgen.Inner, Msg: fields(8, 8, "closed-cpi")}).Build()
		}},

		// --- Account-content edge cases ---
		{"regular_wrong_owner", func() (*testgen.Bundle, error) {
			badOwner := key(0x99)
			return testgen.NewBuilder(baseCfg("regular_wrong_owner")).
				AddRegular(testgen.RegularSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(9, 9, "hi"), Account: testgen.AccountContent{Owner: &badOwner}}).Build()
		}},
		{"regular_not_finalized", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("regular_not_finalized")).
				AddRegular(testgen.RegularSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(10, 10, "hi"), Account: testgen.AccountContent{MessageStatus: 1}}).Build()
		}},
		{"regular_bad_prefix", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("regular_bad_prefix")).
				AddRegular(testgen.RegularSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(11, 11, "hi"), Account: testgen.AccountContent{Prefix: "bad"}}).Build()
		}},

		// --- Mixed: several types/locations in one transaction ---
		{"mixed_multi", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("mixed_multi")).
				AddRegular(testgen.RegularSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(20, 20, "a")}).
				AddRegular(testgen.RegularSpec{Location: testgen.Inner, Kind: testgen.PostMessageUnreliable, Msg: fields(21, 21, "b")}).
				AddShim(testgen.ShimSpec{Topology: testgen.Integrator, Msg: fields(22, 22, "c")}).
				AddClose(testgen.CloseSpec{Location: testgen.Outer, Msg: fields(23, 23, "d")}).Build()
		}},
	}
}

func locOf(outer bool) testgen.Location {
	if outer {
		return testgen.Outer
	}
	return testgen.Inner
}

func locName(outer bool) string {
	if outer {
		return "outer"
	}
	return "inner"
}

// boundaryScenarios sweeps the genuinely-unbounded integer fields (nonce, sequence,
// timestamp) at 0 / middle / max, for each message type and location. Consistency
// level is deliberately NOT swept: on Solana it is a two-variant enum (Confirmed /
// Finalized), not a free integer, so it is held at a valid Finalized baseline.
//
//	4 types x 2 locations x 3 fields x 3 levels = 72 cases.
func boundaryScenarios() []scenario {
	types := []struct {
		name string
		add  func(b *testgen.Builder, outer bool, m testgen.WormholeFields)
	}{
		{"postmessage", func(b *testgen.Builder, outer bool, m testgen.WormholeFields) {
			b.AddRegular(testgen.RegularSpec{Location: locOf(outer), Kind: testgen.PostMessage, Msg: m})
		}},
		{"postmessageunreliable", func(b *testgen.Builder, outer bool, m testgen.WormholeFields) {
			b.AddRegular(testgen.RegularSpec{Location: locOf(outer), Kind: testgen.PostMessageUnreliable, Msg: m})
		}},
		{"shim", func(b *testgen.Builder, outer bool, m testgen.WormholeFields) {
			topo := testgen.Integrator
			if outer {
				topo = testgen.Direct
			}
			b.AddShim(testgen.ShimSpec{Topology: topo, Msg: m})
		}},
		{"close", func(b *testgen.Builder, outer bool, m testgen.WormholeFields) {
			b.AddClose(testgen.CloseSpec{Location: locOf(outer), Msg: m})
		}},
	}
	fields := []string{"nonce", "sequence", "timestamp"}
	levels := []string{"zero", "mid", "max"}

	u32 := func(level string) uint32 {
		switch level {
		case "zero":
			return 0
		case "mid":
			return 0x7FFFFFFF
		default:
			return 0xFFFFFFFF
		}
	}
	u64 := func(level string) uint64 {
		switch level {
		case "zero":
			return 0
		case "mid":
			return 0x7FFFFFFFFFFFFFFF
		default:
			return 0xFFFFFFFFFFFFFFFF
		}
	}

	// baseline is a valid, finalized message; only the field under test is varied.
	baseline := func() testgen.WormholeFields {
		return testgen.WormholeFields{
			Nonce:          7,
			Sequence:       99,
			Timestamp:      1736530812,
			Payload:        []byte("boundary"), // non-empty so the unreliable variant is not dropped
			Commitment:     testgen.Finalized,
			EmitterChain:   uint16(vaa.ChainIDSolana),
			EmitterAddress: emitter,
		}
	}

	var out []scenario
	for _, ty := range types {
		for _, outer := range []bool{true, false} {
			for _, field := range fields {
				for _, level := range levels {
					ty, outer, field, level := ty, outer, field, level // capture
					name := fmt.Sprintf("%s_%s_%s_%s", ty.name, locName(outer), field, level)
					out = append(out, scenario{name: name, build: func() (*testgen.Bundle, error) {
						m := baseline()
						switch field {
						case "nonce":
							m.Nonce = u32(level)
						case "sequence":
							m.Sequence = u64(level)
						case "timestamp":
							m.Timestamp = u32(level)
						}
						b := testgen.NewBuilder(baseCfg(name))
						ty.add(b, outer, m)
						return b.Build()
					}})
				}
			}
		}
	}
	return out
}

// fieldsB is a valid finalized message with the given (binary) payload.
func fieldsB(payload []byte) testgen.WormholeFields {
	return testgen.WormholeFields{
		Nonce:          7,
		Sequence:       99,
		Timestamp:      1736530812,
		Payload:        payload,
		EmitterChain:   uint16(vaa.ChainIDSolana),
		EmitterAddress: emitter,
	}
}

func allBytesPayload() []byte {
	p := make([]byte, 256)
	for i := range p {
		p[i] = byte(i)
	}
	return p
}

// addType adds one message of the named type to b, threading the edge-case knobs.
// numAccounts and wrongCore apply to the account-based paths (regular/close);
// wrongShim targets a shim CPI part; both are ignored where not applicable.
func addType(b *testgen.Builder, typeName string, outer bool, m testgen.WormholeFields, ac testgen.AccountContent, wrongCore bool, wrongShim *testgen.ShimPart, numAccounts int) {
	switch typeName {
	case "postmessage":
		b.AddRegular(testgen.RegularSpec{Location: locOf(outer), Kind: testgen.PostMessage, Msg: m, Account: ac, NumAccounts: numAccounts, WrongCoreProgram: wrongCore})
	case "postmessageunreliable":
		b.AddRegular(testgen.RegularSpec{Location: locOf(outer), Kind: testgen.PostMessageUnreliable, Msg: m, Account: ac, NumAccounts: numAccounts, WrongCoreProgram: wrongCore})
	case "shim":
		topo := testgen.Integrator
		if outer {
			topo = testgen.Direct
		}
		b.AddShim(testgen.ShimSpec{Topology: topo, Msg: m, WrongProgramPart: wrongShim})
	case "close":
		b.AddClose(testgen.CloseSpec{Location: locOf(outer), Msg: m, Account: ac, NumAccounts: numAccounts, WrongCoreProgram: wrongCore})
	}
}

// situationScenarios generates the edge-case matrix. Each category is crossed with
// {outer, inner} x {postmessage, postmessageunreliable, shim, close}, skipping combos
// that don't make sense (reported by the caller). 60 bundles.
func situationScenarios() []scenario {
	allTypes := []string{"postmessage", "postmessageunreliable", "shim", "close"}
	both := []bool{true, false}

	cfgN := func(name string, filler int) testgen.Config {
		c := baseCfg(name)
		c.LeadingFillerAccounts = filler
		return c
	}

	var out []scenario
	add := func(name string, build func() (*testgen.Bundle, error)) {
		out = append(out, scenario{name: name, build: build})
	}

	// Category 1: payload empty / large / all-bytes, for every type.
	payloads := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"large", bytes.Repeat([]byte{0xAB}, 4096)},
		{"allbytes", allBytesPayload()},
	}
	for _, pv := range payloads {
		for _, ty := range allTypes {
			for _, outer := range both {
				pv, ty, outer := pv, ty, outer
				name := fmt.Sprintf("payload_%s_%s_%s", pv.name, ty, locName(outer))
				add(name, func() (*testgen.Bundle, error) {
					b := testgen.NewBuilder(baseCfg(name))
					addType(b, ty, outer, fieldsB(pv.data), testgen.AccountContent{}, false, nil, 0)
					return b.Build()
				})
			}
		}
	}

	// Category 2: wrong owner of the fetched account — regular paths only.
	badOwner := key(0x99)
	for _, ty := range []string{"postmessage", "postmessageunreliable"} {
		for _, outer := range both {
			ty, outer := ty, outer
			name := fmt.Sprintf("wrongowner_%s_%s", ty, locName(outer))
			add(name, func() (*testgen.Bundle, error) {
				b := testgen.NewBuilder(baseCfg(name))
				addType(b, ty, outer, fieldsB([]byte("owner")), testgen.AccountContent{Owner: &badOwner}, false, nil, 0)
				return b.Build()
			})
		}
	}

	// Category 3: bad account prefix — regular paths and close (all carry an account blob).
	for _, ty := range []string{"postmessage", "postmessageunreliable", "close"} {
		for _, outer := range both {
			ty, outer := ty, outer
			name := fmt.Sprintf("badprefix_%s_%s", ty, locName(outer))
			add(name, func() (*testgen.Bundle, error) {
				b := testgen.NewBuilder(baseCfg(name))
				addType(b, ty, outer, fieldsB([]byte("prefix")), testgen.AccountContent{Prefix: "bad"}, false, nil, 0)
				return b.Build()
			})
		}
	}

	// Category 4: wrong CPI program.
	// Regular + close: the core-emitted instruction under a non-core program.
	for _, ty := range []string{"postmessage", "postmessageunreliable", "close"} {
		for _, outer := range both {
			ty, outer := ty, outer
			name := fmt.Sprintf("wrongcpi_%s_%s", ty, locName(outer))
			add(name, func() (*testgen.Bundle, error) {
				b := testgen.NewBuilder(baseCfg(name))
				addType(b, ty, outer, fieldsB([]byte("cpi")), testgen.AccountContent{}, true, nil, 0)
				return b.Build()
			})
		}
	}
	// Shim: BOTH program checks matter — the core post (core-bridge program) and the
	// shim MessageEvent (shim-contract program). Generate a case for each.
	corePart, eventPart := testgen.CorePostPart, testgen.ShimEventPart
	shimWrong := []struct {
		tag  string
		part *testgen.ShimPart
	}{
		{"core", &corePart},   // core post_message under a non-core program: core-bridge check
		{"event", &eventPart}, // shim MessageEvent under a non-shim program: shim-contract check
	}
	for _, sw := range shimWrong {
		for _, outer := range both {
			sw, outer := sw, outer
			name := fmt.Sprintf("wrongcpi_shim_%s_%s", sw.tag, locName(outer))
			add(name, func() (*testgen.Bundle, error) {
				b := testgen.NewBuilder(baseCfg(name))
				addType(b, "shim", outer, fieldsB([]byte("cpi")), testgen.AccountContent{}, false, sw.part, 0)
				return b.Build()
			})
		}
	}

	// Category 5: large amount of accounts — many filler keys + a large instruction Accounts[].
	for _, ty := range allTypes {
		for _, outer := range both {
			ty, outer := ty, outer
			name := fmt.Sprintf("largeaccounts_%s_%s", ty, locName(outer))
			add(name, func() (*testgen.Bundle, error) {
				b := testgen.NewBuilder(cfgN(name, 128))
				addType(b, ty, outer, fieldsB([]byte("large-accts")), testgen.AccountContent{}, false, nil, 128)
				return b.Build()
			})
		}
	}

	// Category 6: accounts at non-default indices — a handful of leading fillers shift
	// core/shim/message off their usual positions.
	for _, ty := range allTypes {
		for _, outer := range both {
			ty, outer := ty, outer
			name := fmt.Sprintf("diffindex_%s_%s", ty, locName(outer))
			add(name, func() (*testgen.Bundle, error) {
				b := testgen.NewBuilder(cfgN(name, 5))
				addType(b, ty, outer, fieldsB([]byte("diff-idx")), testgen.AccountContent{}, false, nil, 0)
				return b.Build()
			})
		}
	}

	return out
}

// tripletVariant is one (message type, location) building block for triplets.
type tripletVariant struct {
	code  string // short filename token
	ty    string // addType type name
	outer bool
}

// tripletVariants enumerates the 8 building blocks: each of the four message types in
// both an outer (top-level) and inner (CPI) position.
func tripletVariants() []tripletVariant {
	return []tripletVariant{
		{"pmo", "postmessage", true},
		{"pmi", "postmessage", false},
		{"pmuo", "postmessageunreliable", true},
		{"pmui", "postmessageunreliable", false},
		{"sho", "shim", true},
		{"shi", "shim", false},
		{"clo", "close", true},
		{"cli", "close", false},
	}
}

// tripletCombos returns the first `limit` distinct 3-combinations of the 8 variants in
// lexicographic order. C(8,3) = 56, so up to 56 are available.
func tripletCombos(limit int) [][3]int {
	var combos [][3]int
	for i := 0; i < 8; i++ {
		for j := i + 1; j < 8; j++ {
			for k := j + 1; k < 8; k++ {
				if len(combos) == limit {
					return combos
				}
				combos = append(combos, [3]int{i, j, k})
			}
		}
	}
	return combos
}

// tripletScenarios builds 40 bundles, each executing a triplet of three distinct
// (type, location) variants in a single transaction. Each of the three messages gets
// distinct nonce / sequence / payload so their digests are all different, and every
// message is otherwise valid, so each triplet publishes exactly three messages.
func tripletScenarios() []scenario {
	variants := tripletVariants()
	combos := tripletCombos(40)

	var out []scenario
	for idx, combo := range combos {
		idx, combo := idx, combo
		v0, v1, v2 := variants[combo[0]], variants[combo[1]], variants[combo[2]]
		name := fmt.Sprintf("triplet_%02d_%s_%s_%s", idx, v0.code, v1.code, v2.code)
		out = append(out, scenario{name: name, build: func() (*testgen.Bundle, error) {
			b := testgen.NewBuilder(baseCfg(name))
			base := idx * 10
			for p, v := range []tripletVariant{v0, v1, v2} {
				m := fields(uint32(base+p+1), uint64(base+p+1), fmt.Sprintf("t%d-%d", idx, p))
				addType(b, v.ty, v.outer, m, testgen.AccountContent{}, false, nil, 0)
			}
			return b.Build()
		}})
	}
	return out
}

// txStatusScenarios generates the failed-transaction matrix: each message type in an
// outer and inner position, but with the transaction marked as failed on-chain
// (meta.Err set). validateTransactionMeta rejects the transaction up front, so every
// case must publish nothing. 4 types x 2 locations = 8 bundles.
func txStatusScenarios() []scenario {
	allTypes := []string{"postmessage", "postmessageunreliable", "shim", "close"}

	var out []scenario
	for _, ty := range allTypes {
		for _, outer := range []bool{true, false} {
			ty, outer := ty, outer
			name := fmt.Sprintf("txfailed_%s_%s", ty, locName(outer))
			out = append(out, scenario{name: name, build: func() (*testgen.Bundle, error) {
				cfg := baseCfg(name)
				cfg.TxFailed = true
				b := testgen.NewBuilder(cfg)
				addType(b, ty, outer, fieldsB([]byte("tx-failed")), testgen.AccountContent{}, false, nil, 0)
				return b.Build()
			}})
		}
	}
	return out
}

// altMovableCount is the number of referenced movable accounts an account-indexing message
// contributes in ALT mode: the message account plus the (NumAccounts-2) filler references
// accountRefs adds (post_message defaults to 8 accounts, close to 6).
func altMovableCount(ty string) int {
	if ty == "close" {
		return 5 // 6 accounts: signer + msg + 4 fillers
	}
	return 7 // 8 accounts: signer + msg + 6 fillers
}

// altLayout returns the LookupTables spec and message-account placement for a named layout,
// sized so Σ(Writable+Readonly) == n (the referenced movable accounts).
func altLayout(layout string, n int) ([]testgen.LookupTableSpec, int, bool) {
	switch layout {
	case "1table_writable":
		// One table, message + all fillers writable.
		return []testgen.LookupTableSpec{{Writable: n}}, 0, false
	case "1table_readonly":
		// One table; the message account is reached through the readonly section.
		return []testgen.LookupTableSpec{{Writable: n - 1, Readonly: 1}}, 0, true
	default: // "2table_mixed"
		// Two tables, each with writable and readonly entries; the message account lives in
		// the second table's readonly section — the most ordering-sensitive placement.
		x := 2 // table 1 writable count
		remaining := n - 1 - x
		w0 := (remaining + 1) / 2
		r0 := remaining - w0
		return []testgen.LookupTableSpec{{Writable: w0, Readonly: r0}, {Writable: x, Readonly: 1}}, 1, true
	}
}

// lookuptableScenarios generates the Address-Lookup-Table matrix: each account-indexing
// message type (regular reliable/unreliable and close), at an outer and inner position,
// under three lookup-table layouts (one writable table, one table with the message reached
// via readonly, and two tables mixing writable/readonly with the message in table 1's
// readonly section). Each is a versioned (v0) transaction whose referenced accounts are
// distributed across the synthesized ALTs. 3 layouts x 3 types x 2 locations = 18 bundles.
func lookuptableScenarios() []scenario {
	layouts := []string{"1table_writable", "1table_readonly", "2table_mixed"}
	types := []string{"postmessage", "postmessageunreliable", "close"}

	var out []scenario
	i := 0
	for _, layout := range layouts {
		for _, ty := range types {
			for _, outer := range []bool{true, false} {
				layout, ty, outer := layout, ty, outer
				name := fmt.Sprintf("lookuptable_%s_%s_%s", layout, ty, locName(outer))
				m := fields(uint32(200+i), uint64(200+i), fmt.Sprintf("alt-%d", i))
				out = append(out, scenario{name: name, build: func() (*testgen.Bundle, error) {
					cfg := baseCfg(name)
					specs, msgTable, msgReadonly := altLayout(layout, altMovableCount(ty))
					cfg.LookupTables = specs
					cfg.LookupMessageTable = msgTable
					cfg.LookupMessageReadonly = msgReadonly
					b := testgen.NewBuilder(cfg)
					addType(b, ty, outer, m, testgen.AccountContent{}, false, nil, 0)
					return b.Build()
				}})
				i++
			}
		}
	}
	return out
}

func main() {
	out := flag.String("out", "./bundles.json", "output JSON file for the generated bundle array")
	matrix := flag.String("matrix", "curated", "scenario set to emit: curated | boundary | situations | triplets | txstatus | lookuptable | all")
	flag.Parse()

	if dir := filepath.Dir(*out); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			fmt.Fprintln(os.Stderr, "failed to create output dir:", err)
			os.Exit(1)
		}
	}

	var scns []scenario
	switch *matrix {
	case "curated":
		scns = scenarios()
	case "boundary":
		scns = boundaryScenarios()
	case "situations":
		scns = situationScenarios()
	case "triplets":
		scns = tripletScenarios()
	case "txstatus":
		scns = txStatusScenarios()
	case "lookuptable":
		scns = lookuptableScenarios()
	case "all":
		scns = append(scenarios(), boundaryScenarios()...)
		scns = append(scns, situationScenarios()...)
		scns = append(scns, tripletScenarios()...)
		scns = append(scns, txStatusScenarios()...)
		scns = append(scns, lookuptableScenarios()...)
	default:
		fmt.Fprintf(os.Stderr, "unknown --matrix %q (want curated | boundary | situations | triplets | txstatus | lookuptable | all)\n", *matrix)
		os.Exit(2)
	}

	// Collect every scenario into a single array so the whole matrix lives in one file.
	bundles := make([]*testgen.Bundle, 0, len(scns))
	for _, s := range scns {
		bundle, err := s.build()
		if err != nil {
			fmt.Fprintf(os.Stderr, "scenario %s failed: %v\n", s.name, err)
			os.Exit(1)
		}
		bundles = append(bundles, bundle)
		fmt.Printf("built %s (%d top-level, %d inner-sets, %d accounts)\n",
			s.name, len(bundle.Transaction.Message.Instructions), len(bundle.Meta.InnerInstructions), len(bundle.Accounts))
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	if err := enc.Encode(bundles); err != nil {
		fmt.Fprintf(os.Stderr, "marshal failed: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil { //nolint:gosec // test fixtures
		fmt.Fprintf(os.Stderr, "write failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("generated %d bundles in %s\n", len(bundles), *out)
}
