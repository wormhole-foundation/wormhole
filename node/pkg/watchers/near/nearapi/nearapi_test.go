package nearapi_test

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	mockserver "github.com/certusone/wormhole/node/pkg/watchers/near/nearapi/mock"
	"go.uber.org/zap/zaptest"

	"github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
	"github.com/stretchr/testify/assert"
)

type (
	MockNearRpc struct{}
)

func TestNewBlockFromBytes(t *testing.T) {
	blockBytes, err := os.ReadFile("mock/static/block.json")
	assert.NoError(t, err)

	block, err := nearapi.NewBlockFromBytes(blockBytes)
	assert.NoError(t, err)

	assert.Equal(t, block.Header.Hash, "NSM5RDZDF7uxGWiUwhBqJcqCEw6g7axx4TxGYB7XZVt")
	assert.Equal(t, block.Header.Height, uint64(75398642))
	assert.Equal(t, block.Header.LastFinalBlock, "ARo7pHDH5hk1qpfdwRYtcuWh5dEjTHXwNw8wCTJb78jf")
	assert.Equal(t, block.Header.PrevBlockHash, "FqPKohapMjpemtYh8nuQAB7iVJ3rDWtAZQnRjsXVFVbB")
	assert.Equal(t, block.Header.Timestamp, uint64(1664754166351210892/1_000_000_000))
}

func TestNewChunkFromBytes(t *testing.T) {
	chunkBytes, err := os.ReadFile("mock/static/chunk.json")
	assert.NoError(t, err)

	c, err := nearapi.NewChunkFromBytes(chunkBytes)
	assert.NoError(t, err)

	assert.Equal(t, c.Hash, "CMUBdbgha1cK8zmjMnR9y9d1XjQ3rWGDqMhJWwWMQQ6c")
	assert.Equal(t, c.Height(), uint64(76648958))

	assert.Equal(t, c.Transactions()[1].Hash, "Ghke9UK93vhqVburswfd7fvYk5PZ3xNvY9xP6PvufN9U")
	assert.Equal(t, c.Transactions()[1].SignerId, "65ca40c4de59b439db917daf9f527f605b448b3f5c6d5777d0b83a78e8dcf062")
}

func TestNearApi(t *testing.T) {
	// ---Setup---
	parentCtx := context.Background()
	ctx, cancelFunc := context.WithTimeout(parentCtx, time.Second*5)
	defer cancelFunc()

	logger := zaptest.NewLogger(t)

	mockServer := mockserver.NewForwardingCachingServer(logger, "https://rpc.mainnet.near.org", "mock/apitest/", nil)
	mockHttpServer := httptest.NewServer(mockServer)

	api := nearapi.NewNearApiImpl(nearapi.NewHttpNearRpc(mockHttpServer.URL))

	// ---Test---
	b1, err := api.GetBlock(ctx, "5uNbd6DTxC3kes7JS6o5LeE7e7rF8kXHZqpzYrtVJfmz")
	assert.NoError(t, err)
	expectedBh := nearapi.BlockHeader{
		Hash:           "5uNbd6DTxC3kes7JS6o5LeE7e7rF8kXHZqpzYrtVJfmz",
		PrevBlockHash:  "HqUJVzPwJcgVKtta9BfonjUMWBhKMxpNVNgVDbHzsmwd",
		Height:         76647304,
		Timestamp:      1666276832612460208 / 1_000_000_000,
		LastFinalBlock: "HLU4HVM2SiJGbjLm41Hnq2jstawL4Pir97BsxqeN9NCw",
	}

	assert.Equal(t, b1.Header, expectedBh)

	ch := b1.ChunkHashes()

	expectedCh := []nearapi.ChunkHeader{
		{"6R6hRUHSQh6BAXFVMYiSnc4gEH5ncjCUSGF3wYB66eeV"},
		{"2QjYXzAHMZxSa5kFFTZ7eomVF3AMvBLQ5Ei4L6cjA4pu"},
		{"9NVFcm2UNNFNzi3gGXTrNQgwzdMHiA5GPSMkP7MMP5bA"},
		{"7FirWt1xXsDbjfr85FMpzy55UYBJtP6SKEWZkNMUiLBC"},
	}
	assert.Equal(t, ch, expectedCh)

	// get the same block by height
	b2, err := api.GetBlockByHeight(ctx, b1.Header.Height)
	assert.NoError(t, err)
	assert.Equal(t, b1.Header, b2.Header)
	assert.Equal(t, b1.ChunkHashes(), b2.ChunkHashes())

	// get a chunk from that block
	c, err := api.GetChunk(ctx, expectedCh[0])
	assert.NoError(t, err)
	assert.Equal(t, c.Hash, expectedCh[0].Hash)
	assert.Equal(t, c.Height(), expectedBh.Height)

	// get transactions from that chunk
	txs := c.Transactions()

	expectedTxs := []nearapi.Transaction{
		{Hash: "AFtsvPoA4zRhHY2LrD2VmFzcgWCBm351oZkHbq7D6EdH", SignerId: "1ce2567f7f49cb34cea72179223ec7fc4ba91077da3e01364d0c729dfbe26467"},
		{Hash: "2rqzjwSCRGuSgHi2xSaZ9acXJ7XDUzzZFLJCEwY2cErf", SignerId: "app.nearcrowd.near"},
		{Hash: "HLLcwgFb5cNKULGu2Vjb6qFqrgmFvEDfNBVbthNh552M", SignerId: "51491d5937b8e39fa2a38b8e87673ebe678c7d7f0530e1ab11579cd8d2ee185d"},
		{Hash: "Ed5ecLLSjiXYoC7P4kZj5FtR3YDnEyyLyeXZWFxbC9z4", SignerId: "a5fa6df6a016af406bd62d4d4f1d6de2b6a678603d1f32dc39689d8693301ec4"},
		{Hash: "C3vgRrTEGrB4cRszRhbyWyVfGyqJusH9GMmLeLipT3n6", SignerId: "32321118014239e6400ff018eef8f593aacbe93b7bc6f595ca02d2a791a2f440"},
		{Hash: "8FWJcU2ownJ7VSh61fcTdK6P3PdcDMKR3faEkWgMQCuy", SignerId: "1488.near"},
	}
	assert.Equal(t, txs, expectedTxs)
}
