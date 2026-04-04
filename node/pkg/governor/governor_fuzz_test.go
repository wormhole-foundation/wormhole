package governor

import (
	"encoding/binary"
	"flag"
	"fmt"
	"math/big"
	"sort"
	"sync"
	"testing"
	"time"

	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/certusone/wormhole/node/pkg/common"
	guardianDB "github.com/certusone/wormhole/node/pkg/db"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

var fuzzVerboseGov = flag.Bool("fuzz.verbose-gov", false, "enable governor internal logging during fuzz tests")

// =============================================================================
// Lookup tables
// =============================================================================

type fuzzChainConfig struct {
	chainID         vaa.ChainID
	tokenBridgeAddr string
	dailyLimit      uint64
	bigTxSize       uint64
}

type fuzzEmitterConfig struct {
	chainID     vaa.ChainID
	emitterAddr string
}

type fuzzTokenConfig struct {
	originChain vaa.ChainID
	addr        string
	symbol      string
	price       float64
	flowCancels bool
	governed    bool
}

// Chain table (4 entries) — governor config and target chain selection.
var fuzzChainTable = []fuzzChainConfig{
	{vaa.ChainIDEthereum, "0x0290fb167208af455bb137780163b7b7a9a10c16", 100_000, 50_000},
	{vaa.ChainIDSui, "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9", 50_000, 25_000},
	{vaa.ChainIDSolana, "0x0e0a589e6488147a94dcfa592b90fdd41152bb2ca77bf6016758a6f4df9d21b4", 80_000, 40_000},
	{vaa.ChainIDPolygon, "0x5a58505a96d1dbf8df91cb21b54419fc36e93fde", 30_000, 15_000},
}

// Emitter table (5 entries) — 4 valid token bridges + 1 invalid.
var fuzzEmitterTable = []fuzzEmitterConfig{
	{vaa.ChainIDEthereum, "0x0290fb167208af455bb137780163b7b7a9a10c16"},
	{vaa.ChainIDSui, "0xc57508ee0d4595e5a8728974a4a93a787d38f339757230d441e895422c07aba9"},
	{vaa.ChainIDSolana, "0x0e0a589e6488147a94dcfa592b90fdd41152bb2ca77bf6016758a6f4df9d21b4"},
	{vaa.ChainIDPolygon, "0x5a58505a96d1dbf8df91cb21b54419fc36e93fde"},
	{vaa.ChainIDEthereum, "0x00000000000000000000000000000000000000000000000000000000BAADF00D"}, // invalid
}

// Token table (8 entries) — 2 flow-cancel, 5 governed, 1 ungoverned.
var fuzzTokenTable = []fuzzTokenConfig{
	// Flow-cancel tokens (idx 0-1) — USDC must stay at $1.0
	{vaa.ChainIDSolana, "c6fa7af3bedbad3a3d65f36aabc97431b1bbe4c2d2f6e0e47ca60203452f5d61", "USDC_SOL", 1.0, true, true},
	{vaa.ChainIDEthereum, "000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", "USDC_ETH", 1.0, true, true},
	// Governed non-flow-cancel tokens (idx 2-6)
	{vaa.ChainIDEthereum, "0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E", "WETH", 1774.62, false, true},
	{vaa.ChainIDEthereum, "000000000000000000000000dac17f958d2ee523a2206206994597c13d831ec7", "USDT", 1.0, false, true},
	{vaa.ChainIDEthereum, "0000000000000000000000006b175474e89094c44da98b954eedeac495271d0f", "ETH", 1337.0, false, true},
	{vaa.ChainIDSolana, "ce010e60afedb22717bd63192f54145a3f965a33bb82d2c7029eb2ce1e208264", "USDT_SOL", 1.0, false, true},
	{vaa.ChainIDSui, "0x84a5f374d29fc77e370014dce4fd6a55b58ad608de8074b0be5571701724da31", "SUI_TOKEN", 0.50, false, true},
	// Ungoverned token (idx 7) — NOT registered
	{vaa.ChainIDEthereum, "000000000000000000000000DEADBEEFDEADBEEFDEADBEEFDEADBEEFDEADBEEF", "UNGOVERNED", 0, false, false},
}

// fuzzLimitEntry holds a (dailyLimit, bigTxSize) pair. The fuzzer picks from
// this table by index so that values are easy to reason about in traces.
type fuzzLimitEntry struct {
	dailyLimit uint64
	bigTxSize  uint64
}

// Predefined governor limit configurations. One representative from each
// magnitude band (tight → default → large), plus asymmetric edge cases.
var fuzzLimitTable = [15]fuzzLimitEntry{
	// Representative from each band
	{500, 100},              // 0: very tight — most transfers enqueued
	{5_000, 1_000},          // 1: tight
	{30_000, 10_000},        // 2: low-moderate
	{50_000, 25_000},        // 3: moderate (matches Sui/Polygon defaults)
	{100_000, 50_000},       // 4: default (matches Ethereum default)
	{300_000, 100_000},      // 5: high
	{1_000_000, 500_000},    // 6: very high
	{10_000_000, 5_000_000}, // 7: extreme — nearly everything passes
	// Edge cases
	{0, 0},           // 8: zero — big tx disabled, everything enqueued by daily limit
	{1, 1},           // 9: near-zero — everything enqueued
	{100_000, 1},     // 10: normal daily, big tx threshold of $1
	{1, 100_000},     // 11: tiny daily, huge big tx threshold
	{100, 1},         // 12: tiny daily, trivial big tx
	{50_000, 0},      // 13: big tx checking disabled
	{50_000, 50_000}, // 14: daily == big tx (everything is "big" or fits exactly)
}

// Pre-resolved vaa.Address values for the token table.
var fuzzTokenAddrs [8]vaa.Address

// Fixed recipient address for all fuzz payloads.
var fuzzRecipientAddr vaa.Address

func init() {
	for i, tok := range fuzzTokenTable {
		addr, err := vaa.StringToAddress(tok.addr)
		if err != nil {
			panic(fmt.Sprintf("fuzzTokenTable[%d]: bad address %q: %v", i, tok.addr, err))
		}
		fuzzTokenAddrs[i] = addr
	}
	fuzzRecipientAddr, _ = vaa.StringToAddress("0x707f9118e33a9b8998bea41dd0d46f38bb963fc8")
}

// =============================================================================
// Operation constants
// =============================================================================

const (
	opProcessMsg       = 0
	opCheckPending     = 1
	opAdmin            = 2
	opAdvanceTime      = 3
	opChangeTokenPrice = 4
	opChangeGovLimit   = 5
	fuzzOpCount        = 6
	fuzzPayloadBytes   = 11
	fuzzBytesPerOp     = 12 // 1 opcode + 11 payload

	// Admin sub-actions (selected by emitterIdx byte % 3)
	adminRelease     = 0
	adminDrop        = 1
	adminResetTimer  = 2
	adminActionCount = 3
)

// =============================================================================
// Serialization: fuzzOp → []byte
// =============================================================================

// fuzzOp is the structured representation of a single fuzz operation.
// All operations serialize to the same 12 bytes (1 opcode + 11 payload).
// Fields are reused by different opcodes:
//
//	ProcessMsg:        tokenIdx, emitterIdx, targetChainIdx, amount
//	CheckPending:      (all payload discarded)
//	Admin:             tokenIdx=pendingIdx, emitterIdx=actionByte (release/drop/resetTimer), amount=numDays (for resetTimer)
//	AdvanceTime:       tokenIdx=minutesByte
//	ChangeTokenPrice:  tokenIdx, amount=priceCents
//	ChangeGovLimit:   tokenIdx=chainIdx, emitterIdx=limitIdx
type fuzzOp struct {
	opcode         byte
	tokenIdx       byte
	emitterIdx     byte
	targetChainIdx byte
	amount         uint64
}

func encodeFuzzOps(ops []fuzzOp) []byte {
	buf := make([]byte, 0, len(ops)*fuzzBytesPerOp)
	for _, op := range ops {
		buf = append(buf, op.opcode)
		buf = append(buf, op.tokenIdx, op.emitterIdx, op.targetChainIdx)
		var amt [8]byte
		binary.LittleEndian.PutUint64(amt[:], op.amount)
		buf = append(buf, amt[:]...)
	}
	return buf
}

// =============================================================================
// Deserialization: fuzzReader
// =============================================================================

type fuzzReader struct {
	data []byte
	pos  int
}

func (r *fuzzReader) readByte() byte {
	if r.pos >= len(r.data) {
		return 0
	}
	b := r.data[r.pos]
	r.pos++
	return b
}

func (r *fuzzReader) readUint64LE() uint64 {
	var buf [8]byte
	for i := range buf {
		buf[i] = r.readByte()
	}
	return binary.LittleEndian.Uint64(buf[:])
}

func (r *fuzzReader) remaining() int {
	return len(r.data) - r.pos
}

func (r *fuzzReader) discard(n int) {
	for i := 0; i < n; i++ {
		r.readByte()
	}
}

// =============================================================================
// Governor setup
// =============================================================================

func newFuzzGovernor(t *testing.T) *ChainGovernor {
	t.Helper()
	var db guardianDB.MockGovernorDB
	return newFuzzGovernorWithDB(t, &db)
}

func newFuzzGovernorWithDB(t *testing.T, db guardianDB.GovernorDB) *ChainGovernor {
	t.Helper()

	var logger *zap.Logger
	if *fuzzVerboseGov {
		logger = zaptest.NewLogger(t)
	} else {
		logger = zap.NewNop()
	}
	gov := NewChainGovernor(logger, db, common.GoTest, true, "")
	// Skip gov.Run() to avoid loading ~1700 mainnet tokens per iteration.
	// Manually configure only the chains/tokens we need.
	gov.setDayLengthInMinutes(24 * 60)
	gov.flowCancelCorridors = []corridor{
		{first: vaa.ChainIDEthereum, second: vaa.ChainIDSui},
	}

	for _, c := range fuzzChainTable {
		if err := gov.setChainForTesting(c.chainID, c.tokenBridgeAddr, c.dailyLimit, c.bigTxSize); err != nil {
			t.Fatalf("setChainForTesting(%s): %v", c.chainID, err)
		}
	}

	for _, tok := range fuzzTokenTable {
		if !tok.governed {
			continue
		}
		if err := gov.setTokenForTesting(tok.originChain, tok.addr, tok.symbol, tok.price, tok.flowCancels); err != nil {
			t.Fatalf("setTokenForTesting(%s): %v", tok.symbol, err)
		}
	}

	return gov
}

// =============================================================================
// Message builder
// =============================================================================

func buildFuzzPayload(tokenIdx int, targetChainIdx int, amount uint64) []byte {
	payload := make([]byte, 101)
	payload[0] = 1 // type 1 = transfer (type 3 is processed identically)

	amtBig := new(big.Int).SetUint64(amount)
	amtBytes := amtBig.Bytes()
	if len(amtBytes) <= 32 {
		copy(payload[33-len(amtBytes):33], amtBytes)
	}

	copy(payload[33:65], fuzzTokenAddrs[tokenIdx].Bytes())
	binary.BigEndian.PutUint16(payload[65:67], uint16(fuzzTokenTable[tokenIdx].originChain))
	copy(payload[67:99], fuzzRecipientAddr.Bytes())
	binary.BigEndian.PutUint16(payload[99:101], uint16(fuzzChainTable[targetChainIdx].chainID))

	return payload
}

func buildFuzzMsg(seq uint64, now time.Time, emitterIdx int, tokenIdx int, targetChainIdx int, amount uint64) common.MessagePublication {
	emitter := fuzzEmitterTable[emitterIdx]
	emitterAddr, _ := vaa.StringToAddress(emitter.emitterAddr)
	txHash := eth_common.BytesToHash([]byte(fmt.Sprintf("fuzz-%d", seq)))

	return common.MessagePublication{
		TxID:             txHash.Bytes(),
		Timestamp:        now,
		Nonce:            uint32(seq),
		Sequence:         seq,
		EmitterChain:     emitter.chainID,
		EmitterAddress:   emitterAddr,
		ConsistencyLevel: 32,
		Payload:          buildFuzzPayload(tokenIdx, targetChainIdx, amount),
	}
}

// =============================================================================
// Invariant checks
// =============================================================================

func checkGovernorInvariants(t *testing.T, gov *ChainGovernor, tracker *invariantTracker, now time.Time) {
	t.Helper()

	gov.mutex.Lock()

	enqueuedInSeen := 0
	for _, complete := range gov.msgsSeen {
		if !complete {
			enqueuedInSeen++
		}
	}

	totalPendingFromChains := 0
	for _, ce := range gov.chains {
		totalPendingFromChains += len(ce.pending)
	}

	// Compare per-chain transfer counts and pending counts against shadow state.
	for chainID, ce := range gov.chains {
		govTransferCount := len(ce.transfers)
		govPendingCount := len(ce.pending)

		shadowCS, ok := tracker.chains[chainID]
		if !ok {
			t.Errorf("invariant violation: governor has chain %s but shadow does not", chainID)
			continue
		}
		shadowTransferCount := len(shadowCS.transfers)

		if govTransferCount != shadowTransferCount {
			t.Errorf("invariant violation: chain %s transfer count mismatch: governor=%d shadow=%d",
				chainID, govTransferCount, shadowTransferCount)
		}

		shadowPendingCount := 0
		for _, pe := range tracker.pending {
			if pe.emitterChain == chainID {
				shadowPendingCount++
			}
		}
		if govPendingCount != shadowPendingCount {
			t.Errorf("invariant violation: chain %s pending count mismatch: governor=%d shadow=%d",
				chainID, govPendingCount, shadowPendingCount)
		}

		for i := 1; i < len(ce.pending); i++ {
			prev := ce.pending[i-1].dbData.ReleaseTime
			curr := ce.pending[i].dbData.ReleaseTime
			if curr.Before(prev) {
				t.Errorf("invariant violation: chain %s pending[%d] releaseTime %v is before pending[%d] releaseTime %v",
					chainID, i, curr, i-1, prev)
			}
		}

		// Compare the trimmed sum for this chain without side effects.
		startTime := now.Add(-time.Duration(shadowDayMinutes) * time.Minute)

		var govSum int64
		for _, tr := range ce.transfers {
			if !tr.dbTransfer.Timestamp.Before(startTime) {
				govSum += tr.scaledValue
			}
		}
		if govSum < 0 {
			govSum = 0
		}

		var shadowSumSigned int64
		for _, st := range shadowCS.transfers {
			if !st.timestamp.Before(startTime) {
				shadowSumSigned += st.scaledValue
			}
		}
		if shadowSumSigned < 0 {
			shadowSumSigned = 0
		}

		if govSum != shadowSumSigned {
			t.Errorf("invariant violation: chain %s sum mismatch: governor=%d shadow=%d delta=%d",
				chainID, govSum, shadowSumSigned, shadowSumSigned-govSum)
		}
	}

	gov.mutex.Unlock()

	// Note: admin Release/Drop remove from ce.pending but do not update msgsSeen,
	// so enqueuedInSeen >= totalPendingFromChains. The reverse (more pending than
	// msgsSeen thinks) would indicate a real bug.
	if totalPendingFromChains > enqueuedInSeen {
		t.Errorf("invariant violation: chain pending=%d > msgsSeen enqueued=%d",
			totalPendingFromChains, enqueuedInSeen)
	}

	numTrans, _, numPending, _ := gov.getStatsForAllChains()
	if numPending != totalPendingFromChains {
		t.Errorf("invariant violation: stats numPending=%d != chain pending=%d",
			numPending, totalPendingFromChains)
	}

	if numTrans < 0 {
		t.Errorf("invariant violation: numTrans is negative: %d", numTrans)
	}
}

// =============================================================================
// Invariant tracker — fully independent shadow state machine
// =============================================================================

// shadowTransfer represents a published transfer in our independent 24h window tracker.
type shadowTransfer struct {
	timestamp   time.Time
	scaledValue int64 // signed: positive for outgoing, negative for flow cancel
}

// shadowChainState tracks published (non-big) transfer values for one chain.
type shadowChainState struct {
	transfers []shadowTransfer
}

// shadowPendingEntry tracks an enqueued transfer in our shadow pending queue.
type shadowPendingEntry struct {
	emitterChain vaa.ChainID
	targetChain  vaa.ChainID
	tokenIdx     int
	amount       *big.Int // raw amount, for re-evaluation at current price
	releaseTime  time.Time
	msgID        string // MessageIDString() — used to match admin ops
}

// invariantTracker is an independent state machine that mirrors the governor's
// chain usage and queuing decisions, built from the specification rather than
// the implementation. It never reads the governor's internal state.
type invariantTracker struct {
	chains      map[vaa.ChainID]*shadowChainState
	chainOrder  []vaa.ChainID                  // sorted ascending, matches governor's chainIds
	prices      map[int]float64                // tokenIdx -> current price
	chainLimits map[vaa.ChainID]fuzzLimitEntry // current per-chain limits (mutable copy)
	pending     []shadowPendingEntry           // our independent pending queue
}

const (
	shadowScaledValueFactor uint64 = 100_000
	shadowDecimals          int64  = 1e8 // all fuzz tokens use 8 decimals
	shadowDayMinutes               = 24 * 60
	shadowMaxEnqueuedTime          = 24 * time.Hour
)

func newInvariantTracker() *invariantTracker {
	tr := &invariantTracker{
		chains:      make(map[vaa.ChainID]*shadowChainState),
		prices:      make(map[int]float64),
		chainLimits: make(map[vaa.ChainID]fuzzLimitEntry),
	}
	for _, c := range fuzzChainTable {
		tr.chains[c.chainID] = &shadowChainState{}
		tr.chainOrder = append(tr.chainOrder, c.chainID)
		tr.chainLimits[c.chainID] = fuzzLimitEntry{dailyLimit: c.dailyLimit, bigTxSize: c.bigTxSize}
	}
	sort.Slice(tr.chainOrder, func(i, j int) bool {
		return tr.chainOrder[i] < tr.chainOrder[j]
	})
	for i, tok := range fuzzTokenTable {
		tr.prices[i] = tok.price
	}
	return tr
}

// computeScaledValue reimplements the USD value calculation from the spec:
// scaledValue = amount * price * ScaledValueFactor / 10^decimals
func (tr *invariantTracker) computeScaledValue(tokenIdx int, amount *big.Int) (uint64, bool) {
	if amount == nil || amount.Sign() < 0 {
		return 0, false
	}
	price := tr.prices[tokenIdx]
	if price <= 0 {
		return 0, false
	}

	amountFloat := new(big.Float).SetInt(amount)
	valueFloat := new(big.Float).Mul(amountFloat, big.NewFloat(price))
	valueBigInt, _ := valueFloat.Int(nil)
	valueBigInt.Mul(valueBigInt, big.NewInt(int64(shadowScaledValueFactor)))
	valueBigInt.Div(valueBigInt, big.NewInt(shadowDecimals))

	if !valueBigInt.IsUint64() {
		return 0, false
	}
	return valueBigInt.Uint64(), true
}

// trimAndSum returns the sum of transfer values within the 24h window for a chain.
// Mirrors the governor's trimAndSumValueForChain: trims old entries, sums remaining,
// and clamps negative sums to 0.
func (tr *invariantTracker) trimAndSum(chainID vaa.ChainID, now time.Time) uint64 {
	cs, ok := tr.chains[chainID]
	if !ok {
		return 0
	}
	startTime := now.Add(-time.Duration(shadowDayMinutes) * time.Minute)

	// Trim old transfers
	trimIdx := 0
	for trimIdx < len(cs.transfers) && cs.transfers[trimIdx].timestamp.Before(startTime) {
		trimIdx++
	}
	cs.transfers = cs.transfers[trimIdx:]

	var sum int64
	for _, t := range cs.transfers {
		sum += t.scaledValue
	}
	// Clamp negative to 0, matching governor's trimAndSumValueForChain
	if sum <= 0 {
		return 0
	}
	return uint64(sum)
}

// isGoverned returns true if the emitter is a valid token bridge and the token
// is in the governed list.
func (tr *invariantTracker) isGoverned(tokenIdx, emitterIdx int) bool {
	if !fuzzTokenTable[tokenIdx].governed {
		return false
	}
	emitter := fuzzEmitterTable[emitterIdx]
	for _, c := range fuzzChainTable {
		if c.chainID == emitter.chainID {
			emitterAddr, _ := vaa.StringToAddress(emitter.emitterAddr)
			bridgeAddr, _ := vaa.StringToAddress(c.tokenBridgeAddr)
			return emitterAddr == bridgeAddr
		}
	}
	return false
}

// chainLimitForEmitter returns the current (possibly mutated) limits for a chain.
func (tr *invariantTracker) chainLimitForEmitter(emitterChainID vaa.ChainID) (fuzzLimitEntry, bool) {
	lim, ok := tr.chainLimits[emitterChainID]
	return lim, ok
}

// isBigTransfer checks whether a scaled value qualifies as a "big transfer"
// for the given emitter chain.
func (tr *invariantTracker) isBigTransfer(emitterChainID vaa.ChainID, scaledValue uint64) bool {
	lim, ok := tr.chainLimitForEmitter(emitterChainID)
	if !ok || lim.bigTxSize == 0 {
		return false
	}
	return scaledValue >= lim.bigTxSize*shadowScaledValueFactor
}

// isFlowCancelEligible returns true if this transfer should generate a
// flow-cancel (inverse) entry on the destination chain.
func isFlowCancelEligible(tokenIdx int, emitterChainID, targetChainID vaa.ChainID) bool {
	if !fuzzTokenTable[tokenIdx].flowCancels {
		return false
	}
	// Only Ethereum <-> Sui corridor is configured in the fuzzer
	return (emitterChainID == vaa.ChainIDEthereum && targetChainID == vaa.ChainIDSui) ||
		(emitterChainID == vaa.ChainIDSui && targetChainID == vaa.ChainIDEthereum)
}

// shouldEnqueue independently determines whether a transfer should be queued.
// Returns (shouldEnqueue, applicable). applicable=false means we can't check
// (ungoverned, invalid emitter, etc.).
func (tr *invariantTracker) shouldEnqueue(
	tokenIdx, emitterIdx, targetChainIdx int,
	amount *big.Int,
	now time.Time,
) (bool, bool) {
	if !tr.isGoverned(tokenIdx, emitterIdx) {
		return false, false
	}

	emitterChainID := fuzzEmitterTable[emitterIdx].chainID

	scaledValue, ok := tr.computeScaledValue(tokenIdx, amount)
	if !ok {
		return false, false
	}

	// Big transfers are always enqueued
	if tr.isBigTransfer(emitterChainID, scaledValue) {
		return true, true
	}

	// Check if adding this transfer would exceed the daily limit
	lim, ok := tr.chainLimitForEmitter(emitterChainID)
	if !ok {
		return false, false
	}

	prevSum := tr.trimAndSum(emitterChainID, now)

	newTotal := prevSum + scaledValue
	// Check for overflow
	if newTotal < prevSum {
		return true, true
	}

	if newTotal > lim.dailyLimit*shadowScaledValueFactor {
		return true, true
	}

	return false, true
}

// recordPublish updates the tracker when a governed transfer is published.
// Adds the transfer to the emitter chain's usage, and if flow-cancel eligible,
// adds the inverse to the destination chain.
func (tr *invariantTracker) recordPublish(
	tokenIdx, emitterIdx, targetChainIdx int,
	amount *big.Int,
	now time.Time,
) {
	emitterChainID := fuzzEmitterTable[emitterIdx].chainID
	targetChainID := fuzzChainTable[targetChainIdx].chainID

	scaledValue, ok := tr.computeScaledValue(tokenIdx, amount)
	if !ok {
		return
	}

	// Big transfers are excluded from the transfer list entirely
	if tr.isBigTransfer(emitterChainID, scaledValue) {
		return
	}

	// Add to emitter chain's transfer list
	if cs, exists := tr.chains[emitterChainID]; exists {
		cs.transfers = append(cs.transfers, shadowTransfer{
			timestamp:   now,
			scaledValue: int64(scaledValue),
		})
	}

	// Flow cancel: add inverse to destination chain
	if isFlowCancelEligible(tokenIdx, emitterChainID, targetChainID) {
		if cs, exists := tr.chains[targetChainID]; exists {
			cs.transfers = append(cs.transfers, shadowTransfer{
				timestamp:   now,
				scaledValue: -int64(scaledValue),
			})
		}
	}
}

// recordEnqueue adds a transfer to the shadow pending queue.
func (tr *invariantTracker) recordEnqueue(
	tokenIdx, emitterIdx, targetChainIdx int,
	amount *big.Int,
	now time.Time,
	msgID string,
) {
	entry := shadowPendingEntry{
		emitterChain: fuzzEmitterTable[emitterIdx].chainID,
		targetChain:  fuzzChainTable[targetChainIdx].chainID,
		tokenIdx:     tokenIdx,
		amount:       new(big.Int).Set(amount),
		releaseTime:  now.Add(shadowMaxEnqueuedTime),
		msgID:        msgID,
	}
	idx := sort.Search(len(tr.pending), func(i int) bool {
		return entry.releaseTime.Before(tr.pending[i].releaseTime)
	})
	tr.pending = append(tr.pending, shadowPendingEntry{})
	copy(tr.pending[idx+1:], tr.pending[idx:])
	tr.pending[idx] = entry
}

// addTransferAndFlowCancel appends a transfer to the emitter chain and,
// if the token is flow-cancel eligible, appends the inverse to the target chain.
func (tr *invariantTracker) addTransferAndFlowCancel(
	emitterChain, targetChain vaa.ChainID,
	tokenIdx int,
	scaledValue uint64,
	now time.Time,
) {
	if cs, exists := tr.chains[emitterChain]; exists {
		cs.transfers = append(cs.transfers, shadowTransfer{
			timestamp:   now,
			scaledValue: int64(scaledValue),
		})
	}
	if isFlowCancelEligible(tokenIdx, emitterChain, targetChain) {
		if cs, exists := tr.chains[targetChain]; exists {
			cs.transfers = append(cs.transfers, shadowTransfer{
				timestamp:   now,
				scaledValue: -int64(scaledValue),
			})
		}
	}
}

// shadowCheckPending reimplements the governor's checkPendingForTime logic.
// For each chain (in sorted order), repeatedly try to release one pending
// entry at a time until no more can be released.
func (tr *invariantTracker) shadowCheckPending(t *testing.T, now time.Time) {
	for _, chainID := range tr.chainOrder {
		lim, ok := tr.chainLimitForEmitter(chainID)
		if !ok {
			continue
		}

		for {
			foundOne := false
			prevSum := tr.trimAndSum(chainID, now)

			for idx := 0; idx < len(tr.pending); idx++ {
				pe := &tr.pending[idx]
				if pe.emitterChain != chainID {
					continue
				}

				scaledValue, ok := tr.computeScaledValue(pe.tokenIdx, pe.amount)
				if !ok {
					continue
				}

				if tr.isBigTransfer(chainID, scaledValue) {
					// Big transfers: release only when timer expires.
					// Do NOT add to transfers.
					if !now.Before(pe.releaseTime) {
						if *fuzzVerboseGov {
							t.Logf("TRACE shadowCheckPending: chain=%s releasing big transfer token=%s scaledValue=%d (timer expired, releaseTime=%v)",
								chainID, fuzzTokenTable[pe.tokenIdx].symbol, scaledValue, pe.releaseTime)
						}
						tr.pending = append(tr.pending[:idx], tr.pending[idx+1:]...)
						foundOne = true
						break
					}
					if *fuzzVerboseGov {
						t.Logf("TRACE shadowCheckPending: chain=%s skipping big transfer token=%s scaledValue=%d (timer not expired, releaseTime=%v)",
							chainID, fuzzTokenTable[pe.tokenIdx].symbol, scaledValue, pe.releaseTime)
					}
					continue
				}

				if now.After(pe.releaseTime) {
					// Non-big, past release time: release regardless of limit.
					// Do NOT add to transfers.
					if *fuzzVerboseGov {
						t.Logf("TRACE shadowCheckPending: chain=%s releasing expired non-big token=%s scaledValue=%d (releaseTime=%v)",
							chainID, fuzzTokenTable[pe.tokenIdx].symbol, scaledValue, pe.releaseTime)
					}
					tr.pending = append(tr.pending[:idx], tr.pending[idx+1:]...)
					foundOne = true
					break
				}

				// Non-big, within release time: release only if it fits under limit.
				newTotal := prevSum + scaledValue
				if newTotal < prevSum {
					if *fuzzVerboseGov {
						t.Logf("TRACE shadowCheckPending: chain=%s skipping non-big token=%s scaledValue=%d (overflow)",
							chainID, fuzzTokenTable[pe.tokenIdx].symbol, scaledValue)
					}
					continue // overflow, doesn't fit
				}
				if newTotal > lim.dailyLimit*shadowScaledValueFactor {
					if *fuzzVerboseGov {
						t.Logf("TRACE shadowCheckPending: chain=%s skipping non-big token=%s scaledValue=%d (exceeds limit: %d+%d=%d > %d)",
							chainID, fuzzTokenTable[pe.tokenIdx].symbol, scaledValue, prevSum, scaledValue, newTotal, lim.dailyLimit*shadowScaledValueFactor)
					}
					continue // doesn't fit
				}

				// Fits: release and add to transfers (+ flow cancel)
				if *fuzzVerboseGov {
					t.Logf("TRACE shadowCheckPending: chain=%s releasing non-big token=%s scaledValue=%d (fits: %d+%d=%d <= %d)",
						chainID, fuzzTokenTable[pe.tokenIdx].symbol, scaledValue, prevSum, scaledValue, newTotal, lim.dailyLimit*shadowScaledValueFactor)
				}
				tr.addTransferAndFlowCancel(chainID, pe.targetChain, pe.tokenIdx, scaledValue, now)
				tr.pending = append(tr.pending[:idx], tr.pending[idx+1:]...)
				foundOne = true
				break
			}

			if !foundOne {
				break
			}
		}
	}
}

// recordAdminRelease removes a pending transfer by msgID.
// Admin releases do NOT count toward the daily limit.
func (tr *invariantTracker) recordAdminRelease(msgID string) {
	for idx, pe := range tr.pending {
		if pe.msgID == msgID {
			tr.pending = append(tr.pending[:idx], tr.pending[idx+1:]...)
			return
		}
	}
}

// recordAdminDrop removes a pending transfer by msgID without publishing.
func (tr *invariantTracker) recordAdminDrop(msgID string) {
	for idx, pe := range tr.pending {
		if pe.msgID == msgID {
			tr.pending = append(tr.pending[:idx], tr.pending[idx+1:]...)
			return
		}
	}
}

// recordAdminResetTimer updates the release time for a pending transfer.
func (tr *invariantTracker) recordAdminResetTimer(msgID string, now time.Time, numDays uint32) {
	for idx := range tr.pending {
		if tr.pending[idx].msgID == msgID {
			tr.pending[idx].releaseTime = now.Add(time.Duration(numDays) * 24 * time.Hour)
			// Re-sort to match the production code's ordering invariant.
			sort.SliceStable(tr.pending, func(i, j int) bool {
				return tr.pending[i].releaseTime.Before(tr.pending[j].releaseTime)
			})
			return
		}
	}
}

// checkQueuingResult verifies that the governor's queuing decision matches
// our independent calculation.
func (tr *invariantTracker) checkQueuingResult(
	t *testing.T,
	tokenIdx, emitterIdx int,
	amount *big.Int,
	now time.Time,
	published bool,
	expectEnqueue bool,
) {
	t.Helper()

	emitterChainID := fuzzEmitterTable[emitterIdx].chainID
	scaledValue, _ := tr.computeScaledValue(tokenIdx, amount)
	lim, _ := tr.chainLimitForEmitter(emitterChainID)
	prevSum := tr.trimAndSum(emitterChainID, now)

	if expectEnqueue && published {
		t.Errorf("queuing invariant violation: transfer was published but should have been enqueued\n"+
			"  token=%s emitterChain=%s scaledValue=%d bigThreshold=%d\n"+
			"  prevSum=%d dailyLimit=%d (scaled=%d)",
			fuzzTokenTable[tokenIdx].symbol, emitterChainID,
			scaledValue, lim.bigTxSize*shadowScaledValueFactor,
			prevSum, lim.dailyLimit, lim.dailyLimit*shadowScaledValueFactor)
	}

	if !expectEnqueue && !published {
		t.Errorf("queuing invariant violation: transfer was enqueued but should have been published\n"+
			"  token=%s emitterChain=%s scaledValue=%d bigThreshold=%d\n"+
			"  prevSum=%d dailyLimit=%d (scaled=%d)",
			fuzzTokenTable[tokenIdx].symbol, emitterChainID,
			scaledValue, lim.bigTxSize*shadowScaledValueFactor,
			prevSum, lim.dailyLimit, lim.dailyLimit*shadowScaledValueFactor)
	}
}

// =============================================================================
// Seed corpus — named functions for each scenario
// =============================================================================

// Helpers for AdvanceTime: minutesByte → 1 + byte*10 minutes.
// advanceMinutes(144) = 1441 min ≈ 24h+1min. advanceMinutes(255) = 2551 min ≈ 42h.
func advanceOp(minutesByte byte) fuzzOp {
	return fuzzOp{opAdvanceTime, minutesByte, 0, 0, 0}
}

func checkPendingOp() fuzzOp {
	return fuzzOp{opCheckPending, 0, 0, 0, 0}
}

// adminOp builds an Admin fuzzOp. action: 0=release, 1=drop, 2=resetTimer.
// For resetTimer, numDays is derived from amount: 1 + (amount % 5).
func adminOp(pendingIdx byte, action byte, numDays uint64) fuzzOp {
	return fuzzOp{opAdmin, pendingIdx, action, 0, numDays}
}

func adminReleaseOp(pendingIdx byte) fuzzOp {
	return adminOp(pendingIdx, adminRelease, 0)
}

func adminDropOp(pendingIdx byte) fuzzOp {
	return adminOp(pendingIdx, adminDrop, 0)
}

func adminResetTimerOp(pendingIdx byte, numDays uint64) fuzzOp {
	// numDays stored in amount field; decoded as 1 + (amount % 5)
	return adminOp(pendingIdx, adminResetTimer, numDays-1)
}

func priceOp(tokenIdx byte, priceCents uint64) fuzzOp {
	return fuzzOp{opChangeTokenPrice, tokenIdx, 0, 0, priceCents}
}

// limitOp builds an opChangeGovLimit. chainIdx selects from fuzzChainTable,
// limitIdx selects from fuzzLimitTable.
func limitOp(chainIdx, limitIdx byte) fuzzOp {
	return fuzzOp{opChangeGovLimit, chainIdx, limitIdx, 0, 0}
}

// Token indices (from fuzzTokenTable)
const (
	tokUSDC_SOL = 0
	tokUSDC_ETH = 1
	tokWETH     = 2
	tokUSDT     = 3
	tokDAI      = 4
	tokUSDT_SOL = 5
	tokSUI      = 6
	tokUNGOVERN = 7
)

// Emitter indices (from fuzzEmitterTable)
const (
	emitEth     = 0
	emitSui     = 1
	emitSol     = 2
	emitPoly    = 3
	emitInvalid = 4
)

// Chain indices (from fuzzChainTable) — for target chain
const (
	chainEth  = 0
	chainSui  = 1
	chainSol  = 2
	chainPoly = 3
)

// msg builds a ProcessMsg fuzzOp.
func msg(tokenIdx, emitterIdx, targetChainIdx byte, amount uint64) fuzzOp {
	return fuzzOp{opProcessMsg, tokenIdx, emitterIdx, targetChainIdx, amount}
}

// wrapSeed sandwiches core operations between a diverse prefix and suffix.
// The prefix/suffix ensure every seed uses all 5 op types, 3+ tokens, and
// has 75%+ ProcessMsg ops across 20-100 total operations.
func wrapSeed(core []fuzzOp) []fuzzOp {
	prefix := []fuzzOp{
		// Diverse transfers across chains and tokens (8 msgs)
		msg(tokWETH, emitEth, chainSui, 500_000),
		msg(tokUSDT, emitEth, chainPoly, 300_000_000),
		msg(tokDAI, emitPoly, chainEth, 200_000_000),
		msg(tokSUI, emitSui, chainEth, 400_000_000),
		msg(tokUSDC_ETH, emitEth, chainSui, 1_000_000_000),
		msg(tokUSDT_SOL, emitSol, chainPoly, 150_000_000),
		msg(tokUSDC_SOL, emitSui, chainEth, 500_000_000),
		msg(tokWETH, emitEth, chainSol, 100_000),
		// Time advance and check
		advanceOp(6),
		checkPendingOp(),
		// Price change
		priceOp(tokWETH, 200_000), // $2000 per WETH
		// Admin op — will no-op if nothing pending
		adminResetTimerOp(0, 2),
		// Limit change — tighten Sui, loosen Ethereum
		limitOp(chainSui, 1), // 5k/1k
		limitOp(chainEth, 6), // 1M/500k
	}

	suffix := []fuzzOp{
		// Transfers (6 msgs)
		msg(tokDAI, emitEth, chainSui, 250_000_000),
		msg(tokUSDT, emitSol, chainEth, 180_000_000),
		msg(tokWETH, emitEth, chainPoly, 350_000),
		msg(tokSUI, emitSui, chainPoly, 600_000_000),
		msg(tokUSDC_ETH, emitSui, chainEth, 800_000_000),
		msg(tokUNGOVERN, emitEth, chainSui, 100_000),
		// Time, check, price, admin
		advanceOp(36),
		checkPendingOp(),
		priceOp(tokSUI, 75),
		adminDropOp(0),
		// Restore limits to defaults
		limitOp(chainSui, 3), // 50k/25k
		limitOp(chainEth, 4), // 100k/50k
		// Post-admin transfers to exercise state after admin ops (4 msgs)
		msg(tokWETH, emitEth, chainSui, 200_000),
		msg(tokUSDT, emitEth, chainSol, 400_000_000),
		msg(tokDAI, emitPoly, chainSui, 300_000_000),
		msg(tokUSDC_ETH, emitEth, chainSui, 150_000_000),
	}

	result := make([]fuzzOp, 0, len(prefix)+len(core)+len(suffix))
	result = append(result, prefix...)
	result = append(result, core...)
	result = append(result, suffix...)
	return result
}

// seedSmallTransferPublished: single small WETH transfer well under limit.
func seedSmallTransferPublished() []fuzzOp {
	return wrapSeed([]fuzzOp{msg(tokWETH, emitEth, chainPoly, 100)})
}

// seedSmallTransferEachChain: one small transfer from each of the 4 chains.
func seedSmallTransferEachChain() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 100),
		msg(tokSUI, emitSui, chainEth, 100),
		msg(tokUSDT_SOL, emitSol, chainPoly, 100),
		msg(tokDAI, emitPoly, chainEth, 100),
	})
}

