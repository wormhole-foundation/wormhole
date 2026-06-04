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

	"go.uber.org/zap"
)

// logMessagePublishedTopic is keccak256("LogMessagePublished(address,uint64,uint32,bytes,uint8)").
var logMessagePublishedTopic = ethCrypto.Keccak256Hash([]byte("LogMessagePublished(address,uint64,uint32,bytes,uint8)"))

// pollMaxErrors is the number of consecutive polling errors before the connector gives up.
const pollMaxErrors = 3

// DefaultMaxLogScanBlocks is the per-iteration eth_getLogs range cap used in
// production. In normal operation the poller should stays near the chain head.
// This is a safety against large gaps that may scan beyond typical provider limits,
// e.g. catching up after a long outage.
const DefaultMaxLogScanBlocks uint64 = 5000

// PollConnector is an HTTP-compatible connector for chains whose eth-compat
// JSON-RPC only exposes HTTP (e.g. Tron). It embeds BatchPollConnector to
// reuse the batch-polling logic for finalized/safe blocks, additionally
// polling for the latest block (since SubscribeNewHead requires WebSocket).
// It also replaces WatchLogMessagePublished (which calls eth_subscribe "logs")
// with a polling loop that uses eth_getLogs.
type PollConnector struct {
	*BatchPollConnector

	// MaxLogScanBlocks caps the per-iteration eth_getLogs range in
	// WatchLogMessagePublished. Zero means no cap. It is set via NewPollConnector
	// (see DefaultMaxLogScanBlocks for the production value); tests scanning
	// historic blocks may pass a smaller value to constrain the scan.
	MaxLogScanBlocks uint64
}

func NewPollConnector(
	_ context.Context,
	logger *zap.Logger,
	baseConnector Connector,
	safeSupported bool,
	delay time.Duration,
	maxLogScanBlocks uint64,
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
		BatchPollConnector: &BatchPollConnector{
			Connector:    baseConnector,
			logger:       logger,
			Delay:        delay,
			batchData:    batchData,
			generateSafe: !safeSupported,
		},
		MaxLogScanBlocks: maxLogScanBlocks,
	}
}

// SubscribeForBlocks overrides the embedded BatchPollConnector implementation
// by skipping the WebSocket-based SubscribeNewHead call. All block types,
// including latest, are obtained via batch polling.
func (p *PollConnector) SubscribeForBlocks(ctx context.Context, errC chan error, sink chan<- *NewBlock) (ethereum.Subscription, error) {
	sub := NewPollSubscription()

	lastBlocks, err := p.getBlocks(ctx, p.logger)
	if err != nil {
		return sub, fmt.Errorf("failed to get initial blocks: %w", err)
	}

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
// replacing the WebSocket-based subscription on the base connector. It begins
// at the current finalized block.
func (p *PollConnector) WatchLogMessagePublished(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished) (ethEvent.Subscription, error) {
	block, err := GetBlockByFinality(ctx, p.Connector, Finalized)
	if err != nil {
		return NewPollSubscription(), fmt.Errorf("failed to get initial block for log polling: %w", err)
	}
	return p.watchLogMessagePublishedFrom(ctx, errC, sink, block.Number.Uint64())
}

// watchLogMessagePublishedFrom is the variant of WatchLogMessagePublished that
// lets the caller specify the starting block instead of using the current
// finalized block. It is unexported because production always enters via
// WatchLogMessagePublished; only same-package tests start from an explicit
// block. The scan respects MaxLogScanBlocks.
func (p *PollConnector) watchLogMessagePublishedFrom(ctx context.Context, errC chan error, sink chan<- *ethAbi.AbiLogMessagePublished, fromBlock uint64) (ethEvent.Subscription, error) {
	sub := NewPollSubscription()

	errCount := 0

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
				latest, err := GetBlockByFinality(ctx, p.Connector, Latest)
				if err != nil {
					errCount++
					p.logger.Error("log poller failed to get latest block", zap.Int("errCount", errCount), zap.Error(err))
					if errCount > pollMaxErrors {
						errC <- fmt.Errorf("log polling encountered too many errors: %w", err)
						return nil
					}
					timer.Reset(p.Delay)
					continue
				}

				// Only scan when there are new blocks; otherwise this is a healthy
				// idle poll and falls through to reset errCount below.
				toBlock := latest.Number.Uint64()
				if toBlock >= fromBlock {
					if p.MaxLogScanBlocks > 0 && toBlock-fromBlock+1 > p.MaxLogScanBlocks {
						toBlock = fromBlock + p.MaxLogScanBlocks - 1
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
						errCount++
						p.logger.Error("log poller failed to get logs", zap.Int("errCount", errCount), zap.Error(err), zap.Uint64("fromBlock", fromBlock), zap.Uint64("toBlock", toBlock))
						if errCount > pollMaxErrors {
							errC <- fmt.Errorf("log polling encountered too many errors: %w", err)
							return nil
						}
						timer.Reset(p.Delay)
						continue
					}

					for _, l := range logs {
						ev, err := p.ParseLogMessagePublished(l)
						if err != nil {
							// A single malformed log is not a connectivity failure, so
							// skip it without counting toward the error threshold.
							p.logger.Error("log poller failed to parse log", zap.Error(err))
							continue
						}
						sink <- ev
					}

					fromBlock = toBlock + 1
				}

				errCount = 0
				timer.Reset(p.Delay)
			}
		}
	})

	return sub, nil
}

// TimeOfBlockByHash overrides the base connector's implementation because some
// HTTP-only chains (e.g. Tron) return non-standard header fields (such as an
// empty stateRoot "0x") that cause go-ethereum's HeaderByHash to fail. This
// version uses a raw RPC call and only extracts the timestamp.
func (p *PollConnector) TimeOfBlockByHash(ctx context.Context, hash ethCommon.Hash) (uint64, error) {
	var m *BlockMarshaller
	err := p.RawCallContext(ctx, &m, "eth_getBlockByHash", hash, false)
	if err != nil {
		return 0, fmt.Errorf("failed to get block by hash: %w", err)
	}
	// eth_getBlockByHash returns a JSON null result for an unknown block, which
	// unmarshals into a nil pointer without an error. Mirror go-ethereum's
	// ethclient.HeaderByHash and surface this as ethereum.NotFound so callers
	// don't mistake a missing block for one with a zero timestamp.
	if m == nil {
		return 0, ethereum.NotFound
	}
	return uint64(m.Time), nil
}

// SubscribeNewHead overrides the base connector to make WebSocket subscriptions
// fail loudly. PollConnector polls for latest blocks via SubscribeForBlocks.
func (p *PollConnector) SubscribeNewHead(ctx context.Context, ch chan<- *ethTypes.Header) (ethereum.Subscription, error) {
	return nil, fmt.Errorf("SubscribeNewHead is not supported on HTTP-only connections; use PollConnector.SubscribeForBlocks")
}
