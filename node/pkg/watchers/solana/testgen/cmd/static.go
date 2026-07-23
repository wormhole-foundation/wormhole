package main

import (
	"bytes"
	"fmt"

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
		// --- Post-message (account-based), outer vs inner, reliable vs unreliable ---
		{"pm_reliable_outer", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("pm_reliable_outer")).
				AddPostMessage(testgen.PostMessageSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(1, 1, "hello")}).Build()
		}},
		{"pm_unreliable_outer", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("pm_unreliable_outer")).
				AddPostMessage(testgen.PostMessageSpec{Location: testgen.Outer, Kind: testgen.PostMessageUnreliable, Msg: fields(2, 2, "payload")}).Build()
		}},
		{"pm_reliable_inner", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("pm_reliable_inner")).
				AddPostMessage(testgen.PostMessageSpec{Location: testgen.Inner, Kind: testgen.PostMessage, Msg: fields(3, 3, "cpi-message")}).Build()
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
		{"pm_reliable_wrong_owner", func() (*testgen.Bundle, error) {
			badOwner := key(0x99)
			return testgen.NewBuilder(baseCfg("pm_reliable_wrong_owner")).
				AddPostMessage(testgen.PostMessageSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(9, 9, "hi"), Account: testgen.AccountContent{Owner: &badOwner}}).Build()
		}},
		{"pm_reliable_not_finalized", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("pm_reliable_not_finalized")).
				AddPostMessage(testgen.PostMessageSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(10, 10, "hi"), Account: testgen.AccountContent{MessageStatus: 1}}).Build()
		}},
		{"pm_reliable_bad_prefix", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("pm_reliable_bad_prefix")).
				AddPostMessage(testgen.PostMessageSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(11, 11, "hi"), Account: testgen.AccountContent{Prefix: "bad"}}).Build()
		}},

		// --- Mixed: several types/locations in one transaction ---
		{"mixed_multi", func() (*testgen.Bundle, error) {
			return testgen.NewBuilder(baseCfg("mixed_multi")).
				AddPostMessage(testgen.PostMessageSpec{Location: testgen.Outer, Kind: testgen.PostMessage, Msg: fields(20, 20, "a")}).
				AddPostMessage(testgen.PostMessageSpec{Location: testgen.Inner, Kind: testgen.PostMessageUnreliable, Msg: fields(21, 21, "b")}).
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

// boundaryScenarios sweeps the hashed VAA-body fields at their edges in various message types
func dataTypeBoundaryScenarios() []scenario {
	types := []struct {
		name string
		add  func(b *testgen.Builder, outer bool, m testgen.WormholeFields)
	}{
		{"postmessage", func(b *testgen.Builder, outer bool, m testgen.WormholeFields) {
			b.AddPostMessage(testgen.PostMessageSpec{Location: locOf(outer), Kind: testgen.PostMessage, Msg: m})
		}},
		{"postmessageunreliable", func(b *testgen.Builder, outer bool, m testgen.WormholeFields) {
			b.AddPostMessage(testgen.PostMessageSpec{Location: locOf(outer), Kind: testgen.PostMessageUnreliable, Msg: m})
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
	for _, ty := range types { // Message type (postmessage, shim, etc.)
		for _, outer := range []bool{true, false} { // Top level or inner message
			for _, field := range fields { // The portion of the message to fuzz
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

	// Consistency: a Confirmed message (baseline is Finalized) should be rejected.
	for _, ty := range types {
		for _, outer := range []bool{true, false} {
			ty, outer := ty, outer
			name := fmt.Sprintf("%s_%s_consistency_confirmed", ty.name, locName(outer))
			out = append(out, scenario{name: name, build: func() (*testgen.Bundle, error) {
				m := baseline()
				m.Commitment = testgen.Confirmed
				b := testgen.NewBuilder(baseCfg(name))
				ty.add(b, outer, m)
				return b.Build()
			}})
		}
	}

	// Emitter address: 32 hashed bytes held constant everywhere else. Sweep the extremes
	// (all-zero, all-max, high-bit-set) to pin the digest across the full byte range.
	emitters := []struct {
		name string
		addr [32]byte
	}{
		{"zero", [32]byte{}},
		{"max", fillAddr(0xFF)},
		{"highbit", fillAddr(0x80)},
	}
	for _, ev := range emitters {
		for _, ty := range types {
			for _, outer := range []bool{true, false} {
				ev, ty, outer := ev, ty, outer
				name := fmt.Sprintf("%s_%s_emitter_%s", ty.name, locName(outer), ev.name)
				out = append(out, scenario{name: name, build: func() (*testgen.Bundle, error) {
					m := baseline()
					m.EmitterAddress = ev.addr
					b := testgen.NewBuilder(baseCfg(name))
					ty.add(b, outer, m)
					return b.Build()
				}})
			}
		}
	}

	// The published EmitterChain must always be the watcher's own chain id, never a value claimed by the message.
	for _, ty := range []string{"postmessage", "postmessageunreliable", "close"} {
		for _, outer := range []bool{true, false} {
			for _, ec := range []struct {
				name  string
				chain uint16
			}{
				{"ethereum", uint16(vaa.ChainIDEthereum)},
				{"max", 0xFFFF},
			} {
				ty, outer, ec := ty, outer, ec
				name := fmt.Sprintf("%s_%s_emitterchain_%s", ty, locName(outer), ec.name)
				out = append(out, scenario{name: name, build: func() (*testgen.Bundle, error) {
					m := baseline()
					m.EmitterChain = ec.chain
					b := testgen.NewBuilder(baseCfg(name))
					addType(b, ty, outer, m, testgen.AccountContent{}, false, nil, 0)
					return b.Build()
				}})
			}
		}
	}
	// Payload edges: empty / large / all-byte-values.
	payloads := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"large", bytes.Repeat([]byte{0xAB}, 4096)},
		{"allbytes", allBytesPayload()},
	}
	for _, pv := range payloads {
		for _, ty := range types {
			for _, outer := range []bool{true, false} {
				pv, ty, outer := pv, ty, outer
				name := fmt.Sprintf("payload_%s_%s_%s", pv.name, ty.name, locName(outer))
				out = append(out, scenario{name: name, build: func() (*testgen.Bundle, error) {
					m := baseline()
					m.Payload = pv.data
					b := testgen.NewBuilder(baseCfg(name))
					ty.add(b, outer, m)
					return b.Build()
				}})
			}
		}
	}
	return out
}

// fillAddr returns a 32-byte emitter address with every byte set to v.
func fillAddr(v byte) [32]byte {
	var a [32]byte
	for i := range a {
		a[i] = v
	}
	return a
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

// addType adds one message of the named type to b.
// numAccounts and wrongCore apply to the account-based paths (post_message/close);
// wrongShim targets a shim CPI part; both are ignored where not applicable.
func addType(b *testgen.Builder, typeName string, outer bool, m testgen.WormholeFields, ac testgen.AccountContent, wrongCore bool, wrongShim *testgen.ShimPart, numAccounts int) {
	switch typeName {
	case "postmessage":
		b.AddPostMessage(testgen.PostMessageSpec{Location: locOf(outer), Kind: testgen.PostMessage, Msg: m, Account: ac, NumAccounts: numAccounts, WrongCoreProgram: wrongCore})
	case "postmessageunreliable":
		b.AddPostMessage(testgen.PostMessageSpec{Location: locOf(outer), Kind: testgen.PostMessageUnreliable, Msg: m, Account: ac, NumAccounts: numAccounts, WrongCoreProgram: wrongCore})
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

// Program, account, and data validation
func dataValidationScenarios() []scenario {
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

	// Wrong owner of the fetched account — post_message paths only.
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

	// Bad account prefix — post_message paths and close (all carry an account blob).
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

	// Wrong CPI program.
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
	// The core post (core-bridge program) and the shim MessageEvent (shim-contract program) are checked.
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

	// Large amount of accounts — many filler keys + a large instruction Accounts[].
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

	// Accounts at non-default indices
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

// multiMsgVariant is one (message type, location) building block for multi-message bundles.
type multiMsgVariant struct {
	code  string // short filename token
	ty    string // addType type name
	outer bool
}

// multiMsgVariants enumerates the 8 building blocks: each of the four message types in
// both an outer (top-level) and inner (CPI) position.
func multiMsgVariants() []multiMsgVariant {
	return []multiMsgVariant{
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

// multiMsgCombos returns the first `limit` distinct 3-combinations of the 8 variants in
// lexicographic order. C(8,3) = 56, so up to 56 are available.
func multiMsgCombos(limit int) [][3]int {
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

// multiMsgScenarios builds 40 bundles, each executing three distinct (type, location)
// variants in a single transaction. Each of the three messages gets distinct nonce /
// sequence / payload so their digests are all different, and every message is otherwise
// valid, so each bundle publishes exactly three messages.
func multiMsgScenarios() []scenario {
	variants := multiMsgVariants()
	combos := multiMsgCombos(40)

	var out []scenario
	for idx, combo := range combos {
		idx, combo := idx, combo
		v0, v1, v2 := variants[combo[0]], variants[combo[1]], variants[combo[2]]
		name := fmt.Sprintf("multi_msg_%02d_%s_%s_%s", idx, v0.code, v1.code, v2.code)
		out = append(out, scenario{name: name, build: func() (*testgen.Bundle, error) {
			b := testgen.NewBuilder(baseCfg(name))
			base := idx * 10
			for p, v := range []multiMsgVariant{v0, v1, v2} {
				m := fields(uint32(base+p+1), uint64(base+p+1), fmt.Sprintf("m%d-%d", idx, p))
				addType(b, v.ty, v.outer, m, testgen.AccountContent{}, false, nil, 0)
			}
			return b.Build()
		}})
	}
	return out
}

// txStatusScenarios generates the failed tx scenarios
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

// allScenarios returns every bundle scenario across all matrices.
func allScenarios() []scenario {
	var scns []scenario
	scns = append(scns, scenarios()...)
	scns = append(scns, dataTypeBoundaryScenarios()...)
	scns = append(scns, dataValidationScenarios()...)
	scns = append(scns, multiMsgScenarios()...)
	scns = append(scns, txStatusScenarios()...)
	return scns
}

// buildStaticBundles builds the full synthetic matrix.
func buildStaticBundles() ([]*testgen.Bundle, error) {
	scns := allScenarios()
	bundles := make([]*testgen.Bundle, 0, len(scns))
	for _, s := range scns {
		bundle, err := s.build()
		if err != nil {
			return nil, fmt.Errorf("scenario %s: %w", s.name, err)
		}
		bundles = append(bundles, bundle)
	}
	return bundles, nil
}