// seedBigTransferEnqueuedAndAutoReleased: big WETH transfer enqueued, 42h advance, auto-release.
func seedBigTransferEnqueuedAndAutoReleased() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000_000_000),
		advanceOp(255), // ~42h, past 24h auto-release
		checkPendingOp(),
	})
}

// seedFlowCancelFullOffset: Eth→Sui USDC, then Sui→Eth USDC of equal size — full cancel.
func seedFlowCancelFullOffset() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_ETH, emitEth, chainSui, 1_000_000_000), // $10k
		msg(tokUSDC_ETH, emitSui, chainEth, 1_000_000_000), // $10k
	})
}

// seedFlowCancelPartialOffset: outbound > inbound, partial cancel.
func seedFlowCancelPartialOffset() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_ETH, emitEth, chainSui, 5_000_000_000), // $50k
		msg(tokUSDC_ETH, emitSui, chainEth, 1_000_000_000), // $10k
	})
}

// seedFlowCancelExceedsOutbound: inbound flow-cancel exceeds outbound — sum clamps to 0.
func seedFlowCancelExceedsOutbound() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_ETH, emitEth, chainSui, 1_000_000_000), // $10k
		msg(tokUSDC_ETH, emitSui, chainEth, 5_000_000_000), // $50k
	})
}

