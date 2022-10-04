package near

import (
	"context"
	"testing"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
	. "github.com/certusone/wormhole/node/pkg/watchers/near/nearapi"
	"github.com/certusone/wormhole/node/pkg/watchers/near/timerqueue"
	"github.com/test-go/testify/assert"
)

type (
	MockNearAPI struct{}
)

func (n MockNearAPI) GetBlock(ctx context.Context, blockId string) (Block, error) {
	if blockId == "NSM5RDZDF7uxGWiUwhBqJcqCEw6g7axx4TxGYB7XZVt"
}
func (n MockNearAPI) GetBlockByHeight(ctx context.Context, blockHeight uint64) (Block, error) {

}
func (n MockNearAPI) GetFinalBlock(ctx context.Context) (Block, error) {

}
func (n MockNearAPI) GetChunk(ctx context.Context, chunkHeader ChunkHeader) (Chunk, error) {

}
func (n MockNearAPI) GetTxStatus(ctx context.Context, txHash string, senderAccountId string) ([]byte, error) {

}

func TestTxProcessing(t *testing.T) {
	msgC = make(chan *common.MessagePublication)
	nearTestWatcher = Watcher{
		mainnet:                       false,
		wormholeAccount:               "wormhole.test.near",
		nearRPC:                       "",
		msgC:                          msgC,
		obsvReqC:                      obsvReqC,
		transactionProcessingQueue:    *timerqueue.New(),
		chunkProcessingQueue:          make(chan nearapi.ChunkHeader, quequeSize),
		eventChanBlockProcessedHeight: make(chan uint64, 10),
		eventChanTxProcessedDuration:  make(chan time.Duration, 10),
		eventChan:                     make(chan eventType, 10),
	}
}

func TestSuccessValueToInt(t *testing.T) {

	type test struct {
		input  string
		output int
	}

	testsPositive := []test{
		{"MjU=", 25},
		{"MjQ4", 248},
	}

	testsNegative := []test{
		{"", 0},
		{"?", 0},
		{"MjQ4=", 0},
		{"eAo=", 0},
		{"Cg==", 0},
	}

	for _, tc := range testsPositive {
		t.Run(tc.input, func(t *testing.T) {
			i, err := successValueToInt(tc.input)
			assert.Equal(t, tc.output, i)
			assert.NoError(t, err)
		})
	}

	for _, tc := range testsNegative {
		t.Run(tc.input, func(t *testing.T) {
			i, err := successValueToInt(tc.input)
			assert.Equal(t, tc.output, i)
			assert.NotNil(t, err)
		})
	}
}
