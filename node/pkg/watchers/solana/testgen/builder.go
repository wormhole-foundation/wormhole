// Package testgen constructs realistic bundles for the Solana watcher's observation
// and reobservation replay paths.
//
// The builder owns all account-key indexing — the part that is tedious and
// error-prone to do by hand — so callers describe messages in high-level terms and
// get back a Bundle containing a correctly indexed transaction.
package testgen

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"

	solwatch "github.com/certusone/wormhole/node/pkg/watchers/solana"
	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/near/borsh-go"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

// Instruction ids, prefixes and discriminators.
const (
	postMessageInstructionID           = 0x01
	postMessageUnreliableInstructionID = 0x08
	closePostedMessageInstructionID    = 0x09

	postMessageMinNumAccounts = 8 // client.go: postMessageInstructionMinNumAccounts
	closeMinNumAccounts       = 6 // close_event.go: closePostedMessageMinNumAccounts
	messageAccountRefIndex    = 1
	minAccountRefs            = messageAccountRefIndex + 1
	finalizedAccountLevel     = 32

	prefixReliable   = "msg" // client.go: accountPrefixReliable
	prefixUnreliable = "msu" // client.go: accountPrefixUnreliable

	shimPostMessageDiscriminatorHex  = "d63264d12622074c"                 // shim.go
	shimMessageEventDiscriminatorHex = "e445a52e51cb9a1d441b8f004d4c8970" // shim.go
	closeEventTagHex                 = "e445a52e51cb9a1d"                 // close_event.go: EVENT_IX_TAG_LE
	closeEventDiscriminatorHex       = "9ef61bc2241428b9"                 // close_event.go: MessageAccountClosed
)

// Account-key role tags used to derive deterministic, collision-free pubkeys for
// accounts the caller does not pin explicitly.
const (
	fillerTag       byte = 0xEE
	messageTag      byte = 0xB0
	wrapperTag      byte = 0xC0
	wrongProgramTag byte = 0x77 // a non-core, non-shim program used for WrongCoreProgram cases
)

// Location identifies a top-level or inner instruction.
type Location int

const (
	Outer Location = iota
	Inner
)

// MessageKind selects the reliable (PostMessage, 0x01/"msg") vs unreliable
// (PostMessageUnreliable, 0x08/"msu") variant for the post_message path.
type MessageKind int

const (
	PostMessage MessageKind = iota
	PostMessageUnreliable
)

// Commitment is the logical commitment for a message.
type Commitment int

const (
	CommitmentDefault Commitment = iota
	Confirmed
	Finalized
)

// ShimTopology selects the shim publishing topology
type ShimTopology int

const (
	Direct ShimTopology = iota
	Integrator
)

// ShimPart identifies one of the three instructions in a shim publication.
type ShimPart int

const (
	ShimPostPart ShimPart = iota
	CorePostPart
	ShimEventPart
)

// DefaultShimOrder is the valid arrangement: shim post_message, then the core
// post_message, then the shim MessageEvent.
var DefaultShimOrder = []ShimPart{ShimPostPart, CorePostPart, ShimEventPart}

// WormholeFields are the message fields shared between an instruction and its account.
type WormholeFields struct {
	Nonce          uint32
	Payload        []byte
	Commitment     Commitment
	Sequence       uint64
	EmitterChain   uint16
	EmitterAddress [32]byte
	Timestamp      uint32 // shim MessageEvent timestamp and account SubmissionTime
}

// AccountContent describes the generated message account data.
type AccountContent struct {
	Prefix           string            // overrides "msg"/"msu"; empty => derived from Kind
	Owner            *solana.PublicKey // overrides owner; nil => core contract
	VaaVersion       uint8
	RawConsistency   *uint8 // overrides the account consistency byte; nil => derived from Commitment
	EmitterAuthority [32]byte
	MessageStatus    uint8
	Gap              [3]byte
	SubmissionTime   *uint32 // overrides; nil => WormholeFields.Timestamp
}

// PostMessageSpec describes an account-based post_message. The message account is
// fetched by RPC, so its content is delivered via the bundle's Accounts list.
type PostMessageSpec struct {
	Name              string
	Location          Location
	Kind              MessageKind
	Msg               WormholeFields
	Account           AccountContent
	NumAccounts       int               // instruction account count (>=8); 0 => 8
	MessageAccountKey *solana.PublicKey // pin the message account pubkey; nil => auto
	// WrongCoreProgram emits the post_message instruction under a non-core program id,
	// so the watcher does not recognize it as a Wormhole instruction.
	WrongCoreProgram bool
}

