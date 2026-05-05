package connectors

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"

	ethAbi "github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	ethereum "github.com/ethereum/go-ethereum"
	ethCommon "github.com/ethereum/go-ethereum/common"
	ethTypes "github.com/ethereum/go-ethereum/core/types"
	ethCrypto "github.com/ethereum/go-ethereum/crypto"
	ethEvent "github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/rpc"

	"go.uber.org/zap"
)

// logMessagePublishedTopic is keccak256("LogMessagePublished(address,uint64,uint32,bytes,uint8)").
var logMessagePublishedTopic = ethCrypto.Keccak256Hash([]byte("LogMessagePublished(address,uint64,uint32,bytes,uint8)"))

const (
	// pollMaxErrors is the number of consecutive polling errors before the connector gives up.
	pollMaxErrors = 3

	// pollRPCTimeout is the timeout for batch RPC calls in the poll connector.
	pollRPCTimeout = 15 * time.Second
)

// PollConnector is an HTTP-compatible connector that replaces all WebSocket-based
// subscriptions with timer-driven polling. It handles chains whose eth-compat
// JSON-RPC only exposes HTTP (e.g. Tron).
//
// Unlike BatchPollConnector, it does NOT call SubscribeNewHead (which requires
// eth_subscribe over WebSocket). Instead it polls for latest, finalized, and
// optionally safe blocks entirely via batch eth_getBlockByNumber calls.
//
// It also replaces WatchLogMessagePublished (which calls eth_subscribe "logs")
// with a polling loop that uses eth_getLogs.
type PollConnector struct {
	Connector
	logger       *zap.Logger
	Delay        time.Duration
	batchData    []BatchEntry
	generateSafe bool
}

func NewPollConnector(
	_ context.Context,
	logger *zap.Logger,
	baseConnector Connector,
	safeSupported bool,
	delay time.Duration,
) *PollConnector {
	batchData := []BatchEntry{
		{tag: "finalized", finality: Finalized},
	}
	if safeSupported {
		batchData = append(batchData, BatchEntry{tag: "safe", finality: Safe})
	}
	// Always poll for latest — this is the key difference from BatchPollConnector.
	batchData = append(batchData, BatchEntry{tag: "latest", finality: Latest})

	return &PollConnector{
		Connector:    baseConnector,
		logger:       logger,
		Delay:        delay,
		batchData:    batchData,
		generateSafe: !safeSupported,
	}
}

func (p *PollConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	sub := NewPollSubscription()

	// Get the initial blocks.
	lastBlocks, err := p.getBlocks(ctx, p.logger)
	if err != nil {
		return sub, fmt.Errorf("failed to get initial blocks: %w", err)
	}

	// Publish initial blocks so downstream has a starting point.
	for idx, block := range lastBlocks {
		p.logger.Info(fmt.Sprintf("publishing initial %s block", p.batchData[idx].finality), zap.Uint64("initial_block", block.Number.Uint64()))
		sink <- block
		if p.generateSafe && p.batchData[idx].finality == Finalized {
			safe := block.Copy(Safe)
			p.logger.Info("publishing generated initial safe block", zap.Uint64("initial_block", safe.Number.Uint64()))
			sink <- safe
		}
	}

	errCount := 0

	common.RunWithScissors(ctx, errC, "poller_subscribe_for_blocks", func(ctx context.Context) error {
		timer := time.NewTimer(p.Delay)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-sub.quit:
				sub.unsubDone <- struct{}{}
				return nil
			case <-timer.C:
				lastBlocks, err = p.pollBlocks(ctx, sink, lastBlocks)
				if err != nil {
					errCount++
					p.logger.Error("poll connector encountered an error", zap.Int("errCount", errCount), zap.Error(err))
					if errCount > pollMaxErrors {
						errC <- fmt.Errorf("polling encountered too many errors: %w", err)
						return nil
					}
				} else if errCount != 0 {
					errCount = 0
				}
				timer.Reset(p.Delay)
			}
		}
	})

	return sub, nil
}

