package connectors

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
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

// mockConnectorForBatchPoller implements the connector interface for testing purposes.
type mockConnectorForBatchPoller struct {
	address         ethCommon.Address
	client          *ethClient.Client
	mutex           sync.Mutex
	headSink        chan<- *ethTypes.Header
	sub             ethEvent.Subscription
	err             error
	persistentError bool
	blockNumbers    []uint64
	prevLatest      uint64
	prevSafe        uint64
	prevFinalized   uint64
}

// setError takes an error which will be returned on the next RPC call. The error will persist until cleared.
func (m *mockConnectorForBatchPoller) setError(err error) {
	m.mutex.Lock()
	m.err = err
	m.persistentError = true
	m.mutex.Unlock()
}

// setSingleError takes an error which will be returned on the next RPC call. After that, the error is reset to nil.
func (m *mockConnectorForBatchPoller) setSingleError(err error) {
	m.mutex.Lock()
	m.err = err
	m.persistentError = false
	m.mutex.Unlock()
}

func (e *mockConnectorForBatchPoller) NetworkName() string {
	return "mockConnectorForBatchPoller"
}

func (e *mockConnectorForBatchPoller) ContractAddress() ethCommon.Address {
	return e.address
}

func (e *mockConnectorForBatchPoller) GetCurrentGuardianSetIndex(ctx context.Context) (uint32, error) {
	return 0, fmt.Errorf("not implemented")
}

func (e *mockConnectorForBatchPoller) GetGuardianSet(ctx context.Context, index uint32) (ethAbi.StructsGuardianSet, error) {
	return ethAbi.StructsGuardianSet{}, fmt.Errorf("not implemented")
}

func (e *mockConnectorForBatchPoller) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	var s ethEvent.Subscription
	return s, fmt.Errorf("not implemented")
}

func (e *mockConnectorForBatchPoller) TransactionReceipt(ctx context.Context, txHash ethCommon.Hash) (*ethTypes.Receipt, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *mockConnectorForBatchPoller) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	return 0, fmt.Errorf("not implemented")
}

func (e *mockConnectorForBatchPoller) ParseLogMessagePublished(log ethTypes.Log) (*ethAbi.AbiLogMessagePublished, error) {
	return nil, fmt.Errorf("not implemented")
}

func (e *mockConnectorForBatchPoller) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	return e.sub, fmt.Errorf("not implemented")
}

func (e *mockConnectorForBatchPoller) GetLatest(ctx context.Context) (latest, finalized, safe uint64, err error) {
	return e.prevLatest, e.prevFinalized, e.prevSafe, nil
}

func (e *mockConnectorForBatchPoller) RawCallContext(ctx context.Context, result interface{}, method string, args ...interface{}) error {
	panic("method not implemented by mockConnectorForBatchPoller")
}

