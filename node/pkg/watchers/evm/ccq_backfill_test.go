package evm

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"

	ethHexUtil "github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.uber.org/zap"
)

// mockBackfillConn simulates a batch query but fails if the batch size is greater than maxBatchSize.
type mockBackfillConn struct {
	maxBatchSize int64
}

func (conn *mockBackfillConn) RawBatchCallContext(ctx context.Context, b []rpc.BatchElem) error {
	if int64(len(b)) > conn.maxBatchSize {
		return fmt.Errorf("batch too large")
	}

	for _, b := range b {
		blockNum, err := strconv.ParseUint(fmt.Sprintf("%v", b.Args[0])[2:], 16, 64)
		if err != nil {
			return fmt.Errorf("invalid hex number: %s", b.Args[0])
		}

		result := ccqBlockMarshaller{Number: ethHexUtil.Uint64(blockNum), Time: ethHexUtil.Uint64(blockNum * 10)}
		bytes, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}

		err = json.Unmarshal(bytes, b.Result)
		if err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}

		b.Error = nil
	}
	return nil
}

// TestCcqBackFillDetermineMaxBatchSize verifies that the search for the max allowed block size converges for all values between 1 and CCQ_MAX_BATCH_SIZE + 1 inclusive.
// It also verifies the returned set of blocks.
func TestCcqBackFillDetermineMaxBatchSize(t *testing.T) {
	ctx := context.Background()
	logger := zap.NewNop()
	latestBlockNum := int64(17533044)

	for maxBatchSize := int64(1); maxBatchSize <= CCQ_MAX_BATCH_SIZE+1; maxBatchSize++ {
		conn := &mockBackfillConn{maxBatchSize: maxBatchSize}
		batchSize, blocks, err := ccqBackFillDetermineMaxBatchSize(ctx, logger, conn, latestBlockNum, time.Microsecond)
		require.NoError(t, err)
		if maxBatchSize > CCQ_MAX_BATCH_SIZE { // If the node supports more than our max size, we should cap the batch size at our max.
			require.Equal(t, CCQ_MAX_BATCH_SIZE, batchSize)
		} else {
			require.Equal(t, maxBatchSize, batchSize)
		}
		require.Equal(t, batchSize, int64(len(blocks)))

		blockNum := uint64(latestBlockNum) // #nosec G115 -- This value is set above so the conversion is safe
		for _, block := range blocks {
			assert.Equal(t, blockNum, block.BlockNum)
			assert.Equal(t, blockNum*10, block.Timestamp)
			blockNum--
		}
	}
}