// ShimSpec describes a shim publication (three instructions).
type ShimSpec struct {
	Name             string
	Topology         ShimTopology
	Ordering         []ShimPart // nil => DefaultShimOrder
	Msg              WormholeFields
	WrongProgramPart *ShimPart
}

// CloseSpec describes a close_posted_message event (processed only during
// reobservation). The account content is embedded inline in the CPI event, not fetched.
type CloseSpec struct {
	Name              string
	Location          Location
	Msg               WormholeFields
	Account           AccountContent
	NumAccounts       int               // close instruction account count (>=6); 0 => 6
	MessageAccountKey *solana.PublicKey // pin the message account pubkey; nil => auto
	// WrongCoreProgram emits the CPI event under a non-core program id
	WrongCoreProgram bool
}

// AccountInfoResponse is one getAccountInfo result.
type AccountInfoResponse struct {
	Pubkey     solana.PublicKey
	Owner      solana.PublicKey
	Lamports   uint64
	Data       []byte
	Executable bool
	RentEpoch  uint64
}

// MarshalJSON emits the RPC getAccountInfo value shape plus the pubkey.
func (a AccountInfoResponse) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"pubkey":     a.Pubkey,
		"owner":      a.Owner,
		"lamports":   a.Lamports,
		"data":       []interface{}{base64.StdEncoding.EncodeToString(a.Data), "base64"},
		"executable": a.Executable,
		"rentEpoch":  a.RentEpoch,
	})
}

// Bundle is the generated input for the watcher replay paths.
type Bundle struct {
	Name         string                `json:"name"`
	Slot         uint64                `json:"slot"`
	Contract     solana.PublicKey      `json:"contract"`
	ShimContract solana.PublicKey      `json:"shimContract"`
	Transaction  *solana.Transaction   `json:"transaction"`
	Meta         *rpc.TransactionMeta  `json:"meta"`
	Accounts     []AccountInfoResponse `json:"accounts"`
	// Signature is the source transaction signature, set only for live-collected bundles.
	Signature string `json:"signature,omitempty"`
}

const (
	StaticBundlesFilename = "static_bundles.json"
	LiveBundlesFilename   = "live_bundles.json"
)

// Config configures a Builder.
type Config struct {
	Contract              solana.PublicKey
	ShimContract          solana.PublicKey
	WatcherCommitment     rpc.CommitmentType // defaults to finalized if empty
	Slot                  uint64
	RecentBlockhash       solana.Hash
	Name                  string
	LeadingFillerAccounts int
	TxFailed              bool
}

// Builder accumulates instructions, inner-instruction sets, and account responses,
// assigning account-key indices as it goes.
type Builder struct {
	cfg      Config
	keys     []solana.PublicKey
	keyIndex map[solana.PublicKey]int
	topLevel []solana.CompiledInstruction
	inner    []rpc.InnerInstruction
	accounts []AccountInfoResponse

	shimEnabled bool
	msgCounter  int
	err         error
}

// NewBuilder creates a Builder. Index 0 is reserved for a filler account so the core
// contract is never at index 0 (index 0 collides with noise instructions, matching
// the TestProcessTransaction convention).
func NewBuilder(cfg Config) *Builder {
	if cfg.WatcherCommitment == "" {
		cfg.WatcherCommitment = rpc.CommitmentFinalized
	}
	b := &Builder{
		cfg:         cfg,
		keyIndex:    map[solana.PublicKey]int{},
		shimEnabled: !cfg.ShimContract.IsZero(),
	}
	b.ensureKey(genKey(fillerTag, 0)) // index 0: filler
	for i := 1; i <= cfg.LeadingFillerAccounts; i++ {
		b.ensureKey(genKey(fillerTag, i)) // extra leading fillers shift real indices
	}
	b.ensureKey(cfg.Contract) // core contract (index >=1, further shifted by fillers)
	if b.shimEnabled {
		b.ensureKey(cfg.ShimContract)
	}
	return b
}

func (b *Builder) wrongProgramIndex() uint16 {
	return b.ensureKey(genKey(wrongProgramTag, 0))
}

// Err returns the first error encountered while building, if any.
func (b *Builder) Err() error { return b.err }

func (b *Builder) fail(err error) {
	if b.err == nil && err != nil {
		b.err = err
	}
}

