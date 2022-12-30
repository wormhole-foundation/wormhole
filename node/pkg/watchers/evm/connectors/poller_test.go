package connectors

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

// TestBlockPoller is one big, ugly test because of all the set up required.
func TestBlockPoller(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	mockConnector := MockConnector{}

	finalizer := NewMockFinalizer(true) // Start by assuming blocks are finalized.
	assert.NotNil(t, finalizer)

	poller := &BlockPollConnector{
		Connector:    &mockConnector,
		Delay:        1 * time.Millisecond,
		useFinalized: false,
		finalizer:    finalizer,
	}

	// Set the starting block.
	mockConnector.SetBlockNumber(0x309a0c)

	// The go routines will post results here.
	var mutex sync.Mutex
	var block *NewBlock
	var err error
	var pollerStatus int

	// Start the poller running.
	go func() {
		mutex.Lock()
		pollerStatus = 1
		mutex.Unlock()
		err := poller.run(ctx, logger)
		require.NoError(t, err)
		mutex.Lock()
		pollerStatus = 2
		mutex.Unlock()
	}()

	// Subscribe for events to be processed by our go routine.
	headSink := make(chan *NewBlock, 2)
	headerSubscription, suberr := poller.SubscribeForBlocks(ctx, headSink)
	require.NoError(t, suberr)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case thisErr := <-headerSubscription.Err():
				mutex.Lock()
				err = thisErr
				mutex.Unlock()
			case thisBlock := <-headSink:
				require.NotNil(t, thisBlock)
				mutex.Lock()
				block = thisBlock
				mutex.Unlock()
			}
		}
	}()

	// First sleep a bit and make sure there were no start up errors.
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	assert.Nil(t, block)
	mutex.Unlock()

	// Post the first new block and verify we get it.
	mockConnector.SetBlockNumber(0x309a0d)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a0d), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// Sleep some more and verify we don't see any more blocks, since we haven't posted a new one.
	mockConnector.SetBlockNumber(0x309a0d)
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.Nil(t, block)
	mutex.Unlock()

	// Post the next block and verify we get it.
	mockConnector.SetBlockNumber(0x309a0e)
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a0e), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// Post the next block but mark it as not finalized, so we shouldn't see it yet.
	mutex.Lock()
	finalizer.SetFinalized(false)
	mockConnector.SetBlockNumber(0x309a0f)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.Nil(t, block)
	mutex.Unlock()

	// Once it goes finalized we should see it.
	mutex.Lock()
	finalizer.SetFinalized(true)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a0f), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// An RPC error should be returned to us.
	err = nil
	mockConnector.SetError(fmt.Errorf("RPC failed"))

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	assert.Error(t, err)
	assert.Nil(t, block)
	mockConnector.SetError(nil)
	err = nil
	mutex.Unlock()

	// Post the next block and verify we get it (so we survived the RPC error).
	mockConnector.SetBlockNumber(0x309a10)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a10), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// Post an old block and we should not hear about it.
	mockConnector.SetBlockNumber(0x309a0c)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.Nil(t, block)
	mutex.Unlock()

	// But we should keep going when we get a new one.
	mockConnector.SetBlockNumber(0x309a11)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a11), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// If there's a gap in the blocks, we should keep going.
	mockConnector.SetBlockNumber(0x309a13)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a13), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// Should retry on a transient error and be able to continue.
	mockConnector.SetSingleError(fmt.Errorf("RPC failed"))
	mockConnector.SetBlockNumber(0x309a14)

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.Equal(t, 1, pollerStatus)
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a14), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()
}
