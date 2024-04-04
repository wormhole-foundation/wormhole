package evm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

func cacheIsValid(t *testing.T, bts *BlocksByTimestamp) bool {
	t.Helper()
	prevBlock := Block{}
	for idx := range bts.cache {
		// fmt.Println("Compare: prev: ", prevBlock, ", this: ", bts.cache[idx], ": ", prevBlock.Cmp(bts.cache[idx]))
		if prevBlock.Cmp(bts.cache[idx]) != -1 {
			return false
		}
		prevBlock = bts.cache[idx]
	}

	return true
}

func TestBlocksByTimestamp_TestCacheIsValid(t *testing.T) {
	bts := NewBlocksByTimestamp(BTS_MAX_BLOCKS, false)

	// Empty cache is valid.
	assert.True(t, cacheIsValid(t, bts))

	bts.cache = append(bts.cache, Block{Timestamp: 1698621628, BlockNum: 420}) // 0
	bts.cache = append(bts.cache, Block{Timestamp: 1698621629, BlockNum: 430}) // 1
	bts.cache = append(bts.cache, Block{Timestamp: 1698621629, BlockNum: 440}) // 2
	bts.cache = append(bts.cache, Block{Timestamp: 1698621631, BlockNum: 450}) // 3
	bts.cache = append(bts.cache, Block{Timestamp: 1698621632, BlockNum: 460}) // 4

	// Make sure a valid cache is valid.
	assert.True(t, cacheIsValid(t, bts))

	// Timestamps match but duplicate block should fail.
	bts.cache[2] = Block{Timestamp: 1698621629, BlockNum: 430}
	assert.False(t, cacheIsValid(t, bts))

	// Restore things.
	bts.cache[2] = Block{Timestamp: 1698621629, BlockNum: 440}
	assert.True(t, cacheIsValid(t, bts))

	// Timestamps match but block out of order should fail.
	bts.cache[2] = Block{Timestamp: 1698621629, BlockNum: 425}
	assert.False(t, cacheIsValid(t, bts))

	// Restore things.
	bts.cache[2] = Block{Timestamp: 1698621629, BlockNum: 440}
	assert.True(t, cacheIsValid(t, bts))

	// Timestamps out of order should fail.
	bts.cache[2] = Block{Timestamp: 1698621620, BlockNum: 440}
	assert.False(t, cacheIsValid(t, bts))
}

func TestBlocksByTimestamp_AddLatest(t *testing.T) {
	logger := zap.NewNop()
	bts := NewBlocksByTimestamp(BTS_MAX_BLOCKS, false)

	bts.AddLatest(logger, 1698621628, 420)
	bts.AddLatest(logger, 1698621628, 421)
	bts.AddLatest(logger, 1698621628, 422)
	bts.AddLatest(logger, 1698621629, 423)
	bts.AddLatest(logger, 1698621630, 424)
	bts.AddLatest(logger, 1698621630, 425)
	require.Equal(t, 6, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))

	// Timestamp going back should trim by timestamp.
	bts.AddLatest(logger, 1698621629, 427)
	require.Equal(t, 5, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))
	assert.Equal(t, uint64(427), bts.cache[4].BlockNum)
	assert.Equal(t, uint64(1698621629), bts.cache[4].Timestamp)

	// Block number only going back should trim by block number only.
	bts.AddLatest(logger, 1698621629, 426)
	require.Equal(t, 5, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))
	assert.Equal(t, uint64(426), bts.cache[4].BlockNum)
	assert.Equal(t, uint64(1698621629), bts.cache[4].Timestamp)
}