// genKey derives a deterministic 32-byte pubkey for a role tag and counter.
func genKey(tag byte, n int) solana.PublicKey {
	var raw [32]byte
	for i := range raw {
		raw[i] = tag
	}
	raw[31] = byte(n) // #nosec G115 -- generated test keys use tiny per-role counters
	return solana.PublicKeyFromBytes(raw[:])
}

func (b *Builder) ensureKey(pk solana.PublicKey) uint16 {
	if idx, ok := b.keyIndex[pk]; ok {
		return uint16(idx) // #nosec G115 -- account count is tiny in generated inputs
	}
	idx := len(b.keys)
	b.keys = append(b.keys, pk)
	b.keyIndex[pk] = idx
	return uint16(idx) // #nosec G115
}

func (b *Builder) coreIndex() uint16 { return uint16(b.keyIndex[b.cfg.Contract]) }     // #nosec G115
func (b *Builder) shimIndex() uint16 { return uint16(b.keyIndex[b.cfg.ShimContract]) } // #nosec G115

// addWrapperTopLevel appends a wrapper top-level instruction
func (b *Builder) addWrapperTopLevel() uint16 {
	prog := b.ensureKey(genKey(wrapperTag, len(b.topLevel)))
	idx := uint16(len(b.topLevel)) // #nosec G115
	b.topLevel = append(b.topLevel, solana.CompiledInstruction{
		ProgramIDIndex: prog,
		Data:           solana.Base58{0xAB, 0xCD}, // benign noise
	})
	return idx
}

func (b *Builder) addInnerSet(index uint16, insts []solana.CompiledInstruction) {
	b.inner = append(b.inner, rpc.InnerInstruction{Index: index, Instructions: insts})
}

func (b *Builder) accountRefs(minLen int, msgIdx uint16) []uint16 {
	if minLen < minAccountRefs {
		minLen = minAccountRefs
	}
	a := make([]uint16, minLen)
	a[messageAccountRefIndex] = msgIdx
	return a
}

func (b *Builder) resolveCommitment(c Commitment) Commitment {
	if c == CommitmentDefault {
		if b.cfg.WatcherCommitment == rpc.CommitmentConfirmed {
			return Confirmed
		}
		return Finalized
	}
	return c
}

// instrConsistency maps a resolved commitment to the instruction-data scale
// (0=confirmed, 1=finalized), matching consistencyLevel{Confirmed,Finalized}.
func instrConsistency(c Commitment) solwatch.ConsistencyLevel {
	if c == Confirmed {
		return 0
	}
	return 1
}

// accountConsistency maps a resolved commitment to the account-byte scale
// (1=confirmed, 32=finalized), matching accountConsistencyLevelToCommitment.
func accountConsistency(c Commitment) uint8 {
	if c == Confirmed {
		return 1
	}
	return finalizedAccountLevel
}

func toAddr(b [32]byte) vaa.Address {
	var a vaa.Address
	copy(a[:], b[:])
	return a
}

func mustHex(s string) []byte {
	out, err := hex.DecodeString(s)
	if err != nil {
		panic("testgen: bad hex constant: " + err.Error())
	}
	return out
}

func (b *Builder) postMessageInstrData(kind MessageKind, m WormholeFields) []byte {
	id := byte(postMessageInstructionID)
	if kind == PostMessageUnreliable {
		id = postMessageUnreliableInstructionID
	}
	body, err := borsh.Serialize(solwatch.PostMessageData{
		Nonce:            m.Nonce,
		Payload:          m.Payload,
		ConsistencyLevel: instrConsistency(b.resolveCommitment(m.Commitment)),
	})
	b.fail(err)
	return append([]byte{id}, body...)
}

// accountBlob builds the full message-account bytes: prefix + borsh(MessagePublicationAccount).
func (b *Builder) accountBlob(kind MessageKind, m WormholeFields, ac AccountContent) []byte {
	prefix := ac.Prefix
	if prefix == "" {
		prefix = prefixReliable
		if kind == PostMessageUnreliable {
			prefix = prefixUnreliable
		}
	}
	cons := accountConsistency(b.resolveCommitment(m.Commitment))
	if ac.RawConsistency != nil {
		cons = *ac.RawConsistency
	}
	st := m.Timestamp
	if ac.SubmissionTime != nil {
		st = *ac.SubmissionTime
	}
	body, err := borsh.Serialize(solwatch.MessagePublicationAccount{
		VaaVersion:       ac.VaaVersion,
		ConsistencyLevel: cons,
		EmitterAuthority: toAddr(ac.EmitterAuthority),
		MessageStatus:    ac.MessageStatus,
		Gap:              ac.Gap,
		SubmissionTime:   st,
		Nonce:            m.Nonce,
		Sequence:         m.Sequence,
		EmitterChain:     m.EmitterChain,
		EmitterAddress:   toAddr(m.EmitterAddress),
		Payload:          m.Payload,
	})
	b.fail(err)
	return append([]byte(prefix), body...)
}