// seedFlowCancelThenNewTransfer: flow cancel frees capacity, then a new transfer fits.
func seedFlowCancelThenNewTransfer() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_ETH, emitEth, chainSui, 4_000_000_000), // $40k of 100k Eth limit
		msg(tokWETH, emitEth, chainSui, 1_127_000),         // ~$20k WETH
		msg(tokUSDC_ETH, emitSui, chainEth, 3_000_000_000), // flow cancel frees $30k on Eth
		msg(tokWETH, emitEth, chainPoly, 282_000),          // ~$5k WETH, should fit now
	})
}

// seedFlowCancelWrongCorridor: flow cancel on Eth→Polygon (not a corridor), should NOT cancel.
func seedFlowCancelWrongCorridor() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_ETH, emitEth, chainPoly, 1_000_000_000), // $10k
		msg(tokUSDC_ETH, emitPoly, chainEth, 1_000_000_000), // $10k
	})
}

// seedFlowCancelNonFlowToken: WETH (non-flow-cancel) between corridor chains — no cancel.
func seedFlowCancelNonFlowToken() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 563_000), // ~$10k WETH
		msg(tokWETH, emitSui, chainEth, 563_000), // ~$10k WETH
	})
}

// seedEnqueueResetTimerAndRelease: enqueue big transfer, reset timer to 3 days, advance past, release.
func seedEnqueueResetTimerAndRelease() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainPoly, 1_000_000_000_000),
		adminResetTimerOp(0, 3), // 3 days
		advanceOp(255),          // ~42h (not enough for 3 days)
		checkPendingOp(),        // should still be pending
		advanceOp(255),          // ~84h total (~3.5 days)
		checkPendingOp(),        // should release now
	})
}

