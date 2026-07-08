package testgen

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	solwatch "github.com/certusone/wormhole/node/pkg/watchers/solana"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testBuilder() *Builder {
	return NewBuilder(Config{
		Contract:          genKey(0xAA, 0),
		ShimContract:      genKey(0xDD, 0),
		WatcherCommitment: rpc.CommitmentFinalized,
		Slot:              42,
	})
}

// emitter041c is the 32-byte emitter used in the watcher's own known-good shim vectors.
func emitter041c(t *testing.T) [32]byte {
	t.Helper()
	raw, err := hex.DecodeString("041c657e845d65d009d59ceeb1dda172bd6bc9e7ee5a19e56573197cf7fdffde")
	require.NoError(t, err)
	var out [32]byte
	copy(out[:], raw)
	return out
}

// TestKnownGoodVectors asserts the encoders reproduce the exact byte strings that
// the watcher's own tests hard-code (client_test.go:834-836), proving the builder's
// borsh encoding cannot drift from what the watcher deserializes.
func TestKnownGoodVectors(t *testing.T) {
	b := testBuilder()
	// A finalized, default-commitment message with nonce 42 and payload "hello world".
	msg := WormholeFields{
		Nonce:          42,
		Payload:        []byte("hello world"),
		Commitment:     Finalized,
		EmitterAddress: emitter041c(t),
		Timestamp:      0x67815b7c,
	}

	assert.Equal(t,
		"d63264d12622074c2a000000010b00000068656c6c6f20776f726c64",
		hex.EncodeToString(b.shimPostData(msg)),
		"shim post_message vector")

	assert.Equal(t,
		"082a0000000000000001",
		hex.EncodeToString(b.shimCoreData(msg)),
		"shim core post_message vector")

	assert.Equal(t,
		"e445a52e51cb9a1d441b8f004d4c8970041c657e845d65d009d59ceeb1dda172bd6bc9e7ee5a19e56573197cf7fdffde00000000000000007c5b8167",
		hex.EncodeToString(b.shimEventData(msg)),
		"shim MessageEvent vector")

	require.NoError(t, b.Err())
}

// TestRegularInstrDataVector checks the regular post_message instruction data matches
// the encodePostMessageData layout (id byte + borsh{nonce, payload, consistency}).
func TestRegularInstrDataVector(t *testing.T) {
	b := testBuilder()
	msg := WormholeFields{Nonce: 7, Payload: []byte("hello"), Commitment: Finalized}
	// 01 | nonce=07000000 | len=05000000 | "hello"=68656c6c6f | consistency=01
	got := hex.EncodeToString(b.regularInstrData(PostMessage, msg))
	assert.Equal(t, "010700000005000000"+hex.EncodeToString([]byte("hello"))+"01", got)

	gotUnrel := hex.EncodeToString(b.regularInstrData(PostMessageUnreliable, msg))
	assert.Equal(t, "080700000005000000"+hex.EncodeToString([]byte("hello"))+"01", gotUnrel)
	require.NoError(t, b.Err())
}

// TestAccountBlobRoundTrips builds a message-account blob and confirms it passes the
// watcher's NewMessageAccountData validation and parses back to the same fields.
func TestAccountBlobRoundTrips(t *testing.T) {
	b := testBuilder()
	msg := WormholeFields{
		Nonce:          7,
		Payload:        []byte("payload-bytes"),
		Commitment:     Finalized,
		Sequence:       99,
		EmitterChain:   1,
		EmitterAddress: emitter041c(t),
		Timestamp:      123456,
	}
	blob := b.accountBlob(PostMessage, msg, AccountContent{VaaVersion: 1})
	require.NoError(t, b.Err())

	mad, err := solwatch.NewMessageAccountData(blob)
	require.NoError(t, err)
	assert.True(t, mad.IsReliable())

	prop, err := solwatch.ParseMessagePublicationAccount(mad)
	require.NoError(t, err)
	assert.Equal(t, uint8(1), prop.VaaVersion)
	assert.Equal(t, uint8(32), prop.ConsistencyLevel) // finalized on the account scale
	assert.Equal(t, uint32(7), prop.Nonce)
	assert.Equal(t, uint64(99), prop.Sequence)
	assert.Equal(t, uint16(1), prop.EmitterChain)
	assert.Equal(t, uint32(123456), prop.SubmissionTime)
	assert.Equal(t, []byte("payload-bytes"), prop.Payload)

	// Unreliable variant uses the "msu" prefix.
	blobU := b.accountBlob(PostMessageUnreliable, msg, AccountContent{})
	require.NoError(t, b.Err())
	madU, err := solwatch.NewMessageAccountData(blobU)
	require.NoError(t, err)
	assert.False(t, madU.IsReliable())
}

