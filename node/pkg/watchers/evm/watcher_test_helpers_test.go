package evm

import (
	"context"
	"encoding/binary"
	"math/big"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/txverifier"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	dgAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/delegated_guardians"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"
	ethereum "github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// testEmitter is a default non-zero emitter address for use across tests.
var testEmitter = eth_common.HexToAddress("0x0290FB167208Af455bB137780163b7B7a9a10C16")

// testTokenBridge is the token bridge address used by MockTransferVerifier.
var testTokenBridge = eth_common.BytesToAddress([]byte{0x01})

// Default block heights used across reobserve tests: the log is observed at testBlockNumber,
// with testFinalizedBlockNum / testSafeBlockNum representing current chain heads.
var (
	testBlockNumber       = uint64(100)
	testFinalizedBlockNum = uint64(200)
	testSafeBlockNum      = uint64(150)
	testBlockTime         = uint64(1234)
)

// NewWatcherForTest creates a minimal Watcher for verifyAndPublish tests.
func NewWatcherForTest(t *testing.T, msgC chan<- *common.MessagePublication) *Watcher {
	t.Helper()
	logger := zap.NewNop()

	w := &Watcher{
		// this is implicit but added here for clarity
		txVerifierEnabled: false,
		msgC:              msgC,
		logger:            logger,
	}

	return w
}

// newTestWatcher creates a Watcher wired up with a mockConnector for processNewBlock and postMessage tests.
// Returns the watcher, the mock (for configuring receipts/errors), and the msgC channel (for reading published messages).
func newTestWatcher(t *testing.T) (*Watcher, *mockConnector, chan *common.MessagePublication) {
	t.Helper()
	mock := newMockConnector(t)
	msgC := make(chan *common.MessagePublication, 10)

	w := &Watcher{
		logger:      zap.NewNop(),
		cclLogger:   zap.NewNop(),
		ethConn:     mock,
		msgC:        msgC,
		pending:     make(map[pendingKey]*pendingMessage),
		networkName: "test",
		chainID:     vaa.ChainIDEthereum,
		contract:    testEmitter,
	}

	return w, mock, msgC
}

// recvMsg reads from msgC, failing the test (rather than blocking forever) if no
// message arrives within a short timeout. Prefer this over a bare `<-msgC` so a
// missing publish surfaces as a clear failure instead of a `go test` hang.
func recvMsg(t *testing.T, msgC <-chan *common.MessagePublication) *common.MessagePublication {
	t.Helper()
	select {
	case msg := <-msgC:
		return msg
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for message on msgC")
		return nil
	}
}

type MockTransferVerifier[E ethclient.Client, C connectors.Connector] struct {
	success bool
}

// TransferIsValid simulates the evaluation made by the Transfer Verifier.
// Always returns nil. The error should be non-nil only when a parsing or RPC error occurs.
// For now, these are not included in the unit tests.
func (m *MockTransferVerifier[E, C]) TransferIsValid(_ context.Context, _ string, _ eth_common.Hash, _ *types.Receipt) (bool, error) {
	return m.success, nil
}
func (m *MockTransferVerifier[E, C]) Addrs() *txverifier.TVAddresses {
	return &txverifier.TVAddresses{
		TokenBridgeAddr: testTokenBridge,
	}
}

// mockConnector is a minimal mock of the connectors.Connector interface for testing
// processNewBlock, postMessage, and reobservation flows. TransactionReceipt, TimeOfBlockByHash,
// and ParseLogMessagePublished have real behavior; all other methods panic because they should
// not be called in these tests.
type mockConnector struct {
	// receipts maps txHash -> receipt to return
	receipts map[eth_common.Hash]*types.Receipt
	// errors maps txHash -> error to return from TransactionReceipt
	errors map[eth_common.Hash]error
	// blockTimes maps blockHash -> time to return from TimeOfBlockByHash
	blockTimes map[eth_common.Hash]uint64
	// filterer delegates ParseLogMessagePublished to the real ABI parser.
	filterer *ethabi.AbiFilterer
}

func newMockConnector(t *testing.T) *mockConnector {
	t.Helper()
	// The zero address and nil filterer are fine here: UnpackLog only needs the parsed ABI
	// metadata baked into the generated package, not an RPC-capable filterer.
	filterer, err := ethabi.NewAbiFilterer(eth_common.Address{}, nil)
	require.NoError(t, err)
	return &mockConnector{
		receipts:   make(map[eth_common.Hash]*types.Receipt),
		errors:     make(map[eth_common.Hash]error),
		blockTimes: make(map[eth_common.Hash]uint64),
		filterer:   filterer,
	}
}

// seedLog wires the mock to return a successful receipt wrapping `log` (keyed by its TxHash)
// and the given block time (keyed by the receipt's BlockHash). For receipts with Status != 1
// or custom shapes, write directly to mock.receipts / mock.blockTimes.
func (m *mockConnector) seedLog(log *types.Log) {
	receipt := newTestReceipt(log.BlockNumber, []*types.Log{log})
	m.receipts[log.TxHash] = receipt
	m.blockTimes[receipt.BlockHash] = testBlockTime
}

// TransactionReceipt returns the configured receipt/error. When neither is set for the txHash,
// it returns a "not found" error to match the real ethclient behavior (and avoid nil-deref in
// callers that don't check err before dereferencing the receipt).
func (m *mockConnector) TransactionReceipt(_ context.Context, txHash eth_common.Hash) (*types.Receipt, error) {
	err := m.errors[txHash]
	r, hasReceipt := m.receipts[txHash]
	if err == nil && !hasReceipt {
		return nil, ethereum.NotFound
	}
	return r, err
}

// Stub implementations that are required by the interface
func (m *mockConnector) NetworkName() string {
	panic("not implemented")
}
func (m *mockConnector) ContractAddress() eth_common.Address {
	panic("not implemented")
}
func (m *mockConnector) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	panic("not implemented")
}
func (m *mockConnector) GetGuardianSet(ctx context.Context, index uint32) (ethabi.StructsGuardianSet, error) {
	panic("not implemented")
}
func (m *mockConnector) GetDelegatedGuardianConfig(ctx context.Context) ([]dgAbi.WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	panic("not implemented")
}
func (m *mockConnector) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethabi.AbiLogMessagePublished) (event.Subscription, error) {
	panic("not implemented")
}

