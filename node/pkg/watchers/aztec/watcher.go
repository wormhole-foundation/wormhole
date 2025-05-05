package aztec

import (
	"context"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	"go.uber.org/zap"
)

// Watcher handles the processing of blocks and logs
type Watcher struct {
	config             Config
	blockFetcher       BlockFetcher
	l1Verifier         L1Verifier
	observationManager ObservationManager
	msgC               chan<- *common.MessagePublication
	logger             *zap.Logger

	// Simplified tracking - just track the last processed block number
	lastBlockNumber int
}

// NewWatcher creates a new Watcher
func NewWatcher(
	config Config,
	blockFetcher BlockFetcher,
	l1Verifier L1Verifier,
	observationManager ObservationManager,
	msgC chan<- *common.MessagePublication,
	logger *zap.Logger,
) *Watcher {
	return &Watcher{
		config:             config,
		blockFetcher:       blockFetcher,
		l1Verifier:         l1Verifier,
		observationManager: observationManager,
		msgC:               msgC,
		logger:             logger,
		lastBlockNumber:    config.StartBlock - 1, // Start by processing the StartBlock
	}
}

// Run starts the watcher with a single goroutine
func (w *Watcher) Run(ctx context.Context) error {
	w.logger.Info("Starting Aztec watcher",
		zap.String("rpc", w.config.RpcURL),
		zap.String("contract", w.config.ContractAddress))

	// Create an error channel
	errC := make(chan error)
	defer close(errC)

	// Start a single goroutine that handles all operations
	common.RunWithScissors(ctx, errC, "aztec_processor", func(ctx context.Context) error {
		return w.processor(ctx)
	})

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-errC:
		return err
	}
}

func (w *Watcher) processor(ctx context.Context) error {
	ticker := time.NewTicker(w.config.LogProcessingInterval)
	defer ticker.Stop()

	w.logger.Info("Starting Aztec event processor using finalized blocks")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Process blocks
			if err := w.processBlocks(ctx); err != nil {
				w.logger.Error("Error processing blocks", zap.Error(err))
			}
		}
	}
}