func TestBlocksByTimestamp_AddLatestRollbackEverything(t *testing.T) {
	logger := zap.NewNop()
	bts := NewBlocksByTimestamp(BTS_MAX_BLOCKS, false)

	bts.AddLatest(logger, 1698621628, 420)
	require.Equal(t, 1, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))

	// Rollback before only block in cache, but not before the timestamp.
	bts.AddLatest(logger, 1698621628, 419)
	require.Equal(t, 1, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))
	assert.Equal(t, uint64(419), bts.cache[0].BlockNum)
	assert.Equal(t, uint64(1698621628), bts.cache[0].Timestamp)

	// Rollback before only timestamp and block in cache.
	bts.AddLatest(logger, 1698621627, 418)
	require.Equal(t, 1, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))
	assert.Equal(t, uint64(418), bts.cache[0].BlockNum)
	assert.Equal(t, uint64(1698621627), bts.cache[0].Timestamp)

	// Add two more blocks at the end, giving a total of three entries.
	bts.AddLatest(logger, 1698621628, 419)
	bts.AddLatest(logger, 1698621629, 420)
	require.Equal(t, 3, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))

	// Rollback before first block in cache.
	bts.AddLatest(logger, 1698621627, 417)
	require.Equal(t, 1, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))
	assert.Equal(t, uint64(417), bts.cache[0].BlockNum)
	assert.Equal(t, uint64(1698621627), bts.cache[0].Timestamp)

	// Add two more blocks at the end, giving a total of three entries.
	bts.AddLatest(logger, 1698621628, 418)
	bts.AddLatest(logger, 1698621629, 419)
	require.Equal(t, 3, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))

	// Rollback before first timestamp and block in cache.
	bts.AddLatest(logger, 1698621626, 416)
	require.Equal(t, 1, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))
	assert.Equal(t, uint64(416), bts.cache[0].BlockNum)
	assert.Equal(t, uint64(1698621626), bts.cache[0].Timestamp)
}

func TestBlocksByTimestamp_AddLatestShouldTrimTheCache(t *testing.T) {
	logger := zap.NewNop()
	bts := NewBlocksByTimestamp(5, false)

	bts.AddLatest(logger, 1698621628, 420)
	bts.AddLatest(logger, 1698621628, 421)
	bts.AddLatest(logger, 1698621628, 422)
	bts.AddLatest(logger, 1698621628, 423)
	bts.AddLatest(logger, 1698621629, 424)
	require.Equal(t, 5, len(bts.cache), 5)
	require.True(t, cacheIsValid(t, bts))

	bts.AddLatest(logger, 1698621629, 425)
	assert.Equal(t, 5, len(bts.cache))

	assert.True(t, cacheIsValid(t, bts))
	assert.Equal(t, uint64(421), bts.cache[0].BlockNum)
	assert.Equal(t, uint64(422), bts.cache[1].BlockNum)
	assert.Equal(t, uint64(423), bts.cache[2].BlockNum)
	assert.Equal(t, uint64(424), bts.cache[3].BlockNum)
	assert.Equal(t, uint64(425), bts.cache[4].BlockNum)
}

func TestBlocksByTimestamp_AddBatch(t *testing.T) {
	logger := zap.NewNop()
	bts := NewBlocksByTimestamp(BTS_MAX_BLOCKS, false)

	// First create a cache with some gaps in it.
	bts.AddLatest(logger, 1698621628, 420)
	bts.AddLatest(logger, 1698621628, 430)
	bts.AddLatest(logger, 1698621728, 440)
	bts.AddLatest(logger, 1698621729, 450)
	bts.AddLatest(logger, 1698621828, 460)
	require.Equal(t, 5, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))

	batch := []Block{
		// Add a couple afterwards.
		{Timestamp: 1698621928, BlockNum: 470},
		{Timestamp: 1698621929, BlockNum: 480},

		// Add a few in the middle.
		{Timestamp: 1698621630, BlockNum: 431},
		{Timestamp: 1698621631, BlockNum: 432},

		// Add one at the front.
		{Timestamp: 1698621528, BlockNum: 410},
	}

	bts.AddBatch(batch)
	assert.Equal(t, 10, len(bts.cache))
	assert.True(t, cacheIsValid(t, bts))
}