// TimeOfBlockByHash returns the configured block time for the given hash, or 0 if unset.
// Tests that care about the exact timestamp should seed blockTimes explicitly.
func (m *mockConnector) TimeOfBlockByHash(_ context.Context, hash eth_common.Hash) (uint64, error) {
	return m.blockTimes[hash], nil
}

// ParseLogMessagePublished delegates to the real generated ABI parser so tests exercise the
// actual unpack path against logs produced by newTestLog.
func (m *mockConnector) ParseLogMessagePublished(log types.Log) (*ethabi.AbiLogMessagePublished, error) {
	return m.filterer.ParseLogMessagePublished(log)
}
func (m *mockConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *connectors.NewBlock) (ethereum.Subscription, error) {
	panic("not implemented")
}
func (m *mockConnector) GetLatest(ctx context.Context) (latest, finalized, safe uint64, err error) {
	panic("not implemented")
}
func (m *mockConnector) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	panic("not implemented")
}
func (m *mockConnector) RawBatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	panic("not implemented")
}
func (m *mockConnector) Client() *ethclient.Client {
	panic("not implemented")
}
func (m *mockConnector) SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error) {
	panic("not implemented")
}

// newBlock creates a *connectors.NewBlock with sensible defaults for testing.
// Hash is deterministically derived from the block number.
func newBlock(number uint64, finality connectors.FinalityLevel) *connectors.NewBlock {
	return &connectors.NewBlock{
		Number:   big.NewInt(int64(number)), // #nosec G115 -- test-only
		Hash:     eth_common.BigToHash(big.NewInt(int64(number))), // #nosec G115 -- test-only
		Time:     number,
		Finality: finality,
	}
}

// addPendingMsg adds an entry to w.pending and returns the key.
// The MessagePublication is constructed with TxID, EmitterChain, and ConsistencyLevel set from the arguments.
func (w *Watcher) addPendingMsg(
	txHash eth_common.Hash,
	blockHash eth_common.Hash,
	effectiveCL uint8,
	additionalBlocks uint64,
	sequence uint64,
) pendingKey {
	key := pendingKey{
		TxHash:    txHash,
		BlockHash: blockHash,
		Sequence:  sequence,
	}

	msg := &common.MessagePublication{
		TxID:         txHash.Bytes(),
		EmitterChain: w.chainID,
	}

	w.pending[key] = &pendingMessage{
		message:          msg,
		height:           testBlockNumber,
		effectiveCL:      effectiveCL,
		additionalBlocks: additionalBlocks,
	}

	return key
}

// enableCCL enables the CCL feature on the watcher and initializes the cache.
func (w *Watcher) enableCCL() {
	w.cclEnabled = true
	w.cclLogger = zap.NewNop()
	w.cclCache = CCLCache{}
}

