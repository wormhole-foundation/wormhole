package evm

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	bts := NewBlocksByTimestamp(BTS_MAX_BLOCKS)

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
	bts := NewBlocksByTimestamp(BTS_MAX_BLOCKS)

	assert.NoError(t, bts.AddLatest(1698621628, 420))
	assert.NoError(t, bts.AddLatest(1698621628, 421))
	assert.NoError(t, bts.AddLatest(1698621628, 422))
	assert.Error(t, bts.AddLatest(1698621627, 423)) // The timestamp going down is an error.
	assert.Error(t, bts.AddLatest(1698621629, 422)) // Even if the timestamp goes up, the block must also go up.
	assert.NoError(t, bts.AddLatest(1698621629, 423))
	assert.Equal(t, 4, len(bts.cache))
	assert.True(t, cacheIsValid(t, bts))
}

func TestBlocksByTimestamp_AddLatestShouldTrimTheCache(t *testing.T) {
	bts := NewBlocksByTimestamp(5)

	require.NoError(t, bts.AddLatest(1698621628, 420))
	require.NoError(t, bts.AddLatest(1698621628, 421))
	require.NoError(t, bts.AddLatest(1698621628, 422))
	require.NoError(t, bts.AddLatest(1698621628, 423))
	require.NoError(t, bts.AddLatest(1698621629, 424))
	require.Equal(t, 5, len(bts.cache), 5)
	require.True(t, cacheIsValid(t, bts))

	assert.NoError(t, bts.AddLatest(1698621629, 425))
	assert.Equal(t, 5, len(bts.cache))

	assert.True(t, cacheIsValid(t, bts))
	assert.Equal(t, uint64(421), bts.cache[0].BlockNum)
	assert.Equal(t, uint64(422), bts.cache[1].BlockNum)
	assert.Equal(t, uint64(423), bts.cache[2].BlockNum)
	assert.Equal(t, uint64(424), bts.cache[3].BlockNum)
	assert.Equal(t, uint64(425), bts.cache[4].BlockNum)
}

func TestBlocksByTimestamp_AddBatch(t *testing.T) {
	bts := NewBlocksByTimestamp(BTS_MAX_BLOCKS)

	// First create a cache with some gaps in it.
	require.NoError(t, bts.AddLatest(1698621628, 420))
	require.NoError(t, bts.AddLatest(1698621628, 430))
	require.NoError(t, bts.AddLatest(1698621728, 440))
	require.NoError(t, bts.AddLatest(1698621729, 450))
	require.NoError(t, bts.AddLatest(1698621828, 460))
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
	bts := NewBlocksByTimestamp(8)

	// First create a cache with some gaps in it.
	require.NoError(t, bts.AddLatest(1698621628, 420))
	require.NoError(t, bts.AddLatest(1698621628, 430))
	require.NoError(t, bts.AddLatest(1698621728, 440))
	require.NoError(t, bts.AddLatest(1698621729, 450))
	require.NoError(t, bts.AddLatest(1698621828, 460))
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
	bts := NewBlocksByTimestamp(BTS_MAX_BLOCKS)

	// Empty cache.
	prev, next, found := bts.LookUp(1698621627)
	assert.False(t, found)
	assert.Equal(t, uint64(0), prev)
	assert.Equal(t, uint64(0), next)

	require.NoError(t, bts.AddLatest(1698621528, 420))
	require.NoError(t, bts.AddLatest(1698621528, 421))
	require.NoError(t, bts.AddLatest(1698621628, 422))
	require.NoError(t, bts.AddLatest(1698621728, 423))
	require.NoError(t, bts.AddLatest(1698621728, 424))
	require.NoError(t, bts.AddLatest(1698621828, 426))
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
