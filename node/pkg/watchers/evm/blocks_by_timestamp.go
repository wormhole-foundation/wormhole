package evm

import (
	"sort"
	"sync"

	"go.uber.org/zap"
)

const (
	BTS_MAX_BLOCKS = 10000
)

type (
	BlocksByTimestamp struct {
		// cache is ordered by timestamp, blockNum. There may be multiple entries for the same timestamp, but not the same block.
		cache Blocks

		// maxCacheSize is used to trim the cache.
		maxCacheSize int

		// unsafeDevMode is used to suppress warnings in dev mode.
		unsafeDevMode bool

		// mutex is used to protect the cache.
		mutex sync.Mutex
	}

	Blocks []Block

	Block struct {
		Timestamp uint64
		BlockNum  uint64
	}
)

// NewBlocksByTimestamp creates an empty cache of blocks by timestamp.
func NewBlocksByTimestamp(maxCacheSize int, unsafeDevMode bool) *BlocksByTimestamp {
	return &BlocksByTimestamp{
		cache:         Blocks{},
		maxCacheSize:  maxCacheSize,
		unsafeDevMode: unsafeDevMode,
	}
}

// AddLatest adds a block to the end of the cache. This is meant to be used in the normal scenario when a new latest block is received. If the specified
// timestamp or block number is less than the latest in the cache (most likely a rollback), the cache will be truncated and the new value inserted.
func (bts *BlocksByTimestamp) AddLatest(logger *zap.Logger, timestamp uint64, blockNum uint64) {
	bts.mutex.Lock()
	defer bts.mutex.Unlock()
	l := len(bts.cache)
	if l > 0 && (blockNum <= bts.cache[l-1].BlockNum || timestamp < bts.cache[l-1].Timestamp) {
		// The cache is in order of both timestamp and block number. Search backwards until we find the entry where the block number is less than the one
		// passed in and the timestamp is less than or equal to the one passed in. We then truncate everything after that before adding the new one.
		idx := l - 1
		for ; idx >= 0; idx-- {
			if bts.cache[idx].BlockNum < blockNum && bts.cache[idx].Timestamp <= timestamp {
				break
			}
		}

		// Anvil trips this when using `anvil_mine`
		if !bts.unsafeDevMode {
			logger.Warn("rollback detected in timestamp cache",
				zap.Uint64("oldLatestBlockNum", bts.cache[l-1].BlockNum),
				zap.Uint64("oldLatestTimestamp", bts.cache[l-1].Timestamp),
				zap.Uint64("newLatestBlockNum", blockNum),
				zap.Uint64("newLatestTimestamp", timestamp),
			)
		}
		bts.cache = bts.cache[:idx+1]
	}

	bts.cache = append(bts.cache, Block{Timestamp: timestamp, BlockNum: blockNum})

	if len(bts.cache) > bts.maxCacheSize {
		bts.cache = bts.cache[1:]
	}
	logger.Debug("cache updated", zap.Int("len", len(bts.cache)), zap.Uint64("lastTimestamp", timestamp), zap.Uint64("lastBlockNum", blockNum))
}

// AddBatch adds a batch of blocks to the cache. This is meant to be used for backfilling the cache. It makes sure there are no duplicate blocks and regenerates the cache in the correct order by timestamp.
func (bts *BlocksByTimestamp) AddBatch(blocks Blocks) {
	bts.mutex.Lock()
	defer bts.mutex.Unlock()

	// First build a map of all the existing blocks so we can avoid duplicates.
	blockMap := make(map[uint64]uint64)
	for _, block := range bts.cache {
		blockMap[block.BlockNum] = block.Timestamp
	}

	// Now add the new blocks to the map, overwriting any duplicates. (Maybe there was a reorg. . .)
	for _, block := range blocks {
		blockMap[block.BlockNum] = block.Timestamp
	}

	// Now put everything into the cache in random order.
	cache := Blocks{}
	for blockNum, timestamp := range blockMap {
		cache = append(cache, Block{Timestamp: timestamp, BlockNum: blockNum})
	}

	// Sort the cache into timestamp order.
	sort.SliceStable(cache, func(i, j int) bool {
		return cache[i].Cmp(cache[j]) < 0
	})

	if len(cache) > bts.maxCacheSize {
		// Trim the cache.
		trimIdx := len(cache) - bts.maxCacheSize
		cache = cache[trimIdx:]
	}

	bts.cache = cache
}

// LookUp searches the cache for the specified timestamp and returns the blocks surrounding that timestamp. It also returns true if the results are complete or false if they are not.
// The following rules apply:
// - If timestamp is less than the first timestamp in the cache, it returns (0, <theFirstBlockInTheCache>, false)
// - If timestamp is greater than or equal to the last timestamp in the cache, it returns (<theLastBlockInTheCache>, 0, false)
// - If timestamp exactly matches one in the cache, it returns (<theLastBlockForThatTimestamp>, <theFirstBlockForTheNextTimestamp>, true)
// - If timestamp is not in the cache, but there are blocks around it, it returns (<theLastBlockForThePreviousTimestamp>, <theFirstBlockForTheNextTimestamp>, false)
func (bts *BlocksByTimestamp) LookUp(timestamp uint64) (uint64, uint64, bool) {
	bts.mutex.Lock()
	defer bts.mutex.Unlock()

	if len(bts.cache) == 0 { // Empty cache.
		return 0, 0, false
	}

	if timestamp < bts.cache[0].Timestamp { // Before the start of the cache.
		return 0, bts.cache[0].BlockNum, false
	}

	if timestamp >= bts.cache[len(bts.cache)-1].Timestamp { // After the end of the cache (including matching the final timestamp).
		return bts.cache[len(bts.cache)-1].BlockNum, 0, false
	}

	// The search returns the first entry where the timestamp is greater than requested.
	idx := bts.cache.SearchForTimestamp(timestamp)

	// If the two blocks are adjacent, then we found what we are looking for.
	found := bts.cache[idx-1].BlockNum+1 == bts.cache[idx].BlockNum
	return bts.cache[idx-1].BlockNum, bts.cache[idx].BlockNum, found
}

func (blocks Blocks) SearchForTimestamp(timestamp uint64) int {
	return sort.Search(len(blocks), func(i int) bool { return blocks[i].Timestamp > timestamp })
}

// Cmp compares two blocks, returning the usual -1, 0, +1.
func (lhs Block) Cmp(rhs Block) int {
	if lhs.Timestamp < rhs.Timestamp {
		return -1
	}
	if lhs.Timestamp > rhs.Timestamp {
		return 1
	}
	if lhs.BlockNum < rhs.BlockNum {
		return -1
	}
	if lhs.BlockNum > rhs.BlockNum {
		return 1
	}

	return 0
}