// seedCCLAdditionalBlocks pre-populates the CCL cache for emitter with an AdditionalBlocks config.
// Allows for mocking, since no RPC calls will be made.
func (w *Watcher) seedCCLAdditionalBlocks(emitter eth_common.Address, consistencyLevel uint8, additionalBlocks uint16) {
	var data [32]byte
	data[0] = byte(AdditionalBlocksType)
	data[1] = consistencyLevel
	binary.BigEndian.PutUint16(data[2:4], additionalBlocks)

	w.cclCache[emitter] = CCLCacheEntry{
		data:     data,
		readTime: time.Now(),
	}
}

// seedCCLNothingSpecial pre-populates the CCL cache for emitter with a NothingSpecial config
// (all zeros), meaning no custom handling — treated as finalized.
func (w *Watcher) seedCCLNothingSpecial(emitter eth_common.Address) {
	w.cclCache[emitter] = CCLCacheEntry{
		data:     cclEmptyData,
		readTime: time.Now(),
	}
}

// seedCCLInvalidType pre-populates the CCL cache for emitter with an invalid type byte,
// triggering a parse error in cclReadAndParseConfig.
func (w *Watcher) seedCCLInvalidType(emitter eth_common.Address) {
	var data [32]byte
	data[0] = 0xFF // invalid CCL request type

	w.cclCache[emitter] = CCLCacheEntry{
		data:     data,
		readTime: time.Now(),
	}
}

// testLogParams holds parameters for constructing a types.Log with ABI-encoded data
// matching the LogMessagePublished event. Only fields that affect control flow are exposed;
// nonce and payload are hardcoded to zero/empty.
type testLogParams struct {
	sender           eth_common.Address
	sequence         uint64
	consistencyLevel uint8
	txHash           eth_common.Hash
	blockHash        eth_common.Hash
	blockNumber      uint64
	contractAddr     eth_common.Address
}

// logMessagePublishedArgs returns the ABI argument types for the non-indexed fields of
// LogMessagePublished(address indexed sender, uint64 sequence, uint32 nonce, bytes payload, uint8 consistencyLevel).
func logMessagePublishedArgs() abi.Arguments {
	uint64Ty, _ := abi.NewType("uint64", "", nil)
	uint32Ty, _ := abi.NewType("uint32", "", nil)
	bytesTy, _ := abi.NewType("bytes", "", nil)
	uint8Ty, _ := abi.NewType("uint8", "", nil)

	return abi.Arguments{
		{Name: "sequence", Type: uint64Ty},
		{Name: "nonce", Type: uint32Ty},
		{Name: "payload", Type: bytesTy},
		{Name: "consistencyLevel", Type: uint8Ty},
	}
}

// newTestLog builds a types.Log with properly ABI-encoded data for the LogMessagePublished event.
// Use this for testing ParseLogMessagePublished and MessageEventsForTransaction.
// To test failure modes, mutate the returned log directly (e.g. truncate Data, swap Topics).
func newTestLog(t *testing.T, p testLogParams) types.Log {
	t.Helper()

	txHash := p.txHash
	if txHash == (eth_common.Hash{}) {
		txHash = eth_common.BigToHash(big.NewInt(int64(p.blockNumber))) // #nosec G115 -- test-only
	}
	blockHash := p.blockHash
	if blockHash == (eth_common.Hash{}) {
		blockHash = eth_common.BigToHash(big.NewInt(int64(p.blockNumber + 0xff))) // #nosec G115 -- test-only
	}

	data, err := logMessagePublishedArgs().Pack(p.sequence, uint32(0), []byte{}, p.consistencyLevel)
	require.NoError(t, err)

	senderTopic := eth_common.BytesToHash(p.sender.Bytes())

	return types.Log{
		Address:     p.contractAddr,
		Topics:      []eth_common.Hash{LogMessagePublishedTopic, senderTopic},
		Data:        data,
		BlockNumber: p.blockNumber,
		TxHash:      txHash,
		TxIndex:     0,
		BlockHash:   blockHash,
		Index:       0,
		Removed:     false,
	}
}

// newValidWormholeLog returns a pointer to a LogMessagePublished log entry with valid ABI-encoded
// data that MessageEventsForTransaction will parse and surface as a MessagePublication.
// Defaults: sender = testEmitter, contract address = testEmitter, sequence = 1.
// newTestWatcher sets w.contract = testEmitter by default, so the by_transaction.go address filter passes.
// For full control over fields, use newTestLog with a testLogParams struct.
func newValidWormholeLog(t *testing.T, blockNumber uint64, consistencyLevel uint8) *types.Log {
	t.Helper()
	log := newTestLog(t, testLogParams{
		sender:           testTokenBridge,
		contractAddr:     testEmitter,
		sequence:         1,
		consistencyLevel: consistencyLevel,
		blockNumber:      blockNumber,
	})
	return &log
}

