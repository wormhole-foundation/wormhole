package connectors

import (
	"context"
	"encoding/json"
	"fmt"
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
)

// mockConnectorForPoller implements the connector interface for testing purposes.
type mockConnectorForPoller struct {
	address         ethCommon.Address
	client          *ethClient.Client
	mutex           sync.Mutex
	err             error
	persistentError bool
	blockNumbers    []uint64
	prevBlockNumber uint64
}

// setError takes an error which will be returned on the next RPC call. The error will persist until cleared.
func (m *mockConnectorForPoller) setError(err error) {
	m.mutex.Lock()
	m.err = err
	m.persistentError = true
	m.mutex.Unlock()
}

// setSingleError takes an error which will be returned on the next RPC call. After that, the error is reset to nil.
func (m *mockConnectorForPoller) setSingleError(err error) {
	m.mutex.Lock()
	m.err = err
	m.persistentError = false
	m.mutex.Unlock()
}

func (e *mockConnectorForPoller) NetworkName() string {
	return "mockConnectorForPoller"
}

func (e *mockConnectorForPoller) ContractAddress() ethCommon.Address {
	return e.address
}

func (e *mockConnectorForPoller) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	return 0, fmt.Errorf("not implemented")
}

func (e *mockConnectorForPoller) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	return ethAbi.StructsGuardianSet{}, fmt.Errorf("not implemented")
}

func (e *mockConnectorForPoller) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	var s ethEvent.Subscription
	return s, fmt.Errorf("not implemented")
}

func (e *mockConnectorForPoller) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *mockConnectorForPoller) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (e *mockConnectorForPoller) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *mockConnectorForPoller) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	var s ethEvent.Subscription
	return s, fmt.Errorf("not implemented")
}

func (e *mockConnectorForPoller) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) (err error) {
	if method != "eth_getBlockByNumber" {
		panic("method not implemented by mockConnectorForPoller")
	}

	e.mutex.Lock()
	// If they set the error, return that immediately.
	if e.err != nil {
		err = e.err
		if !e.persistentError {
			e.err = nil
		}
	} else {
		blockNumber := e.prevBlockNumber
		if len(e.blockNumbers) > 0 {
			blockNumber = e.blockNumbers[0]
			e.blockNumbers = e.blockNumbers[1:]
		}
		str := fmt.Sprintf(`{"author":"0x24c275f0719fdaec6356c4eb9f39ecb9c4d37ce1","baseFeePerGas":"0x3b9aca00","difficulty":"0x0","extraData":"0x","gasLimit":"0xe4e1c0","gasUsed":"0x0","hash":"0xfc8b62a31110121c57cfcccfaf2b147cc2c13b6d01bde4737846cefd29f045cf","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x24c275f0719fdaec6356c4eb9f39ecb9c4d37ce1","nonce":"0x0000000000000000","number":"0x%x","parentHash":"0x09d6d33a658b712f41db7fb9f775f94911ae0132123116aa4f8cf3da9f774e89","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x201","stateRoot":"0x0409ed10e03fd49424ae1489c6fbc6ff1897f45d0e214655ebdb8df94eedc3c0","timestamp":"0x6373ec24","totalDifficulty":"0x0","transactions":[],"transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","uncles":[]}`, blockNumber)
		err = json.Unmarshal([]byte(str), &result)
		e.prevBlockNumber = blockNumber
	}
	e.mutex.Unlock()

	return
}

func (e *mockConnectorForPoller) RawBatchCallContext(ctx context.Context, b []ethRpc.BatchElem) error {
	panic("method not implemented by mockConnectorForPoller")
}

func (e *mockConnectorForPoller) setBlockNumber(blockNumber uint64) {
	e.mutex.Lock()
	e.blockNumbers = []uint64{blockNumber}
	e.mutex.Unlock()
}

func (e *mockConnectorForPoller) setBlockNumbers(blockNumbers []uint64) {
	e.mutex.Lock()
	e.blockNumbers = blockNumbers
	e.mutex.Unlock()
}

func (e *mockConnectorForPoller) expectedHash() ethCommon.Hash {
	return ethCommon.HexToHash("0xfc8b62a31110121c57cfcccfaf2b147cc2c13b6d01bde4737846cefd29f045cf")
}