func (b *Builder) shimPostData(m WormholeFields) []byte {
	body, err := borsh.Serialize(solwatch.ShimPostMessageData{
		Nonce:            m.Nonce,
		ConsistencyLevel: instrConsistency(b.resolveCommitment(m.Commitment)),
		Payload:          m.Payload,
	})
	b.fail(err)
	return append(mustHex(shimPostMessageDiscriminatorHex), body...)
}

func (b *Builder) shimCoreData(m WormholeFields) []byte {
	// The core event accompanying a shim message is always unreliable with an empty payload.
	body, err := borsh.Serialize(solwatch.PostMessageData{
		Nonce:            m.Nonce,
		Payload:          []byte{},
		ConsistencyLevel: instrConsistency(b.resolveCommitment(m.Commitment)),
	})
	b.fail(err)
	return append([]byte{postMessageUnreliableInstructionID}, body...)
}

func (b *Builder) shimEventData(m WormholeFields) []byte {
	body, err := borsh.Serialize(solwatch.ShimMessageEventData{
		EmitterAddress: m.EmitterAddress,
		Sequence:       m.Sequence,
		Timestamp:      m.Timestamp,
	})
	b.fail(err)
	return append(mustHex(shimMessageEventDiscriminatorHex), body...)
}

func (b *Builder) closeEventData(kind MessageKind, m WormholeFields, ac AccountContent) []byte {
	out := append(mustHex(closeEventTagHex), mustHex(closeEventDiscriminatorHex)...)
	return append(out, b.accountBlob(kind, m, ac)...)
}

// AddPostMessage adds a post_message call.
//
//nolint:unparam // The returned builder is used by cross-package fluent fixture definitions.
func (b *Builder) AddPostMessage(spec PostMessageSpec) *Builder {
	msgKey := genKey(messageTag, b.msgCounter)
	if spec.MessageAccountKey != nil {
		msgKey = *spec.MessageAccountKey
	}
	b.msgCounter++
	msgIdx := b.ensureKey(msgKey)

	n := spec.NumAccounts
	if n < postMessageMinNumAccounts {
		n = postMessageMinNumAccounts
	}
	progIdx := b.coreIndex()
	if spec.WrongCoreProgram {
		progIdx = b.wrongProgramIndex()
	}
	inst := solana.CompiledInstruction{
		ProgramIDIndex: progIdx,
		Accounts:       b.accountRefs(n, msgIdx),
		Data:           solana.Base58(b.postMessageInstrData(spec.Kind, spec.Msg)),
	}
	if spec.Location == Outer {
		b.topLevel = append(b.topLevel, inst)
	} else {
		wrapperIdx := b.addWrapperTopLevel()
		b.addInnerSet(wrapperIdx, []solana.CompiledInstruction{inst})
	}

	owner := b.cfg.Contract
	if spec.Account.Owner != nil {
		owner = *spec.Account.Owner
	}
	blob := b.accountBlob(spec.Kind, spec.Msg, spec.Account)
	b.accounts = append(b.accounts, AccountInfoResponse{
		Pubkey:   msgKey,
		Owner:    owner,
		Lamports: 1,
		Data:     blob,
	})
	return b
}