// seedEnqueueResetTimerShort: enqueue, reset to 1 day, advance 25h, release.
func seedEnqueueResetTimerShort() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000_000_000),
		adminResetTimerOp(0, 1), // 1 day
		advanceOp(150),          // ~25h
		checkPendingOp(),
	})
}

// seedTransfersUpToLimit: fill Eth chain to capacity with small transfers.
// Eth limit = 100,000 USD. USDT at $1 with 8 decimals and ×1000 scaling, so seed 10B = $100k.
func seedTransfersUpToLimit() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDT, emitEth, chainSui, 5_000_000_000),  // $50k
		msg(tokUSDT, emitEth, chainPoly, 4_000_000_000), // $40k
		msg(tokUSDT, emitEth, chainSol, 1_000_000_000),  // $10k (at limit)
	})
}

// seedTransfersOverLimit: fill Eth chain, then one more that gets enqueued.
func seedTransfersOverLimit() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDT, emitEth, chainSui, 5_000_000_000),  // $50k
		msg(tokUSDT, emitEth, chainPoly, 5_000_000_000), // $50k (at limit)
		msg(tokUSDT, emitEth, chainSol, 1_000_000_000),  // $10k → enqueued
	})
}

// seedTransfersOverLimitThenExpire: fill, enqueue, advance 24h to expire old ones, check pending.
func seedTransfersOverLimitThenExpire() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDT, emitEth, chainSui, 5_000_000_000),
		msg(tokUSDT, emitEth, chainPoly, 5_000_000_000),
		msg(tokUSDT, emitEth, chainSol, 2_000_000_000), // enqueued
		advanceOp(144),   // ~24h, old transfers expire
		checkPendingOp(), // pending should release
	})
}

// seedMultiplePendingReleaseOrder: multiple enqueued transfers, released in order after time.
func seedMultiplePendingReleaseOrder() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDT, emitEth, chainSui, 5_000_000_000),  // $50k published
		msg(tokUSDT, emitEth, chainPoly, 5_000_000_000), // $50k published (at limit)
		msg(tokUSDT, emitEth, chainSol, 1_000_000_000),  // $10k enqueued
		msg(tokUSDT, emitEth, chainSui, 2_000_000_000),  // $20k enqueued
		msg(tokUSDT, emitEth, chainPoly, 500_000_000),   // $5k enqueued
		advanceOp(255), // ~42h
		checkPendingOp(),
	})
}

// seedBigTransferThresholdBoundary: transfer exactly at big-tx threshold (50k for Eth).
// WETH at $1774.62, big-tx = 50,000. 50000/1774.62 ≈ 28.17 WETH.
// With *1000 scaling: 2_817_000 * 1000 / 1e8 = 28.17 WETH * $1774.62 ≈ $50k.
func seedBigTransferThresholdBoundary() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 2_817_000), // right at big-tx boundary
	})
}

// seedBigTransferJustUnder: just under big-tx threshold.
func seedBigTransferJustUnder() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 2_800_000),
	})
}

// seedBigTransferJustOver: just over big-tx threshold — should enqueue.
func seedBigTransferJustOver() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 2_850_000),
	})
}

// seedInvalidEmitter: message from unrecognized emitter address.
func seedInvalidEmitter() []fuzzOp {
	return wrapSeed([]fuzzOp{msg(tokWETH, emitInvalid, chainSui, 1_000_000)})
}

// seedUngovernedToken: message with unregistered token.
func seedUngovernedToken() []fuzzOp {
	return wrapSeed([]fuzzOp{msg(tokUNGOVERN, emitEth, chainSui, 1_000_000)})
}

// seedPriceDropReleasesEnqueued: enqueue WETH, drop price to $1, pending becomes small.
func seedPriceDropReleasesEnqueued() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000_000_000),
		priceOp(tokWETH, 100), // $1.00
		advanceOp(255),
		checkPendingOp(),
	})
}

// seedPriceSpikeEnqueuesMore: raise WETH price dramatically, smaller transfers now enqueue.
func seedPriceSpikeEnqueuesMore() []fuzzOp {
	return wrapSeed([]fuzzOp{
		priceOp(tokWETH, 5_000_000),              // $50,000 per WETH
		msg(tokWETH, emitEth, chainSui, 100_000), // 1.0 WETH after ×1000 scaling, $50k USD at $50k/WETH
	})
}

// seedPriceChangeWhilePending: enqueue, change price, advance, check if re-evaluation changes behavior.
func seedPriceChangeWhilePending() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000_000_000), // enqueued as big
		priceOp(tokWETH, 1), // $0.01 — now it's tiny
		advanceOp(144),      // ~24h
		checkPendingOp(),    // re-evaluated at new price
	})
}

// seedPriceOscillation: price goes up, transfers enqueue, price goes down, check pending.
func seedPriceOscillation() []fuzzOp {
	return wrapSeed([]fuzzOp{
		priceOp(tokWETH, 10_000_000), // $100,000 per WETH
		msg(tokWETH, emitEth, chainSui, 100_000),
		priceOp(tokWETH, 100), // drop to $1
		msg(tokWETH, emitEth, chainPoly, 100_000),
		checkPendingOp(),
	})
}

// seedMultipleChainsSimultaneous: transfers on all 4 chains.
func seedMultipleChainsSimultaneous() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 563_000),            // ~$10k WETH
		msg(tokSUI, emitSui, chainEth, 1_000_000_000),       // $5k SUI
		msg(tokUSDT_SOL, emitSol, chainPoly, 1_000_000_000), // $10k USDT
		msg(tokDAI, emitPoly, chainEth, 1_000_000_000),      // $10k DAI
	})
}

// seedPolygonLowLimitEnqueue: Polygon has lowest limit (30k). Fill and enqueue.
func seedPolygonLowLimitEnqueue() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokDAI, emitPoly, chainEth, 2_000_000_000), // $20k
		msg(tokDAI, emitPoly, chainSui, 1_500_000_000), // $15k → enqueued (over 30k)
		advanceOp(255),
		checkPendingOp(),
	})
}

// seedSuiChainFillAndRelease: fill Sui limit (50k), enqueue, advance, release.
func seedSuiChainFillAndRelease() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokSUI, emitSui, chainEth, 8_000_000_000),  // $40k at $0.50/token with 8 decimals
		msg(tokSUI, emitSui, chainPoly, 4_000_000_000), // $20k → enqueued
		advanceOp(255),
		checkPendingOp(),
	})
}

// seedDuplicateMessage: same transfer processed twice — second should be no-op or re-publish.
func seedDuplicateMessage() []fuzzOp {
	// Two identical msg ops will get different sequences, so they are NOT true duplicates.
	// But this still exercises the high-volume same-parameters path.
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 500), // small (~$8.87 each after ×1000 scaling)
		msg(tokWETH, emitEth, chainSui, 500),
	})
}

// seedZeroAmountTransfer: transfer with amount 0.
func seedZeroAmountTransfer() []fuzzOp {
	return wrapSeed([]fuzzOp{msg(tokWETH, emitEth, chainSui, 0)})
}

// seedMaxAmountTransfer: transfer with max uint64 amount — tests overflow path.
func seedMaxAmountTransfer() []fuzzOp {
	return wrapSeed([]fuzzOp{msg(tokWETH, emitEth, chainSui, ^uint64(0))})
}

// seedVeryLargeAmountTransfer: very large but not max — near overflow boundary.
func seedVeryLargeAmountTransfer() []fuzzOp {
	return wrapSeed([]fuzzOp{msg(tokWETH, emitEth, chainSui, ^uint64(0)/2)})
}

// seedManySmallTransfers: 20 tiny transfers to accumulate.
func seedManySmallTransfers() []fuzzOp {
	ops := make([]fuzzOp, 20)
	for i := range ops {
		ops[i] = msg(tokUSDT, emitEth, chainSui, 100)
	}
	return wrapSeed(ops)
}