func (e *mockConnectorForPoller) Client() *ethClient.Client {
	return e.client
}

func (e *mockConnectorForPoller) SubscribeNewHead(ctx context.Context, ch chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	return nil, nil
}

type mockFinalizerForPoller struct {
	mutex     sync.Mutex
	finalized bool
}

func newMockFinalizerForPoller(initialState bool) *mockFinalizerForPoller {
	return &mockFinalizerForPoller{finalized: initialState}
}

func (f *mockFinalizerForPoller) setFinalized(finalized bool) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	f.finalized = finalized
}

func (f *mockFinalizerForPoller) IsBlockFinalized(ctx context.Context, block *NewBlock) (bool, error) {
	f.mutex.Lock()
	defer f.mutex.Unlock()
	return f.finalized, nil
}

func shouldHaveAllThree(t *testing.T, block []*NewBlock, blockNum uint64, expectedHash ethCommon.Hash) {
	require.Equal(t, 3, len(block))
	assert.Equal(t, uint64(blockNum), block[0].Number.Uint64())
	assert.Equal(t, Latest, block[0].Finality)
	assert.Equal(t, expectedHash, block[0].Hash)
	assert.Equal(t, uint64(blockNum), block[1].Number.Uint64())
	assert.Equal(t, Safe, block[1].Finality)
	assert.Equal(t, expectedHash, block[1].Hash)
	assert.Equal(t, uint64(blockNum), block[2].Number.Uint64())
	assert.Equal(t, Finalized, block[2].Finality)
	assert.Equal(t, expectedHash, block[2].Hash)
}

func shouldHaveLatestOnly(t *testing.T, block []*NewBlock, blockNum uint64, expectedHash ethCommon.Hash) {
	require.Equal(t, 1, len(block))
	assert.Equal(t, uint64(blockNum), block[0].Number.Uint64())
	assert.Equal(t, Latest, block[0].Finality)
	assert.Equal(t, expectedHash, block[0].Hash)
}

func shouldHaveSafeAndFinalizedButNotLatest(t *testing.T, block []*NewBlock, blockNum uint64, expectedHash ethCommon.Hash) {
	require.Equal(t, 2, len(block))
	assert.Equal(t, uint64(blockNum), block[0].Number.Uint64())
	assert.Equal(t, Safe, block[0].Finality)
	assert.Equal(t, expectedHash, block[0].Hash)
	assert.Equal(t, uint64(blockNum), block[1].Number.Uint64())
	assert.Equal(t, Finalized, block[1].Finality)
	assert.Equal(t, expectedHash, block[1].Hash)
}