// newTestReceipt builds a successful (Status=1) receipt at the given block number, wiring logs verbatim.
// BlockHash matches the default derivation used by newTestLog (blockNumber + 0xff) so a log built
// from the same blockNumber will have a matching receipt.BlockHash by default.
// For Status != 1 or a custom BlockHash, construct the receipt inline in the test.
func newTestReceipt(blockNumber uint64, logs []*types.Log) *types.Receipt {
	return &types.Receipt{
		Status:      1,
		BlockHash:   eth_common.BigToHash(big.NewInt(int64(blockNumber + 0xff))), // #nosec G115 -- test-only
		BlockNumber: big.NewInt(int64(blockNumber)),                              // #nosec G115 -- test-only
		Logs:        logs,
	}
}

// testLogEventParams holds parameters for constructing an AbiLogMessagePublished for postMessage tests.
// Only fields that affect control flow are exposed; nonce and payload are left at zero values.
type testLogEventParams struct {
	sender           eth_common.Address
	sequence         uint64
	consistencyLevel uint8
	txHash           eth_common.Hash
	blockHash        eth_common.Hash
	blockNumber      uint64
}

// newTestLogEvent creates an AbiLogMessagePublished with sensible non-zero defaults for postMessage tests.
// Only blockNumber and consistencyLevel are required; sender and sequence get deterministic non-zero values.
func newTestLogEvent(consistencyLevel uint8) *ethabi.AbiLogMessagePublished {
	return newTestLogEventFromParams(testLogEventParams{
		sender:           testEmitter,
		sequence:         1,
		blockNumber:      testBlockNumber,
		consistencyLevel: consistencyLevel,
	})
}

// newTestLogEventFromParams creates an AbiLogMessagePublished with full control over fields
// that affect control flow (sender, sequence, consistencyLevel, hashes).
func newTestLogEventFromParams(p testLogEventParams) *ethabi.AbiLogMessagePublished {
	txHash := p.txHash
	if txHash == (eth_common.Hash{}) {
		txHash = eth_common.BigToHash(big.NewInt(int64(p.blockNumber))) // #nosec G115 -- test-only
	}
	blockHash := p.blockHash
	if blockHash == (eth_common.Hash{}) {
		blockHash = eth_common.BigToHash(big.NewInt(int64(p.blockNumber + 0xff))) // #nosec G115 -- test-only
	}

	return &ethabi.AbiLogMessagePublished{
		Sender:           p.sender,
		Sequence:         p.sequence,
		Payload:          []byte{},
		ConsistencyLevel: p.consistencyLevel,
		Raw: types.Log{
			Address:     testEmitter,
			Topics:      []eth_common.Hash{LogMessagePublishedTopic},
			TxHash:      txHash,
			BlockHash:   blockHash,
			BlockNumber: p.blockNumber,
		},
	}
}

// assertMessageMatchesEvent verifies that all fields on a MessagePublication match the source event.
func assertMessageMatchesEvent(t *testing.T, msg *common.MessagePublication, ev *ethabi.AbiLogMessagePublished) {
	t.Helper()
	assert.Equal(t, ev.Raw.TxHash.Bytes(), msg.TxID)
	assert.Equal(t, time.Unix(int64(testBlockTime), 0), msg.Timestamp) // #nosec G115 -- test-only
	assert.Equal(t, ev.Nonce, msg.Nonce)
	assert.Equal(t, ev.Sequence, msg.Sequence)
	assert.Equal(t, vaa.ChainIDEthereum, msg.EmitterChain)
	assert.Equal(t, PadAddress(ev.Sender), msg.EmitterAddress)
	assert.Equal(t, ev.Payload, msg.Payload)
	assert.Equal(t, ev.ConsistencyLevel, msg.ConsistencyLevel)
}

// assertPendingMetadata verifies the metadata fields on a pendingMessage.
func assertPendingMetadata(t *testing.T, pe *pendingMessage, effectiveCL uint8, additionalBlocks uint64) {
	t.Helper()
	assert.Equal(t, effectiveCL, pe.effectiveCL)
	assert.Equal(t, testBlockNumber, pe.height)
	assert.Equal(t, additionalBlocks, pe.additionalBlocks)
}
