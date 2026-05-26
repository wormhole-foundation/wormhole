package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethClient "github.com/ethereum/go-ethereum/ethclient"
	ethEvent "github.com/ethereum/go-ethereum/event"
	ethRpc "github.com/ethereum/go-ethereum/rpc"

	dgAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/delegated_guardians"
)

// mockConnectorForPoller implements the Connector interface for PollConnector tests.
// Unlike the batch poller mock, it does NOT provide SubscribeNewHead since the
// PollConnector does not use it.
type mockConnectorForPoller struct {
	address       ethCommon.Address
	client        *ethClient.Client
	mutex         sync.Mutex
	err           error
	blockNumbers  []uint64 // consumed in order by RawBatchCallContext
	prevLatest    uint64
	prevSafe      uint64
	prevFinalized uint64
}

func (m *mockConnectorForPoller) NetworkName() string                { return "mockPoller" }
func (m *mockConnectorForPoller) ContractAddress() ethCommon.Address { return m.address }
func (m *mockConnectorForPoller) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	return 0, fmt.Errorf("not implemented")
}
func (m *mockConnectorForPoller) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	return ethAbi.StructsGuardianSet{}, fmt.Errorf("not implemented")
}
func (m *mockConnectorForPoller) GetDelegatedGuardianConfig(ctx context.Context) ([]dgAbi.WormholeDelegatedGuardiansDelegatedGuardianSet, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockConnectorForPoller) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockConnectorForPoller) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockConnectorForPoller) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}
func (m *mockConnectorForPoller) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockConnectorForPoller) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	return nil, fmt.Errorf("not implemented")
}
func (m *mockConnectorForPoller) GetLatest(ctx context.Context) (latest, finalized, safe uint64, err error) {
	return m.prevLatest, m.prevFinalized, m.prevSafe, nil
}
func (m *mockConnectorForPoller) Client() *ethClient.Client { return m.client }
func (m *mockConnectorForPoller) SubscribeNewHead(ctx context.Context, ch chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	return nil, fmt.Errorf("not supported on HTTP")
}
func (m *mockConnectorForPoller) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if method == "eth_getBlockByNumber" && len(args) >= 1 {
		tag, ok := args[0].(string)
		if !ok {
			return fmt.Errorf("unexpected arg type")
		}
		var blockNumber uint64
		switch tag {
		case "latest":
			blockNumber = m.prevLatest
		case "safe":
			blockNumber = m.prevSafe
		case "finalized":
			blockNumber = m.prevFinalized
		default:
			if strings.HasPrefix(tag, "0x") {
				n, err := strconv.ParseUint(tag[2:], 16, 64)
				if err == nil {
					blockNumber = n
				}
			}
		}
		str := fmt.Sprintf(`{"number":"0x%x","hash":"0xfc8b62a31110121c57cfcccfaf2b147cc2c13b6d01bde4737846cefd29f045cf","timestamp":"0x6373ec24"}`, blockNumber)
		return json.Unmarshal([]byte(str), result)
	}
	return fmt.Errorf("not implemented")
}

func (m *mockConnectorForPoller) RawBatchCallContext(ctx context.Context, b []ethRpc.BatchElem) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.err != nil {
		return m.err
	}

	for i, entry := range b {
		if entry.Method != "eth_getBlockByNumber" {
			return fmt.Errorf("unexpected method: %s", entry.Method)
		}

		var blockNumber uint64
		tag, ok := entry.Args[0].(string)
		if ok {
			switch tag {
			case "latest":
				blockNumber = m.prevLatest
			case "safe":
				blockNumber = m.prevSafe
			case "finalized":
				blockNumber = m.prevFinalized
			default:
				// Handle hex block numbers used by getBlockRange (e.g. "0x65").
				if strings.HasPrefix(tag, "0x") {
					n, err := strconv.ParseUint(tag[2:], 16, 64)
					if err == nil {
						blockNumber = n
					}
				}
			}
		}
		if len(m.blockNumbers) > 0 {
			blockNumber = m.blockNumbers[0]
			m.blockNumbers = m.blockNumbers[1:]
		}

		str := fmt.Sprintf(`{"number":"0x%x","hash":"0xfc8b62a31110121c57cfcccfaf2b147cc2c13b6d01bde4737846cefd29f045cf","timestamp":"0x6373ec24"}`, blockNumber)
		if err := json.Unmarshal([]byte(str), &b[i].Result); err != nil {
			return err
		}

		if ok {
			switch tag {
			case "latest":
				m.prevLatest = blockNumber
			case "safe":
				m.prevSafe = blockNumber
			case "finalized":
				m.prevFinalized = blockNumber
			}
		}
	}
	return nil
}