// TestBundleJSONRoundTrip marshals a Bundle and confirms the transaction and meta
// decode straight back into solana.Transaction / rpc.TransactionMeta.
func TestBundleJSONRoundTrip(t *testing.T) {
	b := testBuilder()
	emitter := emitter041c(t)
	b.AddRegular(RegularSpec{
		Location: Outer,
		Kind:     PostMessage,
		Msg:      WormholeFields{Nonce: 7, Payload: []byte("hi"), Sequence: 1, EmitterChain: 1, EmitterAddress: emitter},
	})
	b.AddShim(ShimSpec{
		Topology: Integrator,
		Msg:      WormholeFields{Nonce: 8, Payload: []byte("world"), Sequence: 2, EmitterAddress: emitter, Timestamp: 5},
	})
	bundle, err := b.Build()
	require.NoError(t, err)

	raw, err := json.Marshal(bundle)
	require.NoError(t, err)

	var env struct {
		Transaction json.RawMessage `json:"transaction"`
		Meta        json.RawMessage `json:"meta"`
	}
	require.NoError(t, json.Unmarshal(raw, &env))

	var tx solana.Transaction
	require.NoError(t, json.Unmarshal(env.Transaction, &tx))
	assert.Equal(t, bundle.Transaction.Message.AccountKeys, tx.Message.AccountKeys)
	require.Equal(t, len(bundle.Transaction.Message.Instructions), len(tx.Message.Instructions))
	for i, want := range bundle.Transaction.Message.Instructions {
		got := tx.Message.Instructions[i]
		assert.Equal(t, want.ProgramIDIndex, got.ProgramIDIndex, "inst %d program", i)
		assert.Equal(t, want.Accounts, got.Accounts, "inst %d accounts", i)
		assert.Equal(t, []byte(want.Data), []byte(got.Data), "inst %d data", i)
	}
	assert.Equal(t, bundle.Transaction.Signatures, tx.Signatures)
	assert.False(t, tx.Message.IsVersioned(), "generated tx must be legacy")

	var meta rpc.TransactionMeta
	require.NoError(t, json.Unmarshal(env.Meta, &meta))
	require.Equal(t, len(bundle.Meta.InnerInstructions), len(meta.InnerInstructions))
	for i, want := range bundle.Meta.InnerInstructions {
		got := meta.InnerInstructions[i]
		assert.Equal(t, want.Index, got.Index, "inner set %d index", i)
		require.Equal(t, len(want.Instructions), len(got.Instructions), "inner set %d len", i)
		for j, wi := range want.Instructions {
			assert.Equal(t, []byte(wi.Data), []byte(got.Instructions[j].Data), "inner %d/%d data", i, j)
		}
	}
}

// TestIndexingIntegrity checks that every instruction's account/program indices and
// every inner-set Index stay within bounds — the core value the builder provides.
func TestIndexingIntegrity(t *testing.T) {
	b := testBuilder()
	b.AddRegular(RegularSpec{Location: Outer, Kind: PostMessage, Msg: WormholeFields{Nonce: 1}})
	b.AddRegular(RegularSpec{Location: Inner, Kind: PostMessageUnreliable, Msg: WormholeFields{Nonce: 2, Payload: []byte("x")}})
	b.AddShim(ShimSpec{Topology: Direct, Msg: WormholeFields{Nonce: 3}})
	b.AddShim(ShimSpec{Topology: Integrator, Msg: WormholeFields{Nonce: 4}})
	b.AddClose(CloseSpec{Location: Outer, Msg: WormholeFields{Nonce: 5, Payload: []byte("y")}})
	b.AddClose(CloseSpec{Location: Inner, Msg: WormholeFields{Nonce: 6, Payload: []byte("z")}})
	bundle, err := b.Build()
	require.NoError(t, err)

	nKeys := len(bundle.Transaction.Message.AccountKeys)
	assert.True(t, bundle.IsReobservation, "close events force reobservation")

	checkInst := func(inst solana.CompiledInstruction, ctx string) {
		assert.Less(t, int(inst.ProgramIDIndex), nKeys, "%s: program index in bounds", ctx)
		for _, a := range inst.Accounts {
			assert.Less(t, int(a), nKeys, "%s: account index in bounds", ctx)
		}
	}
	nTop := len(bundle.Transaction.Message.Instructions)
	for i, inst := range bundle.Transaction.Message.Instructions {
		checkInst(inst, "top-level")
		_ = i
	}
	for _, set := range bundle.Meta.InnerInstructions {
		assert.Less(t, int(set.Index), nTop, "inner set index references a real top-level instruction")
		for _, inst := range set.Instructions {
			checkInst(inst, "inner")
		}
	}

	// The core contract must never be at index 0.
	assert.NotEqual(t, uint16(0), b.coreIndex())
}

// TestShimCustomOrdering confirms the ordering knob actually reorders the emitted
// inner instructions (used to exercise the shimProcessRest ordering invariant).
func TestShimCustomOrdering(t *testing.T) {
	b := testBuilder()
	// Event before core — a deliberately invalid arrangement for the watcher.
	b.AddShim(ShimSpec{
		Topology: Integrator,
		Ordering: []ShimPart{ShimPostPart, ShimEventPart, CorePostPart},
		Msg:      WormholeFields{Nonce: 1},
	})
	bundle, err := b.Build()
	require.NoError(t, err)
	require.Len(t, bundle.Meta.InnerInstructions, 1)
	insts := bundle.Meta.InnerInstructions[0].Instructions
	require.Len(t, insts, 3)
	// Second instruction should be the shim MessageEvent (shim program, event discriminator).
	assert.Equal(t, b.shimIndex(), insts[1].ProgramIDIndex)
	assert.Equal(t, shimMessageEventDiscriminatorHex, hex.EncodeToString([]byte(insts[1].Data))[:len(shimMessageEventDiscriminatorHex)])
}
