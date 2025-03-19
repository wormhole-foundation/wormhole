package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// handleReobservationRequest performs a reobservation request and publishes any observed transactions.
func (w *Watcher) handleReobservationRequest(ctx context.Context, chainId vaa.ChainID, txID []byte, ethConn connectors.Connector, finalizedBlockNum, safeBlockNum uint64) (numObservations uint32, err error) {
	// This can't happen unless there is a programming error - the caller
	// is expected to send us only requests for our chainID.
	if chainId != w.chainID {
		return 0, fmt.Errorf("unexpected chain id: %v", chainId)
	}

	tx := eth_common.BytesToHash(txID)
	w.logger.Info("received observation request", zap.String("tx_hash", tx.Hex()))

	// SECURITY: We loaded the block number before requesting the transaction to avoid a
	// race condition where requesting the tx succeeds and is then dropped due to a fork,
	// but finalizedBlock had already advanced beyond the required threshold.
	//
	// In the primary watcher flow, this is of no concern since we assume the node
	// always sends the head before it sends the logs (implicit synchronization
	// by relying on the same websocket connection).

	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	receipt, blockNumber, msgs, err := MessageEventsForTransaction(timeout, ethConn, w.contract, w.chainID, tx)
	cancel()

	if err != nil {
		return 0, fmt.Errorf("failed to process observation request: %v", err)
	}

	for _, msg := range msgs {
		msg.IsReobservation = true
		if msg.ConsistencyLevel == vaa.ConsistencyLevelPublishImmediately {
			w.logger.Info("re-observed message publication transaction, publishing it immediately",
				zap.String("msgId", msg.MessageIDString()),
				zap.String("txHash", msg.TxIDString()),
				zap.Uint64("current_block", finalizedBlockNum),
				zap.Uint64("observed_block", blockNumber),
			)

			pubErr := w.verifyAndPublish(msg, ctx, eth_common.BytesToHash(msg.TxID), receipt)

			if pubErr != nil {
				w.logger.Error("Error when publishing message", zap.Error(err))
			} else {
				numObservations++
			}

			continue
		}

		if msg.ConsistencyLevel == vaa.ConsistencyLevelSafe {
			if safeBlockNum == 0 {
				w.logger.Error("no safe block number available, ignoring observation request",
					zap.String("msgId", msg.MessageIDString()),
					zap.String("txHash", msg.TxIDString()),
				)
				continue
			}

			if blockNumber <= safeBlockNum {
				w.logger.Info("re-observed message publication transaction",
					zap.String("msgId", msg.MessageIDString()),
					zap.String("txHash", msg.TxIDString()),
					zap.Uint64("current_safe_block", safeBlockNum),
					zap.Uint64("observed_block", blockNumber),
				)

				pubErr := w.verifyAndPublish(msg, ctx, eth_common.BytesToHash(msg.TxID), receipt)

				if pubErr != nil {
					w.logger.Error("Error when publishing message", zap.Error(err))
					// Avoid increasing the observations metrics for messages that weren't published.
					continue
				}

				numObservations++
			} else {
				w.logger.Info("ignoring re-observed message publication transaction",
					zap.String("msgId", msg.MessageIDString()),
					zap.String("txHash", msg.TxIDString()),
					zap.Uint64("current_safe_block", safeBlockNum),
					zap.Uint64("observed_block", blockNumber),
				)
			}

			continue
		}

		if finalizedBlockNum == 0 {
			w.logger.Error("no block number available, ignoring observation request",
				zap.String("msgId", msg.MessageIDString()),
				zap.String("txHash", msg.TxIDString()),
			)
			continue
		}

		// SECURITY: In the recovery flow, we already know which transaction to
		// observe, and we can assume that it has reached the expected finality
		// level a long time ago. Therefore, the logic is much simpler than the
		// primary watcher, which has to wait for finality.
		//
		// Instead, we can simply check if the transaction's block number is in
		// the past by more than the expected confirmation number.
		//
		// Ensure that the current block number is larger than the message observation's block number.
		if blockNumber <= finalizedBlockNum {
			w.logger.Info("re-observed message publication transaction",
				zap.String("msgId", msg.MessageIDString()),
				zap.String("txHash", msg.TxIDString()),
				zap.Uint64("current_block", finalizedBlockNum),
				zap.Uint64("observed_block", blockNumber),
			)

			pubErr := w.verifyAndPublish(msg, ctx, eth_common.BytesToHash(msg.TxID), receipt)

			if pubErr != nil {
				w.logger.Error("Error when publishing message", zap.Error(err))
			} else {
				numObservations++
			}
		} else {
			w.logger.Info("ignoring re-observed message publication transaction",
				zap.String("msgId", msg.MessageIDString()),
				zap.String("txHash", msg.TxIDString()),
				zap.Uint64("current_block", finalizedBlockNum),
				zap.Uint64("observed_block", blockNumber),
			)
		}
	}
	return
}

// Reobserve is the interface for reobserving using a custom URL. It opens a connection to that URL and does the reobservation on it.
func (w *Watcher) Reobserve(ctx context.Context, chainID vaa.ChainID, txID []byte, customEndpoint string) (uint32, error) {
	w.logger.Info("received a request to reobserve using a custom endpoint", zap.Stringer("chainID", chainID), zap.Any("txID", txID), zap.String("url", customEndpoint))

	// Verify that this endpoint is for the correct chain.
	if err := w.verifyEvmChainID(ctx, w.logger, customEndpoint); err != nil {
		return 0, fmt.Errorf("failed to verify evm chain id: %w", err)
	}

	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	// Connect to the node using the appropriate type of connector and the custom endpoint.
	ethConn, _, _, err := w.createConnector(timeout, customEndpoint)
	if err != nil {
		return 0, fmt.Errorf(`failed to connect to endpoint "%v": %w`, customEndpoint, err)
	}

	// Get the current finalized and safe blocks.
	_, finalized, safe, err := ethConn.GetLatest(timeout)
	if err != nil {
		return 0, fmt.Errorf(`failed to get latest blocks: %w`, err)
	}

	// Finally, do the reobservation and return the number of messages observed.
	return w.handleReobservationRequest(ctx, chainID, txID, ethConn, finalized, safe)
}