func (m *mockConnectorForPoller) setBlockNumbers(finalized, latest uint64) {
	m.mutex.Lock()
	// PollConnector with safe=false batches [finalized, latest].
	m.blockNumbers = []uint64{finalized, latest}
	m.mutex.Unlock()
}

// collectBlocks reads up to count blocks from sink within timeout. Fails the
// test if fewer than count blocks are received.
func collectBlocks(t *testing.T, sink <-chan *NewBlock, count int, timeout time.Duration) []*NewBlock {
	t.Helper()
	blocks := make([]*NewBlock, 0, count)
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	for len(blocks) < count {
		select {
		case b := <-sink:
			if b != nil {
				blocks = append(blocks, b)
			}
		case <-deadline.C:
			t.Fatalf("timed out waiting for %d blocks (got %d) after %s", count, len(blocks), timeout)
		}
	}
	return blocks
}

// expectNoBlock asserts that no block is published on sink within the given window.
func expectNoBlock(t *testing.T, sink <-chan *NewBlock, window time.Duration) {
	t.Helper()
	timer := time.NewTimer(window)
	defer timer.Stop()
	select {
	case b := <-sink:
		t.Fatalf("unexpected block published: finality=%v number=%d", b.Finality, b.Number.Uint64())
	case <-timer.C:
	}
}

// TestPollConnector verifies the PollConnector's SubscribeForBlocks produces
// finalized, generated-safe, and latest blocks without requiring WebSocket.
func TestPollConnector(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := zap.NewNop()
	mock := &mockConnectorForPoller{blockNumbers: []uint64{}}

	// safeSupported=false mirrors Tron config.
	poller := NewPollConnector(ctx, logger, mock, false, 1*time.Millisecond)

	// Set initial blocks: finalized=100, latest=110.
	mock.setBlockNumbers(100, 110)

	headSink := make(chan *NewBlock, 10)
	errC := make(chan error)

	sub, err := poller.SubscribeForBlocks(ctx, errC, headSink)
	require.NoError(t, err)
	require.NotNil(t, sub)
	defer sub.Unsubscribe()

	// Initial blocks: finalized, generated-safe, latest = 3 blocks.
	blocks := collectBlocks(t, headSink, 3, time.Second)
	hasFinalized := false
	hasSafe := false
	hasLatest := false
	for _, b := range blocks {
		switch b.Finality {
		case Finalized:
			hasFinalized = true
			assert.Equal(t, uint64(100), b.Number.Uint64())
		case Safe:
			hasSafe = true
			assert.Equal(t, uint64(100), b.Number.Uint64()) // generated from finalized
		case Latest:
			hasLatest = true
			assert.Equal(t, uint64(110), b.Number.Uint64())
		}
	}
	assert.True(t, hasFinalized, "missing finalized")
	assert.True(t, hasSafe, "missing safe (generated)")
	assert.True(t, hasLatest, "missing latest")

	// Advance blocks and verify polling picks them up (new finalized + safe + latest).
	mock.setBlockNumbers(101, 111)
	blocks = collectBlocks(t, headSink, 3, time.Second)
	foundNewFinalized := false
	foundNewLatest := false
	for _, b := range blocks {
		if b.Finality == Finalized && b.Number.Uint64() == 101 {
			foundNewFinalized = true
		}
		if b.Finality == Latest && b.Number.Uint64() == 111 {
			foundNewLatest = true
		}
	}
	assert.True(t, foundNewFinalized, "should see new finalized block 101")
	assert.True(t, foundNewLatest, "should see new latest block 111")

	// No new blocks — verify nothing extra is published.
	mock.setBlockNumbers(101, 111)
	expectNoBlock(t, headSink, 50*time.Millisecond)
}