// seedManySmallTransfersExceedLimit: many $10k USDT transfers to exceed Eth 100k limit.
func seedManySmallTransfersExceedLimit() []fuzzOp {
	ops := make([]fuzzOp, 12)
	for i := range ops {
		ops[i] = msg(tokUSDT, emitEth, chainSui, 1_000_000_000) // ~$10k each
	}
	// Last two should get enqueued
	return wrapSeed(ops)
}

// seedEnqueueCheckAdvanceCheckRepeat: multiple rounds of enqueue/check/advance.
func seedEnqueueCheckAdvanceCheckRepeat() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDT, emitEth, chainSui, 10_000_000_000), // $100k, at limit
		msg(tokUSDT, emitEth, chainSui, 5_000_000_000),  // enqueued
		checkPendingOp(), // nothing released yet
		advanceOp(72),    // ~12h
		checkPendingOp(), // still pending
		advanceOp(72),    // ~24h total
		checkPendingOp(), // old transfers expire, pending released
		msg(tokUSDT, emitEth, chainPoly, 3_000_000_000), // new transfer after release
	})
}

// seedFlowCancelReleasePending: pending transfer released because flow cancel frees capacity.
func seedFlowCancelReleasePending() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_ETH, emitEth, chainSui, 9_000_000_000), // $90k of 100k Eth limit
		msg(tokUSDT, emitEth, chainPoly, 2_000_000_000),    // $20k → enqueued (over limit)
		msg(tokUSDC_ETH, emitSui, chainEth, 5_000_000_000), // flow cancel frees $50k on Eth
		checkPendingOp(), // pending $20k should now fit
	})
}

// seedAllTokenTypes: one transfer per token type.
func seedAllTokenTypes() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_SOL, emitSol, chainEth, 1_000_000),
		msg(tokUSDC_ETH, emitEth, chainSui, 1_000_000),
		msg(tokWETH, emitEth, chainPoly, 1_000_000),
		msg(tokUSDT, emitEth, chainSui, 1_000_000),
		msg(tokDAI, emitEth, chainSol, 1_000_000),
		msg(tokUSDT_SOL, emitSol, chainPoly, 1_000_000),
		msg(tokSUI, emitSui, chainEth, 1_000_000),
		msg(tokUNGOVERN, emitEth, chainSui, 1_000_000),
	})
}

// seedMixedGovernedAndUngoverned: interleave governed and ungoverned transfers.
func seedMixedGovernedAndUngoverned() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000),
		msg(tokUNGOVERN, emitEth, chainSui, 1_000_000),
		msg(tokWETH, emitEth, chainPoly, 1_000_000),
		msg(tokUNGOVERN, emitEth, chainPoly, 1_000_000),
	})
}

// seedInvalidEmitterThenValid: invalid emitter followed by valid — exercises both paths in sequence.
func seedInvalidEmitterThenValid() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitInvalid, chainSui, 1_000_000),
		msg(tokWETH, emitEth, chainSui, 1_000_000),
	})
}

// seedTimerResetMultiple: enqueue two transfers, reset timers on both.
func seedTimerResetMultiple() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDT, emitEth, chainSui, 10_000_000_000), // at limit
		msg(tokUSDT, emitEth, chainPoly, 5_000_000_000), // enqueued
		msg(tokUSDT, emitEth, chainSol, 3_000_000_000),  // enqueued
		adminResetTimerOp(0, 5),                         // pending 0 → 5 days
		adminResetTimerOp(1, 1),                         // pending 1 → 1 day
		advanceOp(255),                                  // ~42h
		checkPendingOp(),                                // pending 1 released (1 day), pending 0 still held (5 days)
	})
}

// seedTimerResetToMax: reset timer to 5 days, advance through each day checking.
func seedTimerResetToMax() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000_000_000), // big, enqueued
		adminResetTimerOp(0, 5),                            // 5 days
		advanceOp(255),                                     // ~42h
		checkPendingOp(),                                   // still pending
		advanceOp(255),                                     // ~84h
		checkPendingOp(),                                   // still pending
		advanceOp(255),                                     // ~126h ≈ 5.25 days
		checkPendingOp(),                                   // should release
	})
}

// seedFlowCancelSolanaUSDC: flow cancel with Solana-origin USDC on Eth→Sui corridor.
func seedFlowCancelSolanaUSDC() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_SOL, emitEth, chainSui, 2_000_000_000), // $20k
		msg(tokUSDC_SOL, emitSui, chainEth, 1_500_000_000), // $15k
	})
}

// seedFlowCancelBothUSDCVariants: both USDC_SOL and USDC_ETH in same session.
func seedFlowCancelBothUSDCVariants() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_SOL, emitEth, chainSui, 1_000_000_000), // $10k
		msg(tokUSDC_ETH, emitEth, chainSui, 1_000_000_000), // $10k
		msg(tokUSDC_SOL, emitSui, chainEth, 500_000_000),   // $5k
		msg(tokUSDC_ETH, emitSui, chainEth, 500_000_000),   // $5k
	})
}

// seedAdvanceTimeSmallSteps: many small time advances to test sliding window granularity.
func seedAdvanceTimeSmallSteps() []fuzzOp {
	ops := []fuzzOp{
		msg(tokUSDT, emitEth, chainSui, 5_000_000_000), // $50k
	}
	// 24 advances of ~1h each
	for i := 0; i < 24; i++ {
		ops = append(ops, advanceOp(6)) // 1 + 6*10 = 61 min ≈ 1h
	}
	ops = append(ops, msg(tokUSDT, emitEth, chainSui, 5_000_000_000)) // should fit after window slides
	return wrapSeed(ops)
}

// seedCheckPendingWithNothingPending: check pending when queue is empty.
func seedCheckPendingWithNothingPending() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 100), // published, nothing pending
		checkPendingOp(),
	})
}

// seedCheckPendingTooEarly: enqueue then check before timer — nothing should release.
func seedCheckPendingTooEarly() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000_000_000), // enqueued
		advanceOp(100),   // ~16h, not enough
		checkPendingOp(), // should stay pending
	})
}

// seedCheckPendingExactly24h: advance exactly 24h (144 → 1441 min = 24h 1min).
func seedCheckPendingExactly24h() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000_000_000),
		advanceOp(144), // 1 + 144*10 = 1441 min ≈ 24h 1min
		checkPendingOp(),
	})
}

// seedTransferToSameChain: emitter and target are the same chain.
func seedTransferToSameChain() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainEth, 1_000_000),
	})
}

// seedHighVolumeEnqueueRelease: 10 transfers that fill and overflow, then advance and release all.
func seedHighVolumeEnqueueRelease() []fuzzOp {
	ops := make([]fuzzOp, 0, 14)
	// 10 transfers of ~$15k each on Eth (limit 100k) — first ~6 publish, rest enqueue
	for i := 0; i < 10; i++ {
		ops = append(ops, msg(tokUSDT, emitEth, chainSui, 1_500_000_000))
	}
	ops = append(ops, advanceOp(255))
	ops = append(ops, checkPendingOp())
	ops = append(ops, advanceOp(255)) // ensure all timers expired
	ops = append(ops, checkPendingOp())
	return wrapSeed(ops)
}

// seedPriceZeroCents: set price to minimum ($0.01) and transfer.
func seedPriceZeroCents() []fuzzOp {
	return wrapSeed([]fuzzOp{
		priceOp(tokWETH, 0),                      // clamps to $0.01
		msg(tokWETH, emitEth, chainSui, 563_000), // ~$0.06 at $0.01/WETH
	})
}

// seedPriceVeryHigh: set price extremely high then transfer — tests large USD values.
func seedPriceVeryHigh() []fuzzOp {
	return wrapSeed([]fuzzOp{
		priceOp(tokSUI, 100_000_000_00), // $100M per SUI token
		msg(tokSUI, emitSui, chainEth, 100_000_000),
	})
}

// seedPriceChangeUSDCIgnored: try to change USDC price — should be ignored.
func seedPriceChangeUSDCIgnored() []fuzzOp {
	return wrapSeed([]fuzzOp{
		priceOp(tokUSDC_ETH, 50_000), // attempt $500, should be ignored
		msg(tokUSDC_ETH, emitEth, chainSui, 10_000_000),
	})
}

// seedPriceChangeUngovernedIgnored: try to change ungoverned token price — should be ignored.
func seedPriceChangeUngovernedIgnored() []fuzzOp {
	return wrapSeed([]fuzzOp{
		priceOp(tokUNGOVERN, 100_000),
		msg(tokUNGOVERN, emitEth, chainSui, 1_000_000),
	})
}

// seedAdminReleasePending: enqueue a transfer, then admin-release it immediately.
func seedAdminReleasePending() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000_000_000), // enqueued
		adminReleaseOp(0), // release first pending
	})
}

// seedAdminDropPending: enqueue a transfer, then admin-drop it.
func seedAdminDropPending() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokWETH, emitEth, chainSui, 1_000_000_000_000), // enqueued
		adminDropOp(0), // drop first pending
	})
}

// seedAdminDropThenTransfer: drop a pending transfer, then send a new one that should fit.
func seedAdminDropThenTransfer() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDT, emitEth, chainSui, 9_000_000_000),  // $90k, near limit
		msg(tokUSDT, emitEth, chainPoly, 2_000_000_000), // $20k, enqueued
		adminDropOp(0), // drop the pending
		msg(tokUSDT, emitEth, chainSol, 500_000_000), // $5k, should fit
	})
}

// seedAdminReleaseDoesNotCountTowardLimit: release bypasses daily limit.
func seedAdminReleaseDoesNotCountTowardLimit() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDT, emitEth, chainSui, 9_000_000_000),  // $90k
		msg(tokUSDT, emitEth, chainPoly, 2_000_000_000), // $20k, enqueued
		adminReleaseOp(0), // release — should NOT count toward limit
		msg(tokUSDT, emitEth, chainSol, 500_000_000), // $5k, should still fit in remaining $10k
	})
}

// seedComplexMultiChainFlow: realistic multi-chain scenario with flow cancel and time advancement.
func seedComplexMultiChainFlow() []fuzzOp {
	return wrapSeed([]fuzzOp{
		msg(tokUSDC_ETH, emitEth, chainSui, 40_000_000_000), // $40k Eth→Sui
		msg(tokWETH, emitEth, chainPoly, 1_127_000),         // ~$20k WETH Eth→Poly
		msg(tokSUI, emitSui, chainEth, 3_000_000_000),       // SUI Sui→Eth
		msg(tokUSDC_ETH, emitSui, chainEth, 20_000_000_000), // flow cancel Sui→Eth
		advanceOp(36), // ~6h
		msg(tokUSDT, emitEth, chainSol, 5_000_000_000), // USDT Eth→Sol
		msg(tokDAI, emitPoly, chainEth, 1_000_000_000), // DAI Poly→Eth
		checkPendingOp(),
		advanceOp(108), // ~18h more (24h total)
		checkPendingOp(),
	})
}

// seedLongRunningSession: extended sequence simulating a long fuzzing session.
func seedLongRunningSession() []fuzzOp {
	ops := []fuzzOp{
		// Day 1: fill Eth chain
		msg(tokWETH, emitEth, chainSui, 1_127_000),      // ~$20k WETH
		msg(tokUSDT, emitEth, chainPoly, 3_000_000_000), // $30k
		msg(tokDAI, emitEth, chainSol, 2_000_000_000),   // $20k
		advanceOp(36), // 6h
		msg(tokWETH, emitEth, chainPoly, 563_000), // ~$10k WETH
		// Day 1.5: check pending, advance
		advanceOp(72), // 12h more
		checkPendingOp(),
		// Day 2: old transfers start expiring
		advanceOp(72), // 12h more (~30h from start)
		msg(tokUSDC_ETH, emitEth, chainSui, 5_000_000_000),
		msg(tokUSDC_ETH, emitSui, chainEth, 3_000_000_000), // flow cancel
		checkPendingOp(),
		// Day 3: price change
		priceOp(tokWETH, 300_000),                // $3000 per WETH
		advanceOp(144),                           // ~24h
		msg(tokWETH, emitEth, chainSui, 282_000), // ~$5k WETH
		checkPendingOp(),
	}
	return wrapSeed(ops)
}