// WatchLogMessagePublished polls for LogMessagePublished events via eth_getLogs,
// replacing the WebSocket-based WatchLogMessagePublished on the base connector.
func (p *PollConnector) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	sub := NewPollSubscription()

	// Start from the current finalized block.
	block, err := GetBlockByFinality(ctx, p.Connector, Finalized)
	if err != nil {
		return sub, fmt.Errorf("failed to get initial block for log polling: %w", err)
	}
	fromBlock := block.Number.Uint64()

	common.RunWithScissors(ctx, errC, "poller_watch_log_message_published", func(ctx context.Context) error {
		timer := time.NewTimer(p.Delay)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return nil
			case <-sub.quit:
				sub.unsubDone <- struct{}{}
				return nil
			case <-timer.C:
				// Query from the last seen block to "latest" for the LogMessagePublished topic.
				latest, err := GetBlockByFinality(ctx, p.Connector, Latest)
				if err != nil {
					p.logger.Error("log poller failed to get latest block", zap.Error(err))
					timer.Reset(p.Delay)
					continue
				}

				toBlock := latest.Number.Uint64()
				if toBlock < fromBlock {
					timer.Reset(p.Delay)
					continue
				}

				from := new(big.Int).SetUint64(fromBlock)
				to := new(big.Int).SetUint64(toBlock)
				logs, err := p.Client().FilterLogs(ctx, ethereum.FilterQuery{
					FromBlock: from,
					ToBlock:   to,
					Addresses: []ethCommon.Address{p.ContractAddress()},
					Topics:    [][]ethCommon.Hash{{logMessagePublishedTopic}},
				})
				if err != nil {
					p.logger.Error("log poller failed to get logs", zap.Error(err), zap.Uint64("fromBlock", fromBlock), zap.Uint64("toBlock", toBlock))
					timer.Reset(p.Delay)
					continue
				}

				for _, l := range logs {
					ev, err := p.ParseLogMessagePublished(l)
					if err != nil {
						p.logger.Error("log poller failed to parse log", zap.Error(err))
						continue
					}
					sink <- ev
				}

				// Advance past the range we just queried. Overlap by 1 to not
				// miss logs at block boundaries if a new log appears in the
				// same block after our query, though duplicates are handled
				// downstream.
				fromBlock = toBlock + 1
				timer.Reset(p.Delay)
			}
		}
	})

	return sub, nil
}

func (p *PollConnector) GetLatest(ctx context.Context) (latest, finalized, safe uint64, err error) {
	block, err := GetBlockByFinality(ctx, p.Connector, Latest)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get latest block: %w", err)
	}
	latest = block.Number.Uint64()

	block, err = GetBlockByFinality(ctx, p.Connector, Finalized)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get finalized block: %w", err)
	}
	finalized = block.Number.Uint64()

	if p.generateSafe {
		safe = finalized
	} else {
		block, err = GetBlockByFinality(ctx, p.Connector, Safe)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to get safe block: %w", err)
		}
		safe = block.Number.Uint64()
	}

	return
}

// TimeOfBlockByHash overrides the base connector's implementation because some
// HTTP-only chains (e.g. Tron) return non-standard header fields (such as an
// empty stateRoot "0x") that cause go-ethereum's HeaderByHash to fail. This
// version uses a raw RPC call and only extracts the timestamp.
func (p *PollConnector) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	var m BlockMarshaller
	err := p.RawCallContext(ctx, &m, "eth_getBlockByHash", hash, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get block by hash: %w", err)
	}
	return uint64(m.Time), nil
}

// SubscribeNewHead is not used by PollConnector (we poll instead), but it must
// satisfy the Connector interface. Callers should not rely on it.
func (p *PollConnector) SubscribeNewHead(ctx context.Context, ch chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	return nil, fmt.Errorf("SubscribeNewHead is not supported on HTTP-only connections; use PollConnector.SubscribeForBlocks")
}

// getBlocks mirrors BatchPollConnector.getBlocks using batch RPC.
func (p *PollConnector) getBlocks(ctx context.Context, logger *zap.Logger) (Blocks, error) {
	timeout, cancel := context.WithTimeout(ctx, pollRPCTimeout)
	defer cancel()

	batch := make([]rpc.BatchElem, len(p.batchData))
	results := make([]BatchResult, len(p.batchData))
	for idx, bd := range p.batchData {
		batch[idx] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args: []interface{}{
				bd.tag,
				false,
			},
			Result: &results[idx].result,
			Error:  results[idx].err,
		}
	}

	err := p.Connector.RawBatchCallContext(timeout, batch)
	if err != nil {
		logger.Error("failed to get blocks", zap.Error(err))
		return nil, err
	}

	ret := make(Blocks, len(p.batchData))
	for idx := range results {
		finality := p.batchData[idx].finality
		if results[idx].err != nil {
			logger.Error("failed to get block", zap.Stringer("finality", finality), zap.Error(results[idx].err))
			return nil, results[idx].err
		}

		var n big.Int
		m := &results[idx].result
		if m.Number == nil {
			logger.Debug("number is nil, treating as zero", zap.Stringer("finality", finality), zap.String("tag", p.batchData[idx].tag))
		} else {
			n = big.Int(*m.Number)
		}

		var l1bn *big.Int
		if m.L1BlockNumber != nil {
			bn := big.Int(*m.L1BlockNumber)
			l1bn = &bn
		}

		ret[idx] = &NewBlock{
			Number:        &n,
			Time:          uint64(m.Time),
			Hash:          m.Hash,
			L1BlockNumber: l1bn,
			Finality:      finality,
		}
	}

	return ret, nil
}