func (e *mockConnectorForBatchPoller) RawBatchCallContext(ctx context.Context, b []ethRpc.BatchElem) (err error) {
	e.mutex.Lock()
	if e.err != nil {
		err := e.err
		if !e.persistentError {
			e.err = nil
		}
		e.mutex.Unlock()
		return err
	}

	for i, entry := range b {
		if entry.Method != "eth_getBlockByNumber" {
			panic("method not implemented by mockConnectorForBatchPoller")
		}

		var blockNumber uint64
		if entry.Args[0] == "latest" {
			blockNumber = e.prevLatest
		} else if entry.Args[0] == "safe" {
			blockNumber = e.prevSafe
		} else if entry.Args[0] == "finalized" {
			blockNumber = e.prevFinalized
		}
		if len(e.blockNumbers) > 0 {
			blockNumber = e.blockNumbers[0]
			e.blockNumbers = e.blockNumbers[1:]
		}
		str := fmt.Sprintf(`{"author":"0x24c275f0719fdaec6356c4eb9f39ecb9c4d37ce1","baseFeePerGas":"0x3b9aca00","difficulty":"0x0","extraData":"0x","gasLimit":"0xe4e1c0","gasUsed":"0x0","hash":"0xfc8b62a31110121c57cfcccfaf2b147cc2c13b6d01bde4737846cefd29f045cf","logsBloom":"0x00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000","miner":"0x24c275f0719fdaec6356c4eb9f39ecb9c4d37ce1","nonce":"0x0000000000000000","number":"0x%x","parentHash":"0x09d6d33a658b712f41db7fb9f775f94911ae0132123116aa4f8cf3da9f774e89","receiptsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","sha3Uncles":"0x1dcc4de8dec75d7aab85b567b6ccd41ad312451b948a7413f0a142fd40d49347","size":"0x201","stateRoot":"0x0409ed10e03fd49424ae1489c6fbc6ff1897f45d0e214655ebdb8df94eedc3c0","timestamp":"0x6373ec24","totalDifficulty":"0x0","transactions":[],"transactionsRoot":"0x56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421","uncles":[]}`, blockNumber)
		err = json.Unmarshal([]byte(str), &b[i].Result)
		if entry.Args[0] == "latest" {
			e.prevLatest = blockNumber
		} else if entry.Args[0] == "safe" {
			e.prevSafe = blockNumber
		} else if entry.Args[0] == "finalized" {
			e.prevFinalized = blockNumber
		}
	}

	e.mutex.Unlock()

	return
}

func (e *mockConnectorForBatchPoller) setBlockNumbers(finalized, safe, latest uint64) {
	e.mutex.Lock()
	e.blockNumbers = []uint64{finalized, safe}
	if latest != 0 {
		e.headSink <- &ethTypes.Header{
			Number: big.NewInt(int64(latest)), // #nosec G115 -- Hardcoded in tests so no risk of overflow
			Time:   latest,
		}
	}
	e.mutex.Unlock()
}

func (e *mockConnectorForBatchPoller) setBlockNumbersTwice(finalized1, safe1, latest1, finalized2, safe2, latest2 uint64) {
	e.mutex.Lock()
	e.blockNumbers = []uint64{finalized1, safe1, finalized2, safe2}
	if latest1 != 0 {
		e.headSink <- &ethTypes.Header{
			Number: big.NewInt(int64(latest1)), // #nosec G115 -- Hardcoded in tests so no risk of overflow
			Time:   latest1,
		}
	}
	if latest2 != 0 {
		e.headSink <- &ethTypes.Header{
			Number: big.NewInt(int64(latest2)), // #nosec G115 -- Hardcoded in tests so no risk of overflow
			Time:   latest2,
		}
	}
	e.mutex.Unlock()
}

func (e *mockConnectorForBatchPoller) expectedHash() ethCommon.Hash {
	return ethCommon.HexToHash("0xfc8b62a31110121c57cfcccfaf2b147cc2c13b6d01bde4737846cefd29f045cf")
}

func (e *mockConnectorForBatchPoller) Client() *ethClient.Client {
	return e.client
}

type mockSubscription struct {
	errC chan error
}

func (m mockSubscription) Unsubscribe() {

}

func (m mockSubscription) Err() <-chan error {
	return m.errC
}

func (e *mockConnectorForBatchPoller) SubscribeNewHead(ctx context.Context, ch chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	e.headSink = ch
	return mockSubscription{}, nil
}

func batchShouldHaveAllThree(t *testing.T, block []*NewBlock, blockNum uint64, expectedHash ethCommon.Hash) {
	require.Equal(t, 3, len(block))
	hasFinalized := false
	hasSafe := false
	hasLatest := false
	for _, b := range block {
		assert.Equal(t, uint64(blockNum), b.Number.Uint64())
		if b.Finality == Finalized {
			hasFinalized = true
			assert.Equal(t, expectedHash, b.Hash)
		} else if b.Finality == Safe {
			hasSafe = true
			assert.Equal(t, expectedHash, b.Hash)
		} else if b.Finality == Latest {
			hasLatest = true
			// Can't check hash on latest because it's generated on the fly by geth.
		}
	}
	assert.True(t, hasFinalized)
	assert.True(t, hasSafe)
	assert.True(t, hasLatest)
}