// seedPriceSpikeOverflowBlocksPending: a price spike on one chain's pending
// transfer causes scaledUsdValue to overflow uint64, which makes
// checkPendingForTime return an error before processing later chains.
// This leaves pending transfers on other chains stuck indefinitely.
//
// This seed intentionally avoids wrapSeed because the suffix's adminDropOp
// would accidentally clean up the stuck polygon entry, masking the bug.
func seedPriceSpikeOverflowBlocksPending() []fuzzOp {
	return []fuzzOp{
		// Enqueue a big USDT_SOL transfer on ethereum (will overflow after price spike).
		msg(tokUSDT_SOL, emitEth, chainEth, ^uint64(0)/1000), // max amount, big transfer
		// Enqueue a big USDC_SOL transfer on polygon (innocent bystander).
		msg(tokUSDC_SOL, emitPoly, chainEth, ^uint64(0)/1000), // max amount, big transfer
		// Advance past the 24h release timer.
		advanceOp(255), // ~42h
		// Spike USDT_SOL price so scaledUsdValue overflows uint64 on re-evaluation.
		priceOp(tokUSDT_SOL, 3_472_328_296_227_680_400),
		// checkPending: governor errors on ethereum's USDT_SOL, never reaches polygon.
		checkPendingOp(),
	}
}

// seedEmptyInput: no operations, just invariant checks on fresh governor.
func seedEmptyInput() []fuzzOp {
	return wrapSeed([]fuzzOp{})
}

// seedLimitDropReleasesEnqueued: a transfer is enqueued because it exceeds
// the daily limit, then the limit is raised so that checkPending releases it.
func seedLimitDropReleasesEnqueued() []fuzzOp {
	return wrapSeed([]fuzzOp{
		// Ethereum default: dailyLimit=100k, bigTxSize=50k.
		// Send 60k which is under big but over a tight limit.
		msg(tokWETH, emitEth, chainSui, 60_000),
		checkPendingOp(), // released (under 100k limit)
		// Now tighten the limit to 5k daily / 1k big (idx 1)
		limitOp(chainEth, 1),
		// Send something that definitely exceeds the new limit:
		msg(tokUSDT, emitEth, chainSui, 6_000),
		checkPendingOp(), // should be enqueued (6k > 5k daily limit)
		// Raise the limit back to 100k (idx 4)
		limitOp(chainEth, 4),
		checkPendingOp(), // should now release
	})
}

// seedLimitChangeAffectsBigTxThreshold: a transfer is not "big" under the
// default threshold but becomes "big" after lowering bigTxSize.
func seedLimitChangeAffectsBigTxThreshold() []fuzzOp {
	return wrapSeed([]fuzzOp{
		// Ethereum default: bigTxSize=50k. Send 30k — not big, fits under daily.
		msg(tokWETH, emitEth, chainSui, 30_000),
		checkPendingOp(),
		// Now set bigTxSize to 1k (idx 1: dailyLimit=5000, bigTxSize=1000)
		limitOp(chainEth, 1),
		// Send 2k — now "big" under the new threshold, always enqueued for 24h.
		msg(tokUSDT, emitEth, chainSui, 2_000),
		checkPendingOp(),
		advanceOp(144),   // ~24h
		checkPendingOp(), // big transfer timer should expire
	})
}

// seedZeroLimitsEnqueueEverything: set limits to zero/near-zero,
// verify that everything gets enqueued.
func seedZeroLimitsEnqueueEverything() []fuzzOp {
	return wrapSeed([]fuzzOp{
		// idx 9: dailyLimit=1, bigTxSize=1 — near-zero
		limitOp(chainEth, 9),
		limitOp(chainSui, 9),
		msg(tokWETH, emitEth, chainSui, 100),
		msg(tokSUI, emitSui, chainEth, 100),
		checkPendingOp(),
		// Restore normal limits and release
		limitOp(chainEth, 4), // 100k/50k
		limitOp(chainSui, 3), // 50k/25k
		advanceOp(144),       // ~24h
		checkPendingOp(),
	})
}

// =============================================================================
// FuzzGovernor
// =============================================================================

func FuzzGovernor(f *testing.F) {
	f.Add(encodeFuzzOps(seedSmallTransferPublished()))              // 1
	f.Add(encodeFuzzOps(seedSmallTransferEachChain()))              // 2
	f.Add(encodeFuzzOps(seedBigTransferEnqueuedAndAutoReleased()))  // 3
	f.Add(encodeFuzzOps(seedFlowCancelFullOffset()))                // 4
	f.Add(encodeFuzzOps(seedFlowCancelPartialOffset()))             // 5
	f.Add(encodeFuzzOps(seedFlowCancelExceedsOutbound()))           // 6
	f.Add(encodeFuzzOps(seedFlowCancelThenNewTransfer()))           // 7
	f.Add(encodeFuzzOps(seedFlowCancelWrongCorridor()))             // 8
	f.Add(encodeFuzzOps(seedFlowCancelNonFlowToken()))              // 9
	f.Add(encodeFuzzOps(seedEnqueueResetTimerAndRelease()))         // 10
	f.Add(encodeFuzzOps(seedEnqueueResetTimerShort()))              // 11
	f.Add(encodeFuzzOps(seedTransfersUpToLimit()))                  // 12
	f.Add(encodeFuzzOps(seedTransfersOverLimit()))                  // 13
	f.Add(encodeFuzzOps(seedTransfersOverLimitThenExpire()))        // 14
	f.Add(encodeFuzzOps(seedMultiplePendingReleaseOrder()))         // 15
	f.Add(encodeFuzzOps(seedBigTransferThresholdBoundary()))        // 16
	f.Add(encodeFuzzOps(seedBigTransferJustUnder()))                // 17
	f.Add(encodeFuzzOps(seedBigTransferJustOver()))                 // 18
	f.Add(encodeFuzzOps(seedInvalidEmitter()))                      // 19
	f.Add(encodeFuzzOps(seedUngovernedToken()))                     // 20
	f.Add(encodeFuzzOps(seedPriceDropReleasesEnqueued()))           // 21
	f.Add(encodeFuzzOps(seedPriceSpikeEnqueuesMore()))              // 22
	f.Add(encodeFuzzOps(seedPriceChangeWhilePending()))             // 23
	f.Add(encodeFuzzOps(seedPriceOscillation()))                    // 24
	f.Add(encodeFuzzOps(seedMultipleChainsSimultaneous()))          // 25
	f.Add(encodeFuzzOps(seedPolygonLowLimitEnqueue()))              // 26
	f.Add(encodeFuzzOps(seedSuiChainFillAndRelease()))              // 27
	f.Add(encodeFuzzOps(seedDuplicateMessage()))                    // 28
	f.Add(encodeFuzzOps(seedZeroAmountTransfer()))                  // 29
	f.Add(encodeFuzzOps(seedMaxAmountTransfer()))                   // 30
	f.Add(encodeFuzzOps(seedVeryLargeAmountTransfer()))             // 31
	f.Add(encodeFuzzOps(seedManySmallTransfers()))                  // 32
	f.Add(encodeFuzzOps(seedManySmallTransfersExceedLimit()))       // 33
	f.Add(encodeFuzzOps(seedEnqueueCheckAdvanceCheckRepeat()))      // 34
	f.Add(encodeFuzzOps(seedFlowCancelReleasePending()))            // 35
	f.Add(encodeFuzzOps(seedAllTokenTypes()))                       // 36
	f.Add(encodeFuzzOps(seedMixedGovernedAndUngoverned()))          // 37
	f.Add(encodeFuzzOps(seedInvalidEmitterThenValid()))             // 38
	f.Add(encodeFuzzOps(seedTimerResetMultiple()))                  // 39
	f.Add(encodeFuzzOps(seedTimerResetToMax()))                     // 40
	f.Add(encodeFuzzOps(seedFlowCancelSolanaUSDC()))                // 41
	f.Add(encodeFuzzOps(seedFlowCancelBothUSDCVariants()))          // 42
	f.Add(encodeFuzzOps(seedAdvanceTimeSmallSteps()))               // 43
	f.Add(encodeFuzzOps(seedCheckPendingWithNothingPending()))      // 44
	f.Add(encodeFuzzOps(seedCheckPendingTooEarly()))                // 45
	f.Add(encodeFuzzOps(seedCheckPendingExactly24h()))              // 46
	f.Add(encodeFuzzOps(seedTransferToSameChain()))                 // 47
	f.Add(encodeFuzzOps(seedHighVolumeEnqueueRelease()))            // 48
	f.Add(encodeFuzzOps(seedPriceZeroCents()))                      // 49
	f.Add(encodeFuzzOps(seedPriceVeryHigh()))                       // 50
	f.Add(encodeFuzzOps(seedPriceChangeUSDCIgnored()))              // 51
	f.Add(encodeFuzzOps(seedPriceChangeUngovernedIgnored()))        // 52
	f.Add(encodeFuzzOps(seedComplexMultiChainFlow()))               // 53
	f.Add(encodeFuzzOps(seedLongRunningSession()))                  // 54
	f.Add(encodeFuzzOps(seedAdminReleasePending()))                 // 55
	f.Add(encodeFuzzOps(seedAdminDropPending()))                    // 56
	f.Add(encodeFuzzOps(seedAdminDropThenTransfer()))               // 57
	f.Add(encodeFuzzOps(seedAdminReleaseDoesNotCountTowardLimit())) // 58
	// f.Add(encodeFuzzOps(seedPriceSpikeOverflowBlocksPending()))   // 59 — disabled: price wrap prevents overflow; kept for reproducing the bug without the wrap
	f.Add(encodeFuzzOps(seedEmptyInput()))                       // 60
	f.Add(encodeFuzzOps(seedLimitDropReleasesEnqueued()))        // 61
	f.Add(encodeFuzzOps(seedLimitChangeAffectsBigTxThreshold())) // 62
	f.Add(encodeFuzzOps(seedZeroLimitsEnqueueEverything()))      // 63

	f.Fuzz(func(t *testing.T, data []byte) {
		gov := newFuzzGovernor(t)
		tracker := newInvariantTracker()
		r := &fuzzReader{data: data}
		now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
		var sequence uint64

		for r.remaining() >= fuzzBytesPerOp {
			op := r.readByte() % fuzzOpCount

			switch op {
			case opProcessMsg:
				tokenIdx := int(r.readByte()) % len(fuzzTokenTable)
				emitterIdx := int(r.readByte()) % len(fuzzEmitterTable)
				targetChainIdx := int(r.readByte()) % len(fuzzChainTable)
				amount := r.readUint64LE()

				// Wrap amount to 100x the max daily limit ($100k) to prevent
				// unrealistic transfers that overflow scaledUsdValue.
				amount = amount%(100*100_000) + 1 // 1 .. 10,000,000

				// Scale amount by 1000 so that seed values like 5_000_000_000
				// map to $50k (not $50) for tokens with 8 decimals at $1.
				scaled := amount * 1000

				sequence++
				msg := buildFuzzMsg(sequence, now, emitterIdx, tokenIdx, targetChainIdx, scaled)

				// Build the amount as big.Int the same way the payload does
				payloadAmount := new(big.Int).SetUint64(scaled)

				// Check queuing invariant BEFORE calling processMsgForTime,
				// since processMsgForTime has side effects (trimming transfers).
				expectEnqueue, applicable := tracker.shouldEnqueue(tokenIdx, emitterIdx, targetChainIdx, payloadAmount, now)

				published, err := gov.processMsgForTime(&msg, now)
				if err != nil {
					if *fuzzVerboseGov {
						t.Logf("TRACE opProcessMsg: seq=%d token=%s emitter=%s target=%s amount=%d scaled=%d err=%v now=%v",
							sequence, fuzzTokenTable[tokenIdx].symbol, fuzzEmitterTable[emitterIdx].chainID,
							fuzzChainTable[targetChainIdx].chainID, amount, scaled, err, now)
					}
					continue
				}

				scaledValue, _ := tracker.computeScaledValue(tokenIdx, payloadAmount)
				governed := tracker.isGoverned(tokenIdx, emitterIdx)
				isBig := false
				if governed {
					isBig = tracker.isBigTransfer(fuzzEmitterTable[emitterIdx].chainID, scaledValue)
				}
				if *fuzzVerboseGov {
					t.Logf("TRACE opProcessMsg: seq=%d token=%s emitter=%s target=%s amount=%d scaledValue=%d published=%v expectEnqueue=%v applicable=%v governed=%v isBig=%v now=%v",
						sequence, fuzzTokenTable[tokenIdx].symbol, fuzzEmitterTable[emitterIdx].chainID,
						fuzzChainTable[targetChainIdx].chainID, scaled, scaledValue, published, expectEnqueue, applicable, governed, isBig, now)
				}

				// Verify queuing decision matches our expectation
				if applicable {
					tracker.checkQueuingResult(t, tokenIdx, emitterIdx, payloadAmount, now, published, expectEnqueue)
				}

				// Update shadow state based on what happened
				if tracker.isGoverned(tokenIdx, emitterIdx) {
					if published {
						tracker.recordPublish(tokenIdx, emitterIdx, targetChainIdx, payloadAmount, now)
					} else {
						tracker.recordEnqueue(tokenIdx, emitterIdx, targetChainIdx, payloadAmount, now, msg.MessageIDString())
					}
				}

			case opCheckPending:
				r.discard(fuzzPayloadBytes)
				if *fuzzVerboseGov {
					t.Logf("TRACE opCheckPending: now=%v shadowPending=%d", now, len(tracker.pending))
				}
				gov.checkPendingForTime(now) //nolint:errcheck
				tracker.shadowCheckPending(t, now)

			case opAdmin:
				pendingIdxByte := r.readByte()
				actionByte := r.readByte()
				_ = r.readByte() // targetChainIdx position, unused
				numDaysRaw := r.readUint64LE()
				// 1 + 1 + 1 + 8 = 11 payload bytes consumed

				// Look up the Nth pending transfer directly from the governor.
				gov.mutex.Lock()
				var allPendingIDs []string
				for _, ce := range gov.chains {
					for _, pe := range ce.pending {
						allPendingIDs = append(allPendingIDs, pe.dbData.Msg.MessageIDString())
					}
				}
				gov.mutex.Unlock()

				if len(allPendingIDs) == 0 {
					if *fuzzVerboseGov {
						t.Logf("TRACE opAdmin: no pending transfers, skipping")
					}
					continue
				}
				vaaId := allPendingIDs[int(pendingIdxByte)%len(allPendingIDs)]

				action := actionByte % adminActionCount
				actionNames := []string{"release", "drop", "resetTimer"}
				switch action {
				case adminRelease:
					if *fuzzVerboseGov {
						t.Logf("TRACE opAdmin: action=%s vaaId=%s now=%v", actionNames[action], vaaId, now)
					}
					gov.ReleasePendingVAA(vaaId) //nolint:errcheck
					tracker.recordAdminRelease(vaaId)
				case adminDrop:
					if *fuzzVerboseGov {
						t.Logf("TRACE opAdmin: action=%s vaaId=%s now=%v", actionNames[action], vaaId, now)
					}
					gov.DropPendingVAA(vaaId) //nolint:errcheck
					tracker.recordAdminDrop(vaaId)
				case adminResetTimer:
					numDays := uint32(1 + numDaysRaw%5)
					if *fuzzVerboseGov {
						t.Logf("TRACE opAdmin: action=%s vaaId=%s numDays=%d now=%v", actionNames[action], vaaId, numDays, now)
					}
					gov.resetReleaseTimerForTime(vaaId, now, numDays) //nolint:errcheck
					tracker.recordAdminResetTimer(vaaId, now, numDays)
				}

			case opAdvanceTime:
				minutesByte := r.readByte()
				r.discard(fuzzPayloadBytes - 1)

				minutes := 1 + int(minutesByte)*10
				now = now.Add(time.Duration(minutes) * time.Minute)
				if *fuzzVerboseGov {
					t.Logf("TRACE opAdvanceTime: +%d min, now=%v", minutes, now)
				}

			case opChangeTokenPrice:
				tokenIdxByte := r.readByte()
				_ = r.readByte() // unused (emitterIdx position)
				_ = r.readByte() // unused (targetChainIdx position)
				priceRaw := r.readUint64LE()
				// 1 + 1 + 1 + 8 = 11 payload bytes consumed

				tokenIdx := int(tokenIdxByte) % len(fuzzTokenTable)
				// USDC tokens (idx 0, 1) must always remain at $1.0
				if tokenIdx <= 1 {
					continue
				}
				// Ungoverned token has no tokenEntry
				if !fuzzTokenTable[tokenIdx].governed {
					continue
				}

				// Interpret amount as price in cents (divide by 100), wrapped to
				// $0.01–$1,000,000. Modulo preserves uniform distribution from the
				// fuzzer. Without this, extreme prices cause scaledUsdValue to overflow
				// uint64, masking real governor bugs behind arithmetic errors.
				priceCents := priceRaw%(100_000_000) + 1 // 1 cent .. $1,000,000
				newPrice := float64(priceCents) / 100.0
				tk := tokenKey{chain: fuzzTokenTable[tokenIdx].originChain, addr: fuzzTokenAddrs[tokenIdx]}

				gov.mutex.Lock()
				if te, exists := gov.tokens[tk]; exists {
					te.price.Set(big.NewFloat(newPrice))
				}
				gov.mutex.Unlock()

				tracker.prices[tokenIdx] = newPrice
				if *fuzzVerboseGov {
					t.Logf("TRACE opChangeTokenPrice: token=%s newPrice=%.2f now=%v", fuzzTokenTable[tokenIdx].symbol, newPrice, now)
				}

			case opChangeGovLimit:
				chainIdxByte := r.readByte()
				limitIdxByte := r.readByte()
				r.discard(fuzzPayloadBytes - 2)

				chainIdx := int(chainIdxByte) % len(fuzzChainTable)
				limitIdx := int(limitIdxByte) % len(fuzzLimitTable)
				newLimit := fuzzLimitTable[limitIdx]
				chainID := fuzzChainTable[chainIdx].chainID

				// Update the governor's chain entry.
				gov.mutex.Lock()
				if ce, exists := gov.chains[chainID]; exists {
					ce.dailyLimit = newLimit.dailyLimit
					ce.bigTransactionSize = newLimit.bigTxSize
					ce.checkForBigTransactions = newLimit.bigTxSize != 0
				}
				gov.mutex.Unlock()

				// Update shadow tracker limits.
				tracker.chainLimits[chainID] = newLimit

				if *fuzzVerboseGov {
					t.Logf("TRACE opChangeGovLimit: chain=%s dailyLimit=%d bigTxSize=%d (tableIdx=%d) now=%v",
						chainID, newLimit.dailyLimit, newLimit.bigTxSize, limitIdx, now)
				}
			}
		}

		checkGovernorInvariants(t, gov, tracker, now)
	})
}

