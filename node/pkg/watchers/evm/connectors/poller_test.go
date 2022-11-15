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

	// Start the poller running.
	go func() {
		err := poller.run(ctx, logger)
		require.NoError(t, err)
	}()

	// Subscribe for events to be processed by our go routine.
	headSink := make(chan *NewBlock, 2)
	headerSubscription, suberr := poller.SubscribeForBlocks(ctx, headSink)
	require.NoError(t, suberr)

	// The go routine will post results here.
	var mutex sync.Mutex
	var block *NewBlock
	var err error

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
	require.NoError(t, err)
	assert.Nil(t, block)
	mutex.Unlock()

	// Post the first new block and verify we get it.
	mutex.Lock()
	mockConnector.SetBlockNumber(0x309a0d)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
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
	require.NoError(t, err)
	require.Nil(t, block)
	mutex.Unlock()

	// Post the next block and verify we get it.
	mockConnector.SetBlockNumber(0x309a0e)
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
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
	require.NoError(t, err)
	require.Nil(t, block)
	mutex.Unlock()

	// Once it goes finalized we should see it.
	mutex.Lock()
	finalizer.SetFinalized(true)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a0f), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// An RPC error should be returned to us.
	mutex.Lock()
	err = nil
	mockConnector.SetError(fmt.Errorf("RPC failed"))
	mutex.Unlock()
	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	assert.Error(t, err)
	assert.Nil(t, block)
	mockConnector.SetError(nil)
	err = nil
	mutex.Unlock()

	// Post the next block and verify we get it (so we survived the RPC error).
	mutex.Lock()
	mockConnector.SetBlockNumber(0x309a10)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a10), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// Post an old block and we should not hear about it.
	mutex.Lock()
	mockConnector.SetBlockNumber(0x309a0c)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, err)
	require.Nil(t, block)
	mutex.Unlock()

	// But we should keep going when we get a new one.
	mutex.Lock()
	mockConnector.SetBlockNumber(0x309a11)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a11), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// If there's a gap in the blocks, we should keep going.
	mutex.Lock()
	mockConnector.SetBlockNumber(0x309a13)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a13), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()

	// Should retry on a transient error and be able to continue.
	mutex.Lock()
	mockConnector.SetSingleError(fmt.Errorf("RPC failed"))
	mockConnector.SetBlockNumber(0x309a14)
	mutex.Unlock()

	time.Sleep(10 * time.Millisecond)
	mutex.Lock()
	require.NoError(t, err)
	require.NotNil(t, block)
	assert.Equal(t, uint64(0x309a14), block.Number.Uint64())
	assert.Equal(t, mockConnector.ExpectedHash(), block.Hash)
	block = nil
	mutex.Unlock()
}