// TestBlockPoller is one big, ugly test because of all the set up required.
func TestBlockPoller(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := mockConnectorForPoller{blockNumbers: []uint64{}}

	finalizer := newMockFinalizerForPoller(true) // Start by assuming blocks are finalized.
	assert.NotNil(t, finalizer)

	poller := &FinalizerPollConnector{
		Connector: &baseConnector,
		logger:    logger,
		Delay:     1 * time.Millisecond,
		finalizer: finalizer,
	}

	// Set the starting block[0].
	baseConnector.setBlockNumber(0x309a0c)

	// The go routines will post results here.
	var mutex sync.Mutex
	var block []*NewBlock
	var err error
	var pollerStatus int

	const pollerRunning = 1
	const pollerExited = 2

	// Start the poller running.
	go func() {
		mutex.Lock()
		pollerStatus = pollerRunning
		mutex.Unlock()
		err := poller.run(ctx)
		require.NoError(t, err)
		mutex.Lock()
		pollerStatus = pollerExited
		mutex.Unlock()
	}()

	// Subscribe for events to be processed by our go routine.
	headSink := make(chan *NewBlock, 2)
	errC := make(chan error)

	headerSubscription, suberr := poller.SubscribeForBlocks(ctx, errC, headSink)
	require.NoError(t, suberr)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case thisErr := <-errC:
				mutex.Lock()
				err = thisErr
				mutex.Unlock()
			case thisErr := <-headerSubscription.Err():
				mutex.Lock()
				err = thisErr
				mutex.Unlock()
			case thisBlock := <-headSink:
				require.NotNil(t, thisBlock)
				mutex.Lock()
				block = append(block, thisBlock)
				mutex.Unlock()
			}
		}
	}()

	// First sleep a bit and make sure there were no start up errors.
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	assert.Nil(t, block)
	mutex.Unlock()

	// Post the first new block and verify we get it.
	baseConnector.setBlockNumber(0x309a0d)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	shouldHaveAllThree(t, block, 0x309a0d, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	// Sleep some more and verify we don't see any more blocks, since we haven't posted a new one.
	baseConnector.setBlockNumber(0x309a0d)
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	require.Nil(t, block)
	mutex.Unlock()

	// Post the next block and verify we get it.
	baseConnector.setBlockNumber(0x309a0e)
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	shouldHaveAllThree(t, block, 0x309a0e, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	// Post the next block but mark it as not finalized, so we should only see latest.
	mutex.Lock()
	finalizer.setFinalized(false)
	baseConnector.setBlockNumber(0x309a0f)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	shouldHaveLatestOnly(t, block, 0x309a0f, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	// Once it goes finalized we should see safe and finalized, but not latest again.
	mutex.Lock()
	finalizer.setFinalized(true)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	shouldHaveSafeAndFinalizedButNotLatest(t, block, 0x309a0f, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	// An RPC error should be returned to us.
	err = nil
	baseConnector.setError(fmt.Errorf("RPC failed"))

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	assert.Error(t, err)
	assert.Nil(t, block)
	baseConnector.setError(nil)
	err = nil
	mutex.Unlock()

	// Post the next block and verify we get it (so we survived the RPC error).
	baseConnector.setBlockNumber(0x309a10)

	// There may be a few errors already queued up. Loop for a bit before we give up.
	success := false
	for count := 0; (count < 20) && (!success); count++ {
		time.Sleep(10 * time.Millisecond)
		mutex.Lock()
		if err == nil {
			success = true
		} else {
			err = nil
		}
		mutex.Unlock()
	}
	require.True(t, success)

	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	shouldHaveAllThree(t, block, 0x309a10, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	// Post an old block and we should not hear about it.
	baseConnector.setBlockNumber(0x309a0c)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	require.Nil(t, block)
	mutex.Unlock()

	// But we should keep going when we get a new one.
	baseConnector.setBlockNumber(0x309a11)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	shouldHaveAllThree(t, block, 0x309a11, baseConnector.expectedHash())
	block = nil

	// If there's a gap in the blocks, we play out the gap.
	baseConnector.setBlockNumbers([]uint64{
		0x309a13, // New Latest
		0x309a12, // Gap Latest
		0x309a12, // Gap Safe & Finalized
		0x309a13, // New Safe & Finalized
	})
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	require.Equal(t, 6, len(block))

	assert.Equal(t, uint64(0x309a12), block[0].Number.Uint64())
	assert.Equal(t, baseConnector.expectedHash(), block[0].Hash)
	assert.Equal(t, Latest, block[0].Finality)
	assert.Equal(t, uint64(0x309a13), block[1].Number.Uint64())
	assert.Equal(t, baseConnector.expectedHash(), block[1].Hash)
	assert.Equal(t, Latest, block[1].Finality)

	assert.Equal(t, uint64(0x309a12), block[2].Number.Uint64())
	assert.Equal(t, baseConnector.expectedHash(), block[2].Hash)
	assert.Equal(t, Safe, block[2].Finality)
	assert.Equal(t, uint64(0x309a12), block[3].Number.Uint64())
	assert.Equal(t, baseConnector.expectedHash(), block[3].Hash)
	assert.Equal(t, Finalized, block[3].Finality)

	assert.Equal(t, uint64(0x309a13), block[4].Number.Uint64())
	assert.Equal(t, baseConnector.expectedHash(), block[4].Hash)
	assert.Equal(t, Safe, block[4].Finality)
	assert.Equal(t, uint64(0x309a13), block[5].Number.Uint64())
	assert.Equal(t, baseConnector.expectedHash(), block[5].Hash)
	assert.Equal(t, Finalized, block[5].Finality)

	block = nil
	mutex.Unlock()

	// Should retry on a transient error and be able to continue.
	baseConnector.setSingleError(fmt.Errorf("RPC failed"))
	baseConnector.setBlockNumber(0x309a14)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, pollerRunning, pollerStatus)
	require.NoError(t, err)
	shouldHaveAllThree(t, block, 0x309a14, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()
}