func TestBlocksByTimestamp_AddBatchShouldTrim(t *testing.T) {
	logger := zap.NewNop()
	bts := NewBlocksByTimestamp(8, false)

	// First create a cache with some gaps in it.
	bts.AddLatest(logger, 1698621628, 420)
	bts.AddLatest(logger, 1698621628, 430)
	bts.AddLatest(logger, 1698621728, 440)
	bts.AddLatest(logger, 1698621729, 450)
	bts.AddLatest(logger, 1698621828, 460)
	require.Equal(t, 5, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))

	batch := []Block{
		// Add a couple afterwards.
		{Timestamp: 1698621928, BlockNum: 470},
		{Timestamp: 1698621929, BlockNum: 480},

		// Add a few in the middle.
		{Timestamp: 1698621630, BlockNum: 431},
		{Timestamp: 1698621631, BlockNum: 432},

		// Add one at the front.
		{Timestamp: 1698621528, BlockNum: 410},
	}

	bts.AddBatch(batch)
	assert.Equal(t, 8, len(bts.cache))
	assert.True(t, cacheIsValid(t, bts))
	assert.Equal(t, uint64(430), bts.cache[0].BlockNum)
	assert.Equal(t, uint64(431), bts.cache[1].BlockNum)
	assert.Equal(t, uint64(432), bts.cache[2].BlockNum)
	assert.Equal(t, uint64(440), bts.cache[3].BlockNum)
	assert.Equal(t, uint64(450), bts.cache[4].BlockNum)
	assert.Equal(t, uint64(460), bts.cache[5].BlockNum)
	assert.Equal(t, uint64(470), bts.cache[6].BlockNum)
	assert.Equal(t, uint64(480), bts.cache[7].BlockNum)
}

func TestBlocksByTimestamp_SearchForTimestamp(t *testing.T) {
	blocks := Blocks{
		{Timestamp: 1698621228, BlockNum: 420}, // 0
		{Timestamp: 1698621328, BlockNum: 430}, // 1
		{Timestamp: 1698621428, BlockNum: 440}, // 2
		{Timestamp: 1698621428, BlockNum: 450}, // 3
		{Timestamp: 1698621528, BlockNum: 460}, // 4
	}

	// Returns the first entry where the timestamp is greater than requested.
	assert.Equal(t, 0, blocks.SearchForTimestamp(1698621128))
	assert.Equal(t, 1, blocks.SearchForTimestamp(1698621228))
	assert.Equal(t, 2, blocks.SearchForTimestamp(1698621328))
	assert.Equal(t, 4, blocks.SearchForTimestamp(1698621428))
	assert.Equal(t, 5, blocks.SearchForTimestamp(1698621528))
	assert.Equal(t, 5, blocks.SearchForTimestamp(1698621628))
}

func TestBlocksByTimestamp_LookUp(t *testing.T) {
	logger := zap.NewNop()
	bts := NewBlocksByTimestamp(BTS_MAX_BLOCKS, false)

	// Empty cache.
	prev, next, found := bts.LookUp(1698621627)
	assert.False(t, found)
	assert.Equal(t, uint64(0), prev)
	assert.Equal(t, uint64(0), next)

	bts.AddLatest(logger, 1698621528, 420)
	bts.AddLatest(logger, 1698621528, 421)
	bts.AddLatest(logger, 1698621628, 422)
	bts.AddLatest(logger, 1698621728, 423)
	bts.AddLatest(logger, 1698621728, 424)
	bts.AddLatest(logger, 1698621828, 426)
	require.Equal(t, 6, len(bts.cache))
	require.True(t, cacheIsValid(t, bts))

	// Before the beginning of the cache.
	prev, next, found = bts.LookUp(1698621527)
	assert.False(t, found)
	assert.Equal(t, uint64(0), prev)
	assert.Equal(t, uint64(420), next)

	// After the end of the cache.
	prev, next, found = bts.LookUp(1698621928)
	assert.False(t, found)
	assert.Equal(t, uint64(426), prev)
	assert.Equal(t, uint64(0), next)

	// Last timestamp in the cache.
	prev, next, found = bts.LookUp(1698621828)
	assert.False(t, found)
	assert.Equal(t, uint64(426), prev)
	assert.Equal(t, uint64(0), next)

	// In the cache, one block for the timestamp.
	prev, next, found = bts.LookUp(1698621628)
	assert.True(t, found)
	assert.Equal(t, uint64(422), prev)
	assert.Equal(t, uint64(423), next)

	// In the cache, multiple blocks for the same timestamp.
	prev, next, found = bts.LookUp(1698621528)
	assert.True(t, found)
	assert.Equal(t, uint64(421), prev)
	assert.Equal(t, uint64(422), next)

	// Not in the cache, no gap.
	prev, next, found = bts.LookUp(1698621600)
	assert.True(t, found)
	assert.Equal(t, uint64(421), prev)
	assert.Equal(t, uint64(422), next)

	// Not in the cache, gap.
	prev, next, found = bts.LookUp(1698621800)
	assert.False(t, found)
	assert.Equal(t, uint64(424), prev)
	assert.Equal(t, uint64(426), next)
}