// =============================================================================
// DB round-trip fuzzer — snapshot types and comparison
// =============================================================================

// snapshotTransfer captures the fields of a transfer for comparison.
// Timestamps are truncated to Unix seconds because DB serialization
// stores them as uint32.
type snapshotTransfer struct {
	Timestamp      int64
	ScaledValue    int64
	OriginChain    vaa.ChainID
	OriginAddress  vaa.Address
	EmitterChain   vaa.ChainID
	EmitterAddress vaa.Address
	MsgID          string
	Hash           string
	TargetChain    vaa.ChainID
	TargetAddress  vaa.Address
}

type snapshotPending struct {
	ReleaseTime int64
	MsgID       string
	Hash        string
	Amount      string
	TokenChain  vaa.ChainID
	TokenAddr   vaa.Address
}

type chainSnapshot struct {
	Transfers []snapshotTransfer
	Pending   []snapshotPending
}

type govSnapshot struct {
	Chains   map[vaa.ChainID]chainSnapshot
	MsgsSeen map[string]bool
}

// takeGovSnapshot captures the governor's state within the active 24h window.
// Transfers with timestamps before the window are excluded — they're expired
// and don't affect decisions. This matters because trimAndSumValue runs
// per-chain lazily, so one chain's trim can delete a DB record while the
// flow-cancel entry on another chain lingers in memory until that chain
// is trimmed. Filtering by the window ensures we compare only the state
// that a DB reload can reconstruct.
// takeGovSnapshot captures governor state within the window [startTime, ∞).
// The caller provides startTime so that both the original and reloaded
// governors are filtered with the same 24h window.
func takeGovSnapshot(gov *ChainGovernor, startTime time.Time) govSnapshot {
	gov.mutex.Lock()
	defer gov.mutex.Unlock()

	snap := govSnapshot{
		Chains:   make(map[vaa.ChainID]chainSnapshot),
		MsgsSeen: make(map[string]bool),
	}

	for chainID, ce := range gov.chains {
		cs := chainSnapshot{}
		for _, tr := range ce.transfers {
			if tr.dbTransfer.Timestamp.Before(startTime) {
				continue
			}
			cs.Transfers = append(cs.Transfers, snapshotTransfer{
				Timestamp:      tr.dbTransfer.Timestamp.Unix(),
				ScaledValue:    tr.scaledValue,
				OriginChain:    tr.dbTransfer.OriginChain,
				OriginAddress:  tr.dbTransfer.OriginAddress,
				EmitterChain:   tr.dbTransfer.EmitterChain,
				EmitterAddress: tr.dbTransfer.EmitterAddress,
				MsgID:          tr.dbTransfer.MsgID,
				Hash:           tr.dbTransfer.Hash,
				TargetChain:    tr.dbTransfer.TargetChain,
				TargetAddress:  tr.dbTransfer.TargetAddress,
			})
		}
		for _, pe := range ce.pending {
			cs.Pending = append(cs.Pending, snapshotPending{
				ReleaseTime: pe.dbData.ReleaseTime.Unix(),
				MsgID:       pe.dbData.Msg.MessageIDString(),
				Hash:        pe.hash,
				Amount:      pe.amount.String(),
				TokenChain:  pe.token.token.chain,
				TokenAddr:   pe.token.token.addr,
			})
		}
		snap.Chains[chainID] = cs
	}

	for hash, complete := range gov.msgsSeen {
		snap.MsgsSeen[hash] = complete
	}

	return snap
}

// transferSortKey returns a deterministic key for ordering transfers.
// Same-timestamp entries may be interleaved differently after a DB reload
// (flow-cancel entries vs direct transfers), but this is benign since
// trimAndSumValue only cares about the timestamp boundary, not same-timestamp order.
func transferSortKey(t snapshotTransfer) string {
	return fmt.Sprintf("%d:%s:%d", t.Timestamp, t.MsgID, t.ScaledValue)
}

func compareGovSnapshots(t *testing.T, original, reloaded govSnapshot) {
	t.Helper()

	for chainID, origCS := range original.Chains {
		reloadCS, ok := reloaded.Chains[chainID]
		if !ok {
			t.Errorf("chain %s missing from reloaded governor", chainID)
			continue
		}

		// Sort transfers before comparing — reload may interleave
		// same-timestamp entries differently than the original.
		sort.Slice(origCS.Transfers, func(i, j int) bool {
			return transferSortKey(origCS.Transfers[i]) < transferSortKey(origCS.Transfers[j])
		})
		sort.Slice(reloadCS.Transfers, func(i, j int) bool {
			return transferSortKey(reloadCS.Transfers[i]) < transferSortKey(reloadCS.Transfers[j])
		})

		if len(origCS.Transfers) != len(reloadCS.Transfers) {
			t.Errorf("chain %s transfer count mismatch: original=%d reloaded=%d",
				chainID, len(origCS.Transfers), len(reloadCS.Transfers))
		} else {
			for i, origTr := range origCS.Transfers {
				reloadTr := reloadCS.Transfers[i]
				if origTr != reloadTr {
					t.Errorf("chain %s transfer[%d] mismatch:\n  original: %+v\n  reloaded: %+v",
						chainID, i, origTr, reloadTr)
				}
			}
		}

		// Sort pending by release time then MsgID.
		sort.Slice(origCS.Pending, func(i, j int) bool {
			if origCS.Pending[i].ReleaseTime != origCS.Pending[j].ReleaseTime {
				return origCS.Pending[i].ReleaseTime < origCS.Pending[j].ReleaseTime
			}
			return origCS.Pending[i].MsgID < origCS.Pending[j].MsgID
		})
		sort.Slice(reloadCS.Pending, func(i, j int) bool {
			if reloadCS.Pending[i].ReleaseTime != reloadCS.Pending[j].ReleaseTime {
				return reloadCS.Pending[i].ReleaseTime < reloadCS.Pending[j].ReleaseTime
			}
			return reloadCS.Pending[i].MsgID < reloadCS.Pending[j].MsgID
		})

		if len(origCS.Pending) != len(reloadCS.Pending) {
			t.Errorf("chain %s pending count mismatch: original=%d reloaded=%d",
				chainID, len(origCS.Pending), len(reloadCS.Pending))
		} else {
			for i, origPe := range origCS.Pending {
				reloadPe := reloadCS.Pending[i]
				if origPe != reloadPe {
					t.Errorf("chain %s pending[%d] mismatch:\n  original: %+v\n  reloaded: %+v",
						chainID, i, origPe, reloadPe)
				}
			}
		}
	}

	// Every entry in reloaded msgsSeen must exist in original with the same value.
	// Original may have extra stale entries from admin release/drop.
	for hash, reloadedComplete := range reloaded.MsgsSeen {
		origComplete, exists := original.MsgsSeen[hash]
		if !exists {
			t.Errorf("reloaded msgsSeen has hash %s but original does not", hash)
		} else if origComplete != reloadedComplete {
			t.Errorf("msgsSeen[%s] mismatch: original=%v reloaded=%v", hash, origComplete, reloadedComplete)
		}
	}
}

