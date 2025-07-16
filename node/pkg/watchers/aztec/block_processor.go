package aztec

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// processBlocks processes new blocks since the last check
func (w *Watcher) processBlocks(ctx context.Context) error {
	// Get the latest finalized block from L2Tips
	finalizedBlock, err := w.l1Verifier.GetFinalizedBlock(ctx)
	if err != nil {
		w.logger.Error("Failed to fetch latest finalized block", zap.Error(err))
		return err
	}

	// Check if there are new blocks to process
	if w.lastBlockNumber >= finalizedBlock.Number {
		w.logger.Debug("No new finalized blocks to process",
			zap.Int("latest_finalized", finalizedBlock.Number),
			zap.Int("last_processed", w.lastBlockNumber))
		return nil
	}

	w.logger.Info("Processing new finalized blocks",
		zap.Int("from", w.lastBlockNumber+1),
		zap.Int("to", finalizedBlock.Number))

	// Process blocks from last+1 to finalized
	for blockNumber := w.lastBlockNumber + 1; blockNumber <= finalizedBlock.Number; blockNumber++ {
		// Fetch block info
		blockInfo, err := w.blockFetcher.FetchBlock(ctx, blockNumber)
		if err != nil {
			w.logger.Error("Failed to fetch block info",
				zap.Int("blockNumber", blockNumber),
				zap.Error(err))
			return err
		}

		// Process the block's logs
		if err := w.processBlockLogs(ctx, blockNumber, blockInfo); err != nil {
			w.logger.Error("Failed to process block logs",
				zap.Int("blockNumber", blockNumber),
				zap.Error(err))
			return err
		}

		// Update the last processed block number
		w.lastBlockNumber = blockNumber
	}

	return nil
}

// processBlockLogs processes the logs in a single block
func (w *Watcher) processBlockLogs(ctx context.Context, blockNumber int, blockInfo BlockInfo) error {
	logs, err := w.blockFetcher.FetchPublicLogs(ctx, blockNumber, blockNumber+1)
	if err != nil {
		return fmt.Errorf("failed to fetch public logs: %v", err)
	}

	// Only log if there are actually logs to process
	if len(logs) > 0 {
		w.logger.Info("Processing logs",
			zap.Int("count", len(logs)),
			zap.Int("blockNumber", blockNumber))
	}

	// Process each log
	for _, log := range logs {
		// Skip logs that don't match our contract address
		if log.Log.ContractAddress != w.config.ContractAddress {
			w.logger.Debug("Skipping log from different contract",
				zap.String("expected", w.config.ContractAddress),
				zap.String("actual", log.Log.ContractAddress))
			continue
		}

		// Get the correct transaction hash for this log
		txHash := "0x0" // Default if we can't find the right transaction

		// Use the TxIndex from the log's ID to get the right transaction hash
		if txIndex := log.ID.TxIndex; txIndex >= 0 {
			if hash, exists := blockInfo.TxHashesByIndex[txIndex]; exists {
				txHash = hash
			} else {
				w.logger.Error("Transaction index from log not found in block",
					zap.Int("blockNumber", blockNumber),
					zap.Int("txIndex", txIndex))
			}
		}

		// Create a copy of blockInfo with the correct transaction hash for this log
		logBlockInfo := blockInfo
		logBlockInfo.TxHash = txHash

		if err := w.processLog(ctx, log, logBlockInfo); err != nil {
			w.logger.Error("Failed to process log",
				zap.Int("block", log.ID.BlockNumber),
				zap.Int("txIndex", log.ID.TxIndex),
				zap.Int("logIndex", log.ID.LogIndex),
				zap.Error(err))
			// Continue with other logs
			continue
		}
	}

	return nil
}