// pollBlocks polls for the latest set of blocks, compares to previous, and publishes new ones with gap filling.
func (p *PollConnector) pollBlocks(ctx context.Context, sink chan<- *NewBlock, prevBlocks Blocks) (Blocks, error) {
	newBlocks, err := p.getBlocks(ctx, p.logger)
	if err != nil {
		return prevBlocks, err
	}

	if len(newBlocks) != len(prevBlocks) {
		panic(fmt.Sprintf("getBlocks returned %d entries when there should be %d", len(newBlocks), len(prevBlocks)))
	}

	for idx, newBlock := range newBlocks {
		if newBlock.Number.Cmp(prevBlocks[idx].Number) > 0 {
			newBlockNum := newBlock.Number.Uint64()
			blockNum := prevBlocks[idx].Number.Uint64() + 1
			errorFound := false
			lastPublishedBlock := prevBlocks[idx]
			for blockNum < newBlockNum && !errorFound {
				batchSize := newBlockNum - blockNum
				if batchSize > MaxGapBatchSize {
					batchSize = MaxGapBatchSize
				}
				gapBlocks, err := p.getBlockRange(ctx, p.logger, blockNum, batchSize, p.batchData[idx].finality)
				if err != nil {
					p.logger.Error("failed to get gap blocks", zap.Stringer("finality", p.batchData[idx].finality), zap.Error(err))
					errorFound = true
				} else {
					for _, block := range gapBlocks {
						if block.Number.Uint64() == 0 {
							errorFound = true
							break
						}
						sink <- block
						if p.generateSafe && p.batchData[idx].finality == Finalized {
							sink <- block.Copy(Safe)
						}
						lastPublishedBlock = block
					}
				}
				blockNum += batchSize
			}

			if !errorFound {
				sink <- newBlock
				if p.generateSafe && p.batchData[idx].finality == Finalized {
					sink <- newBlock.Copy(Safe)
				}
			} else {
				newBlocks[idx] = lastPublishedBlock
			}
		} else if newBlock.Number.Cmp(prevBlocks[idx].Number) < 0 {
			p.logger.Debug("block number went backwards, ignoring it", zap.Stringer("finality", p.batchData[idx].finality), zap.Any("new", newBlock.Number), zap.Any("prev", prevBlocks[idx].Number))
			newBlocks[idx] = prevBlocks[idx]
		}
	}

	return newBlocks, nil
}

// getBlockRange gets a range of blocks by number.
func (p *PollConnector) getBlockRange(ctx context.Context, logger *zap.Logger, blockNum uint64, numBlocks uint64, finality FinalityLevel) (Blocks, error) {
	timeout, cancel := context.WithTimeout(ctx, pollRPCTimeout)
	defer cancel()

	batch := make([]rpc.BatchElem, numBlocks)
	results := make([]BatchResult, numBlocks)
	for idx := uint64(0); idx < numBlocks; idx++ {
		batch[idx] = rpc.BatchElem{
			Method: "eth_getBlockByNumber",
			Args: []interface{}{
				"0x" + fmt.Sprintf("%x", blockNum),
				false,
			},
			Result: &results[idx].result,
			Error:  results[idx].err,
		}
		blockNum++
	}

	err := p.Connector.RawBatchCallContext(timeout, batch)
	if err != nil {
		logger.Error("failed to get blocks", zap.Error(err))
		return nil, err
	}

	ret := make(Blocks, numBlocks)
	for idx := range results {
		if results[idx].err != nil {
			logger.Error("failed to get block", zap.Int("idx", idx), zap.Stringer("finality", finality), zap.Error(results[idx].err))
			return nil, results[idx].err
		}

		var n big.Int
		m := &results[idx].result
		if m.Number == nil {
			logger.Debug("number is nil, treating as zero", zap.Stringer("finality", finality))
		} else {
			n = big.Int(*m.Number)
		}

		var l1bn *big.Int
		if m.L1BlockNumber != nil {
			bn := big.Int(*m.L1BlockNumber)
			l1bn = &bn
		}

		ret[idx] = &NewBlock{
			Number:        &n,
			Time:          uint64(m.Time),
			Hash:          m.Hash,
			L1BlockNumber: l1bn,
			Finality:      finality,
		}
	}

	return ret, nil
}