// =============================================================================
// FuzzGovernorDBRoundTrip
// =============================================================================

func FuzzGovernorDBRoundTrip(f *testing.F) {
	f.Add(encodeFuzzOps(seedSmallTransferPublished()))              // 1
	f.Add(encodeFuzzOps(seedSmallTransferEachChain()))              // 2
	f.Add(encodeFuzzOps(seedBigTransferEnqueuedAndAutoReleased()))  // 3
	f.Add(encodeFuzzOps(seedFlowCancelFullOffset()))                // 4
	f.Add(encodeFuzzOps(seedFlowCancelPartialOffset()))             // 5
	f.Add(encodeFuzzOps(seedFlowCancelExceedsOutbound()))           // 6
	f.Add(encodeFuzzOps(seedFlowCancelThenNewTransfer()))           // 7
	f.Add(encodeFuzzOps(seedFlowCancelWrongCorridor()))             // 8
	f.Add(encodeFuzzOps(seedFlowCancelNonFlowToken()))              // 9
	f.Add(encodeFuzzOps(seedEnqueueResetTimerAndRelease()))         // 10
	f.Add(encodeFuzzOps(seedEnqueueResetTimerShort()))              // 11
	f.Add(encodeFuzzOps(seedTransfersUpToLimit()))                  // 12
	f.Add(encodeFuzzOps(seedTransfersOverLimit()))                  // 13
	f.Add(encodeFuzzOps(seedTransfersOverLimitThenExpire()))        // 14
	f.Add(encodeFuzzOps(seedMultiplePendingReleaseOrder()))         // 15
	f.Add(encodeFuzzOps(seedBigTransferThresholdBoundary()))        // 16
	f.Add(encodeFuzzOps(seedBigTransferJustUnder()))                // 17
	f.Add(encodeFuzzOps(seedBigTransferJustOver()))                 // 18
	f.Add(encodeFuzzOps(seedInvalidEmitter()))                      // 19
	f.Add(encodeFuzzOps(seedUngovernedToken()))                     // 20
	f.Add(encodeFuzzOps(seedPriceDropReleasesEnqueued()))           // 21
	f.Add(encodeFuzzOps(seedPriceSpikeEnqueuesMore()))              // 22
	f.Add(encodeFuzzOps(seedPriceChangeWhilePending()))             // 23
	f.Add(encodeFuzzOps(seedPriceOscillation()))                    // 24
	f.Add(encodeFuzzOps(seedMultipleChainsSimultaneous()))          // 25
	f.Add(encodeFuzzOps(seedPolygonLowLimitEnqueue()))              // 26
	f.Add(encodeFuzzOps(seedSuiChainFillAndRelease()))              // 27
	f.Add(encodeFuzzOps(seedDuplicateMessage()))                    // 28
	f.Add(encodeFuzzOps(seedZeroAmountTransfer()))                  // 29
	f.Add(encodeFuzzOps(seedMaxAmountTransfer()))                   // 30
	f.Add(encodeFuzzOps(seedVeryLargeAmountTransfer()))             // 31
	f.Add(encodeFuzzOps(seedManySmallTransfers()))                  // 32
	f.Add(encodeFuzzOps(seedManySmallTransfersExceedLimit()))       // 33
	f.Add(encodeFuzzOps(seedEnqueueCheckAdvanceCheckRepeat()))      // 34
	f.Add(encodeFuzzOps(seedFlowCancelReleasePending()))            // 35
	f.Add(encodeFuzzOps(seedAllTokenTypes()))                       // 36
	f.Add(encodeFuzzOps(seedMixedGovernedAndUngoverned()))          // 37
	f.Add(encodeFuzzOps(seedInvalidEmitterThenValid()))             // 38
	f.Add(encodeFuzzOps(seedTimerResetMultiple()))                  // 39
	f.Add(encodeFuzzOps(seedTimerResetToMax()))                     // 40
	f.Add(encodeFuzzOps(seedFlowCancelSolanaUSDC()))                // 41
	f.Add(encodeFuzzOps(seedFlowCancelBothUSDCVariants()))          // 42
	f.Add(encodeFuzzOps(seedAdvanceTimeSmallSteps()))               // 43
	f.Add(encodeFuzzOps(seedCheckPendingWithNothingPending()))      // 44
	f.Add(encodeFuzzOps(seedCheckPendingTooEarly()))                // 45
	f.Add(encodeFuzzOps(seedCheckPendingExactly24h()))              // 46
	f.Add(encodeFuzzOps(seedTransferToSameChain()))                 // 47
	f.Add(encodeFuzzOps(seedHighVolumeEnqueueRelease()))            // 48
	f.Add(encodeFuzzOps(seedPriceZeroCents()))                      // 49
	f.Add(encodeFuzzOps(seedPriceVeryHigh()))                       // 50
	f.Add(encodeFuzzOps(seedPriceChangeUSDCIgnored()))              // 51
	f.Add(encodeFuzzOps(seedPriceChangeUngovernedIgnored()))        // 52
	f.Add(encodeFuzzOps(seedComplexMultiChainFlow()))               // 53
	f.Add(encodeFuzzOps(seedLongRunningSession()))                  // 54
	f.Add(encodeFuzzOps(seedAdminReleasePending()))                 // 55
	f.Add(encodeFuzzOps(seedAdminDropPending()))                    // 56
	f.Add(encodeFuzzOps(seedAdminDropThenTransfer()))               // 57
	f.Add(encodeFuzzOps(seedAdminReleaseDoesNotCountTowardLimit())) // 58
	f.Add(encodeFuzzOps(seedEmptyInput()))                          // 60
	f.Add(encodeFuzzOps(seedLimitDropReleasesEnqueued()))           // 61
	f.Add(encodeFuzzOps(seedLimitChangeAffectsBigTxThreshold()))    // 62
	f.Add(encodeFuzzOps(seedZeroLimitsEnqueueEverything()))         // 63

	dbPool := sync.Pool{
		New: func() any {
			return guardianDB.OpenDb(zap.NewNop(), nil)
		},
	}
	f.Cleanup(func() {
		// Drain is best-effort; Pool doesn't expose iteration,
		// but GC will finalize any remaining instances.
	})

	f.Fuzz(func(t *testing.T, data []byte) {
		// Cap input size to keep each execution fast. 50 ops × 12 bytes
		// covers more than the largest seed while preventing the fuzzer
		// from generating inputs with thousands of DB writes.
		const maxBytes = 50 * fuzzBytesPerOp
		if len(data) > maxBytes {
			data = data[:maxBytes]
		}

		realDB := dbPool.Get().(*guardianDB.Database)
		defer func() {
			realDB.DropAll() //nolint:errcheck
			dbPool.Put(realDB)
		}()

		gov := newFuzzGovernorWithDB(t, realDB)

		r := &fuzzReader{data: data}
		now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
		var sequence uint64

		// checkDBRoundTrip creates a fresh governor from the same DB and
		// verifies its state matches the original. The original gov is
		// never replaced — it remains the continuous source of truth.
		//
		// Both snapshots filter to the original's 24h window so we only
		// compare state that actually drives decisions and that the DB
		// can reconstruct. Expired flow-cancel entries may linger in
		// memory on one chain after the source transfer was deleted by
		// another chain's trim — filtering by the window excludes these.
		checkDBRoundTrip := func(label string) {
			t.Helper()
			// Use the original governor's 24h window for both snapshots
			// so they filter identically.
			startTime := now.Add(-time.Minute * time.Duration(gov.dayLengthInMinutes))
			originalSnap := takeGovSnapshot(gov, startTime)

			reloadedGov := newFuzzGovernorWithDB(t, realDB)
			// Use a very large day length so loadFromDB doesn't prune
			// transfers whose timestamps are in simulated past (2024)
			// vs real time.Now().
			reloadedGov.setDayLengthInMinutes(525600 * 3) // 3 years
			err := reloadedGov.loadFromDB()
			require.NoError(t, err, "loadFromDB failed at %s", label)

			reloadedSnap := takeGovSnapshot(reloadedGov, startTime)
			compareGovSnapshots(t, originalSnap, reloadedSnap)
		}
		var opCount int

		for r.remaining() >= fuzzBytesPerOp {
			op := r.readByte() % fuzzOpCount

			switch op {
			case opProcessMsg:
				tokenIdx := int(r.readByte()) % len(fuzzTokenTable)
				emitterIdx := int(r.readByte()) % len(fuzzEmitterTable)
				targetChainIdx := int(r.readByte()) % len(fuzzChainTable)
				amount := r.readUint64LE()

				amount = amount%(100*100_000) + 1
				scaled := amount * 1000

				sequence++
				msg := buildFuzzMsg(sequence, now, emitterIdx, tokenIdx, targetChainIdx, scaled)
				gov.processMsgForTime(&msg, now) //nolint:errcheck

			case opCheckPending:
				r.discard(fuzzPayloadBytes)
				gov.checkPendingForTime(now) //nolint:errcheck

			case opAdmin:
				pendingIdxByte := r.readByte()
				actionByte := r.readByte()
				_ = r.readByte()
				numDaysRaw := r.readUint64LE()

				gov.mutex.Lock()
				var allPendingIDs []string
				for _, ce := range gov.chains {
					for _, pe := range ce.pending {
						allPendingIDs = append(allPendingIDs, pe.dbData.Msg.MessageIDString())
					}
				}
				gov.mutex.Unlock()

				if len(allPendingIDs) == 0 {
					continue
				}
				vaaId := allPendingIDs[int(pendingIdxByte)%len(allPendingIDs)]

				switch actionByte % adminActionCount {
				case adminRelease:
					gov.ReleasePendingVAA(vaaId) //nolint:errcheck
				case adminDrop:
					gov.DropPendingVAA(vaaId) //nolint:errcheck
				case adminResetTimer:
					numDays := uint32(1 + numDaysRaw%5)
					gov.resetReleaseTimerForTime(vaaId, now, numDays) //nolint:errcheck
				}

			case opAdvanceTime:
				minutesByte := r.readByte()
				r.discard(fuzzPayloadBytes - 1)
				minutes := 1 + int(minutesByte)*10
				now = now.Add(time.Duration(minutes) * time.Minute)

			case opChangeTokenPrice:
				tokenIdxByte := r.readByte()
				_ = r.readByte()
				_ = r.readByte()
				priceRaw := r.readUint64LE()

				tokenIdx := int(tokenIdxByte) % len(fuzzTokenTable)
				if tokenIdx <= 1 || !fuzzTokenTable[tokenIdx].governed {
					continue
				}

				priceCents := priceRaw%(100_000_000) + 1
				newPrice := float64(priceCents) / 100.0
				tk := tokenKey{chain: fuzzTokenTable[tokenIdx].originChain, addr: fuzzTokenAddrs[tokenIdx]}

				gov.mutex.Lock()
				if te, exists := gov.tokens[tk]; exists {
					te.price.Set(big.NewFloat(newPrice))
				}
				gov.mutex.Unlock()

			case opChangeGovLimit:
				chainIdxByte := r.readByte()
				limitIdxByte := r.readByte()
				r.discard(fuzzPayloadBytes - 2)

				chainIdx := int(chainIdxByte) % len(fuzzChainTable)
				limitIdx := int(limitIdxByte) % len(fuzzLimitTable)
				newLimit := fuzzLimitTable[limitIdx]
				chainID := fuzzChainTable[chainIdx].chainID

				gov.mutex.Lock()
				if ce, exists := gov.chains[chainID]; exists {
					ce.dailyLimit = newLimit.dailyLimit
					ce.bigTransactionSize = newLimit.bigTxSize
					ce.checkForBigTransactions = newLimit.bigTxSize != 0
				}
				gov.mutex.Unlock()
			}

			opCount++
		}

		// Round-trip check after all operations.
		checkDBRoundTrip("final")
	})
}