// AddShim adds a shim publication
//
//nolint:unparam // The returned builder is used by cross-package fluent fixture definitions.
func (b *Builder) AddShim(spec ShimSpec) *Builder {
	if !b.shimEnabled {
		b.fail(fmt.Errorf("AddShim requires Config.ShimContract to be set"))
		return b
	}
	ordering := spec.Ordering
	if ordering == nil {
		ordering = DefaultShimOrder
	}

	build := func(p ShimPart) solana.CompiledInstruction {
		var progIdx uint16
		var data []byte
		switch p {
		case ShimPostPart:
			progIdx, data = b.shimIndex(), b.shimPostData(spec.Msg)
		case CorePostPart:
			progIdx, data = b.coreIndex(), b.shimCoreData(spec.Msg)
		case ShimEventPart:
			progIdx, data = b.shimIndex(), b.shimEventData(spec.Msg)
		}
		// Redirect this part to a wrong program to exercise the core-bridge
		// (CorePostPart) or shim-contract (ShimEventPart) program check.
		if spec.WrongProgramPart != nil && *spec.WrongProgramPart == p {
			progIdx = b.wrongProgramIndex()
		}
		return solana.CompiledInstruction{ProgramIDIndex: progIdx, Data: solana.Base58(data)}
	}

	if spec.Topology == Direct {
		// Shim post_message is top-level; the remaining parts form the matching inner set.
		b.topLevel = append(b.topLevel, build(ShimPostPart))
		topIdx := uint16(len(b.topLevel) - 1) // #nosec G115
		var innerParts []solana.CompiledInstruction
		for _, p := range ordering {
			if p == ShimPostPart {
				continue // the post is the top-level instruction in the direct case
			}
			innerParts = append(innerParts, build(p))
		}
		b.addInnerSet(topIdx, innerParts)
	} else {
		// Integrator: all parts live in a single inner set behind a wrapper program.
		wrapperIdx := b.addWrapperTopLevel()
		var innerParts []solana.CompiledInstruction
		for _, p := range ordering {
			innerParts = append(innerParts, build(p))
		}
		b.addInnerSet(wrapperIdx, innerParts)
	}
	return b
}

// AddClose adds a close_posted_message event.
//
//nolint:unparam // The returned builder is used by cross-package fluent fixture definitions.
func (b *Builder) AddClose(spec CloseSpec) *Builder {
	msgKey := genKey(messageTag, b.msgCounter)
	if spec.MessageAccountKey != nil {
		msgKey = *spec.MessageAccountKey
	}
	b.msgCounter++
	msgIdx := b.ensureKey(msgKey)

	n := spec.NumAccounts
	if n < closeMinNumAccounts {
		n = closeMinNumAccounts
	}
	closeInst := solana.CompiledInstruction{
		ProgramIDIndex: b.coreIndex(),
		Accounts:       b.accountRefs(n, msgIdx),
		Data:           solana.Base58{closePostedMessageInstructionID},
	}
	// Captured close events are "msg"-prefixed by default. AccountContent.Prefix can
	// override the prefix to exercise invalid or unreliable account data.
	eventProgIdx := b.coreIndex()
	if spec.WrongCoreProgram {
		eventProgIdx = b.wrongProgramIndex()
	}
	eventInst := solana.CompiledInstruction{
		ProgramIDIndex: eventProgIdx,
		Data:           solana.Base58(b.closeEventData(PostMessage, spec.Msg, spec.Account)),
	}

	if spec.Location == Outer {
		// Top-level close instruction; the CPI event is in the matching inner set.
		b.topLevel = append(b.topLevel, closeInst)
		topIdx := uint16(len(b.topLevel) - 1) // #nosec G115
		b.addInnerSet(topIdx, []solana.CompiledInstruction{eventInst})
	} else {
		// close_posted_message called via CPI: close instruction and its event are
		// sibling inner instructions behind a wrapper program.
		wrapperIdx := b.addWrapperTopLevel()
		b.addInnerSet(wrapperIdx, []solana.CompiledInstruction{closeInst, eventInst})
	}
	return b
}

// Build assembles the accumulated instructions and accounts into a Bundle for testing.
func (b *Builder) Build() (*Bundle, error) {
	if b.err != nil {
		return nil, b.err
	}

	// Deterministic non-zero signature (used by the watcher as the TxID).
	var sig solana.Signature
	for i := range sig {
		sig[i] = 0x11
	}

	tx := &solana.Transaction{
		Message: solana.Message{
			AccountKeys:     b.keys,
			Header:          solana.MessageHeader{NumRequiredSignatures: 1, NumReadonlySignedAccounts: 0, NumReadonlyUnsignedAccounts: 1},
			RecentBlockhash: b.cfg.RecentBlockhash,
			Instructions:    b.topLevel,
		},
		Signatures: []solana.Signature{sig},
	}
	meta := &rpc.TransactionMeta{
		InnerInstructions: b.inner,
	}
	if b.cfg.TxFailed {
		// A representative on-chain failure.
		meta.Err = map[string]interface{}{
			"InstructionError": []interface{}{0, map[string]interface{}{"Custom": 1}},
		}
	}

	name := b.cfg.Name
	if name == "" {
		name = "bundle"
	}
	return &Bundle{
		Name:         name,
		Slot:         b.cfg.Slot,
		Contract:     b.cfg.Contract,
		ShimContract: b.cfg.ShimContract,
		Transaction:  tx,
		Meta:         meta,
		Accounts:     b.accounts,
	}, nil
}
