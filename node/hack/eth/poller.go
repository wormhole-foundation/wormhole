package main

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/rpc"
	ethRpc "github.com/ethereum/go-ethereum/rpc"
	"go.uber.org/zap"
)

type endpointEntry struct {
	name string
	url  string
}

var ciEndpoints = []endpointEntry{
	{name: "EVM", url: "ws://eth-devnet:8545"},
	{name: "EVM2", url: "ws://eth-devnet2:8545"},
}

var localEndpoints = []endpointEntry{
	{name: "EVM", url: "ws://localhost:8545"},
	{name: "EVM2", url: "ws://localhost:8546"},
}

const batchSize = uint64(5)
const batchesPerInterval = 5
const duration = 30 * time.Minute

func main() {
	ctx := context.Background()
	logger, _ := zap.NewDevelopment()

	var endpointStr string

	var endpoints []endpointEntry
	if os.Getenv("INTEGRATION") == "" {
		endpointStr = "local"
		endpoints = localEndpoints
	} else {
		endpointStr = "ci"
		endpoints = ciEndpoints
	}

	logger.Info("starting test", zap.Stringer("duration", duration), zap.String("endpoints", endpointStr))
	timer := time.NewTimer(duration)

	for _, endpoint := range endpoints {
		go poller(ctx, logger.With(zap.String("node", endpoint.name)), endpoint.url)
		time.Sleep(100 * time.Millisecond)
	}

	select {
	case <-ctx.Done():
		logger.Error("context closed!")
	case <-timer.C:
		logger.Info("test complete")
	}
}

func poller(ctx context.Context, logger *zap.Logger, endpoint string) {
	rawClient, err := ethRpc.DialContext(ctx, endpoint)
	if err != nil {
		logger.Fatal("failed to connect to endpoint", zap.String("endpoint", endpoint), zap.Error(err))
	}

	minimumBlockNum := batchesPerInterval * batchSize
	for {
		latestBlockNum, err := getLatestBlockNumber(ctx, rawClient)
		if err != nil {
			logger.Fatal("failed to get initial latest block", zap.Error(err))
		}
		if latestBlockNum > minimumBlockNum {
			break
		}

		logger.Info("waiting for minimum latest block", zap.Uint64("latestBlockNum", latestBlockNum), zap.Uint64("minimumBlockNum", minimumBlockNum))
		time.Sleep(time.Second)
	}

	errorCount := 0
	for {
		latestBlockNum := uint64(0)
		latestBlockNum, err := getLatestBlockNumber(ctx, rawClient)
		if err != nil {
			logger.Fatal("failed to get latest block", zap.Error(err))
		}

		startingBlockNum := latestBlockNum - batchSize*5
		logger.Info("reading batches", zap.Uint64("latestBlockNum", latestBlockNum), zap.Uint64("startingBlockNum", startingBlockNum))
		for count := 0; count < batchesPerInterval; count++ {
			blockNums, err := getBlocks(ctx, rawClient, startingBlockNum, batchSize)
			if err != nil {
				logger.Fatal("failed to get batch of blocks", zap.Uint64("startingBlockNum", startingBlockNum), zap.Uint64("latestBlockNum", latestBlockNum), zap.Error(err))
			}

			if len(blockNums) != int(batchSize) {
				logger.Fatal("getBlocks returned the wrong number of blocks", zap.Uint64("expected", batchSize), zap.Int("actual", len(blockNums)))
			}

			expectedBlockNum := startingBlockNum
			errorFound := false
			for idx, actualBlockNum := range blockNums {
				if expectedBlockNum != actualBlockNum {
					errorFound = true
					logger.Error("getBlocks returned an unexpected block number", zap.Int("idx", idx), zap.Uint64("expectedBlockNum", expectedBlockNum), zap.Uint64("actualBlockNum", actualBlockNum))
				}
				expectedBlockNum++
			}

			if errorFound {
				errorCount++
				if errorCount > 10 {
					logger.Fatal("giving up after too many errors!")
				}
			}

			startingBlockNum += batchSize
		}

		time.Sleep(time.Second)
	}
}

type BlockMarshaller struct {
	Number *hexutil.Big
}

type BatchResult struct {
	result BlockMarshaller
	err    error
}

func getLatestBlockNumber(ctx context.Context, conn *ethRpc.Client) (uint64, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	var m BlockMarshaller
	err := conn.CallContext(timeout, &m, "eth_getBlockByNumber", "latest", false)
	if err != nil {
		return 0, err
	}
	if m.Number == nil {
		return 0, fmt.Errorf("failed to unmarshal block: Number is nil")
	}
	n := big.Int(*m.Number)
	return n.Uint64(), nil
}

func getBlocks(ctx context.Context, conn *ethRpc.Client, startingBlock uint64, numBlocks uint64) ([]uint64, error) {
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	batch := make([]rpc.BatchElem, numBlocks)
	results := make([]BatchResult, numBlocks)
	for idx := 0; idx < int(numBlocks); idx++ {
		batch[idx] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args: []interface{}{
				"0x" + fmt.Sprintf("%x", startingBlock),
				false, // no full transaction details
			},
			Result: &results[idx].result,
			Error:  results[idx].err,
		}
		startingBlock++
	}

	err := conn.BatchCallContext(timeout, batch)
	if err != nil {
		return nil, fmt.Errorf("BatchCallContext failed: %w", err)
	}

	blocks := make([]uint64, 0, numBlocks)
	for idx := range results {
		if results[idx].err != nil {
			return nil, fmt.Errorf("failed to get block idx %d: %w", idx, err)
		}

		m := &results[idx].result
		if m.Number == nil {
			return nil, fmt.Errorf("number in idx %d is nil: %w", idx, err)
		}

		n := big.Int(*m.Number)
		blocks = append(blocks, n.Uint64())
	}

	return blocks, nil
}