func batchShouldHaveLatestOnly(t *testing.T, block []*NewBlock, blockNum uint64) {
	require.Equal(t, 1, len(block))
	assert.Equal(t, uint64(blockNum), block[0].Number.Uint64())
	assert.Equal(t, Latest, block[0].Finality)
}

func batchShouldHaveSafeAndFinalizedButNotLatest(t *testing.T, block []*NewBlock, blockNum uint64, expectedHash ethCommon.Hash) {
	require.Equal(t, 2, len(block))
	assert.Equal(t, uint64(blockNum), block[0].Number.Uint64())
	assert.Equal(t, Finalized, block[0].Finality)
	assert.Equal(t, expectedHash, block[0].Hash)
	assert.Equal(t, uint64(blockNum), block[1].Number.Uint64())
	assert.Equal(t, Safe, block[1].Finality)
	assert.Equal(t, expectedHash, block[1].Hash)
}

// TestBatchPoller is one big, ugly test because of all the set up required.
func TestBatchPoller(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	baseConnector := mockConnectorForBatchPoller{blockNumbers: []uint64{}}
	poller := NewBatchPollConnector(ctx, logger, &baseConnector, true, 1*time.Millisecond)

	// The go routine will post results here.
	var mutex sync.Mutex
	var block []*NewBlock
	var publishedErr error
	var publishedSubErr error // This should never be set.

	// Set the initial finalized and safe blocks.
	baseConnector.setBlockNumbers(0x309a0c, 0x309a0c, 0)

	// Subscribe for events to be processed by our go routine.
	headSink := make(chan *NewBlock, 2)
	errC := make(chan error)

	headerSubscription, subErr := poller.SubscribeForBlocks(ctx, errC, headSink)
	require.NoError(t, subErr)
	require.NotNil(t, headerSubscription)
	defer headerSubscription.Unsubscribe()

	// Create a go routine to consume the output of the poller.
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case thisErr := <-errC:
				mutex.Lock()
				publishedErr = thisErr
				mutex.Unlock()
			case thisErr := <-headerSubscription.Err():
				mutex.Lock()
				publishedSubErr = thisErr
				mutex.Unlock()
			case thisBlock := <-headSink:
				require.NotNil(t, thisBlock)
				mutex.Lock()
				block = append(block, thisBlock)
				mutex.Unlock()
			}
		}
	}()

	// First sleep a bit and make sure there were no start up errors and the initial blocks were published.
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	batchShouldHaveSafeAndFinalizedButNotLatest(t, block, 0x309a0c, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	// Post the first new block and verify we get it.
	baseConnector.setBlockNumbers(0x309a0d, 0x309a0d, 0x309a0d)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	batchShouldHaveAllThree(t, block, 0x309a0d, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	// Sleep some more and verify we don't see any more blocks, since we haven't posted a new one.
	baseConnector.setBlockNumbers(0x309a0d, 0x309a0d, 0)
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	require.Nil(t, block)
	mutex.Unlock()

	// Post the next block and verify we get it.
	baseConnector.setBlockNumbers(0x309a0e, 0x309a0e, 0x309a0e)
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	batchShouldHaveAllThree(t, block, 0x309a0e, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	// Post the next block but mark it as not finalized, so we should only see latest.
	mutex.Lock()
	baseConnector.setBlockNumbers(0x309a0e, 0x309a0e, 0x309a0f)

	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	batchShouldHaveLatestOnly(t, block, 0x309a0f)
	block = nil
	mutex.Unlock()

	// Once it goes finalized we should see safe and finalized, but not latest again.
	mutex.Lock()
	baseConnector.setBlockNumbers(0x309a0f, 0x309a0f, 0)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	batchShouldHaveSafeAndFinalizedButNotLatest(t, block, 0x309a0f, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	// Post old finalized and safe blocks and we should not hear about them.
	baseConnector.setBlockNumbers(0x309a0c, 0x309a0c, 0)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	require.Nil(t, block)
	mutex.Unlock()

	// But we should keep going when we get a new one.
	baseConnector.setBlockNumbers(0x309a10, 0x309a10, 0x309a10)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	batchShouldHaveAllThree(t, block, 0x309a10, baseConnector.expectedHash())
	block = nil

	// If there's a gap in the blocks, we play out the gap for finalized and safe, but not latest.
	baseConnector.setBlockNumbersTwice(
		0x309a12, // New Finalized
		0x309a12, // New Safe
		0x309a12, // New Latest)
		0x309a11, // Gap Finalized
		0x309a11, // Gap Safe
		0,        // Latest blocks don't get replayed since they come from the head sync.
	)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	require.Equal(t, 5, len(block))

	// We can't determine how we will see latest vs. finalized / safe. Spit them up so we can verify them independently, but preserve the order.
	{
		latestBlocks := []*NewBlock{}
		otherBlocks := []*NewBlock{}
		for _, b := range block {
			if b.Finality == Latest {
				latestBlocks = append(latestBlocks, b)
			} else {
				otherBlocks = append(otherBlocks, b)
			}
		}

		// We don't gap fill latest blocks, so we should only see one. Note that we can't verify hash on latest because it's generated on the fly by geth.
		require.Equal(t, 1, len(latestBlocks))
		assert.Equal(t, uint64(0x309a12), latestBlocks[0].Number.Uint64())

		// We should see two finalized, followed by two safe.
		require.Equal(t, 4, len(otherBlocks))
		assert.Equal(t, uint64(0x309a11), otherBlocks[0].Number.Uint64())
		assert.Equal(t, Finalized, otherBlocks[0].Finality)
		assert.Equal(t, baseConnector.expectedHash(), otherBlocks[0].Hash)

		assert.Equal(t, uint64(0x309a12), otherBlocks[1].Number.Uint64())
		assert.Equal(t, Finalized, otherBlocks[1].Finality)
		assert.Equal(t, baseConnector.expectedHash(), otherBlocks[1].Hash)

		assert.Equal(t, uint64(0x309a11), otherBlocks[2].Number.Uint64())
		assert.Equal(t, Safe, otherBlocks[2].Finality)
		assert.Equal(t, baseConnector.expectedHash(), otherBlocks[2].Hash)

		assert.Equal(t, uint64(0x309a12), otherBlocks[3].Number.Uint64())
		assert.Equal(t, Safe, otherBlocks[3].Finality)
		assert.Equal(t, baseConnector.expectedHash(), otherBlocks[3].Hash)
	}

	block = nil
	mutex.Unlock()

	// A single RPC error should not be returned to us.
	baseConnector.setSingleError(fmt.Errorf("RPC failed"))

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	assert.Equal(t, 0, len(block))
	block = nil
	mutex.Unlock()

	// And we should be able to continue after a single error.
	baseConnector.setBlockNumbers(0x309a13, 0x309a13, 0x309a13)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, publishedErr)
	require.NoError(t, publishedSubErr)
	batchShouldHaveAllThree(t, block, 0x309a13, baseConnector.expectedHash())
	block = nil
	mutex.Unlock()

	//
	// NOTE: This should be the last part of this test because it kills the poller!
	//

	// A persistent RPC error should be returned to us.
	publishedErr = nil
	baseConnector.setError(fmt.Errorf("RPC failed"))

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	assert.Error(t, publishedErr)
	require.NoError(t, publishedSubErr)
	assert.Nil(t, block)
	baseConnector.setError(nil)
	publishedErr = nil
	mutex.Unlock()
}