// TestPollConnectorGetLatest verifies GetLatest returns correct values.
func TestPollConnectorGetLatest(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	mock := &mockConnectorForPoller{
		prevLatest:    200,
		prevFinalized: 180,
		prevSafe:      180,
	}

	poller := NewPollConnector(ctx, logger, mock, false, time.Second)

	latest, finalized, safe, err := poller.GetLatest(ctx)
	require.NoError(t, err)
	assert.Equal(t, uint64(200), latest)
	assert.Equal(t, uint64(180), finalized)
	assert.Equal(t, uint64(180), safe) // generated from finalized when safe unsupported
}

// TestPollConnectorSubscribeNewHeadReturnsError confirms SubscribeNewHead is unsupported.
func TestPollConnectorSubscribeNewHeadReturnsError(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	mock := &mockConnectorForPoller{}
	poller := NewPollConnector(ctx, logger, mock, false, time.Second)

	ch := make(chan *ethTypes.Header)
	_, err := poller.SubscribeNewHead(ctx, ch)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}

// TestPollConnectorTimeOfBlockByHash verifies the RawCallContext-based override.
func TestPollConnectorTimeOfBlockByHash(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()

	hash := ethCommon.HexToHash("0xabcd")
	expectedTime := uint64(1700000000)

	// Override RawCallContext on the mock to return a block with the expected timestamp.
	mock := &mockConnectorForPollerWithRawCall{
		mockConnectorForPoller: mockConnectorForPoller{},
		rawCallResult:          fmt.Sprintf(`{"number":"0x100","hash":"%s","timestamp":"0x%x"}`, hash.Hex(), expectedTime),
	}

	poller := NewPollConnector(ctx, logger, mock, false, time.Second)

	blockTime, err := poller.TimeOfBlockByHash(ctx, hash)
	require.NoError(t, err)
	assert.Equal(t, expectedTime, blockTime)
}

// mockConnectorForPollerWithRawCall extends the mock to support RawCallContext.
type mockConnectorForPollerWithRawCall struct {
	mockConnectorForPoller
	rawCallResult string
}

func (m *mockConnectorForPollerWithRawCall) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	return json.Unmarshal([]byte(m.rawCallResult), result)
}

// TestPollConnectorGapFill verifies that the poller fills gaps when blocks jump.
func TestPollConnectorGapFill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := zap.NewNop()
	mock := &mockConnectorForPoller{blockNumbers: []uint64{}}

	poller := NewPollConnector(ctx, logger, mock, false, 1*time.Millisecond)

	// Start at finalized=100, latest=110.
	mock.setBlockNumbers(100, 110)

	headSink := make(chan *NewBlock, 100)
	errC := make(chan error)

	sub, err := poller.SubscribeForBlocks(ctx, errC, headSink)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Drain the initial 3 blocks (finalized + generated-safe + latest).
	_ = collectBlocks(t, headSink, 3, time.Second)

	// Jump finalized 100→103 and latest 110→113. Gap fill produces:
	//   finalized: 101, 102, 103  (+ generated safe for each)  = 6 blocks
	//   latest:    111, 112, 113                                = 3 blocks
	// Total = 9 blocks.
	mock.setBlockNumbers(103, 113)
	blocks := collectBlocks(t, headSink, 9, 2*time.Second)

	finalizedNums := make(map[uint64]bool)
	for _, b := range blocks {
		if b.Finality == Finalized {
			finalizedNums[b.Number.Uint64()] = true
		}
	}
	assert.True(t, finalizedNums[101], "gap fill: finalized 101")
	assert.True(t, finalizedNums[102], "gap fill: finalized 102")
	assert.True(t, finalizedNums[103], "gap fill: finalized 103")
}
