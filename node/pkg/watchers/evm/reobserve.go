package evm

import (
	"context"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"
)

// handleReobservationRequest performs a reobservation request and publishes any observed transactions.
func (w *Watcher) handleReobservationRequest(ctx context.Context, chainId vaa.ChainID, txID []byte, ethConn connectors.Connector) (numObservations uint32, err error) {
	// This can't happen unless there is a programming error - the caller
	// is expected to send us only requests for our chainID.
	if chainId != w.chainID {
		return 0, fmt.Errorf("unexpected chain id: %v", chainId)
	}

	tx := eth_common.BytesToHash(txID)
	w.logger.Info("received observation request", zap.String("tx_hash", tx.Hex()))

	// SECURITY: Load the block number before requesting the transaction to avoid a
	// race condition where requesting the tx succeeds and is then dropped due to a fork,
	// but blockNumberU had already advanced beyond the required threshold.
	//
	// In the primary watcher flow, this is of no concern since we assume the node
	// always sends the head before it sends the logs (implicit synchronization
	// by relying on the same websocket connection).
	blockNumberU := atomic.LoadUint64(&w.latestFinalizedBlockNumber)
	safeBlockNumberU := atomic.LoadUint64(&w.latestSafeBlockNumber)

	timeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	blockNumber, msgs, err := MessageEventsForTransaction(timeout, ethConn, w.contract, w.chainID, tx)
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
				zap.Uint64("current_block", blockNumberU),
				zap.Uint64("observed_block", blockNumber),
			)
			w.msgC <- msg
			numObservations++
			continue
		}

		if msg.ConsistencyLevel == vaa.ConsistencyLevelSafe {
			if safeBlockNumberU == 0 {
				w.logger.Error("no safe block number available, ignoring observation request",
					zap.String("msgId", msg.MessageIDString()),
					zap.String("txHash", msg.TxIDString()),
				)
				continue
			}

			if blockNumber <= safeBlockNumberU {
				w.logger.Info("re-observed message publication transaction",
					zap.String("msgId", msg.MessageIDString()),
					zap.String("txHash", msg.TxIDString()),
					zap.Uint64("current_safe_block", safeBlockNumberU),
					zap.Uint64("observed_block", blockNumber),
				)
				w.msgC <- msg
				numObservations++
			} else {
				w.logger.Info("ignoring re-observed message publication transaction",
					zap.String("msgId", msg.MessageIDString()),
					zap.String("txHash", msg.TxIDString()),
					zap.Uint64("current_safe_block", safeBlockNumberU),
					zap.Uint64("observed_block", blockNumber),
				)
			}

			continue
		}

		if blockNumberU == 0 {
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
		if blockNumber <= blockNumberU {
			w.logger.Info("re-observed message publication transaction",
				zap.String("msgId", msg.MessageIDString()),
				zap.String("txHash", msg.TxIDString()),
				zap.Uint64("current_block", blockNumberU),
				zap.Uint64("observed_block", blockNumber),
			)
			w.msgC <- msg
			numObservations++
		} else {
			w.logger.Info("ignoring re-observed message publication transaction",
				zap.String("msgId", msg.MessageIDString()),
				zap.String("txHash", msg.TxIDString()),
				zap.Uint64("current_block", blockNumberU),
				zap.Uint64("observed_block", blockNumber),
			)
		}
	}
	return
}

// Reobserve is the interface for reobserving using a custom URL. It opens a connection to that URL and does the reobservation on it.
func (w *Watcher) Reobserve(ctx context.Context, chainID vaa.ChainID, txID []byte, customEndpoint string) (uint32, error) {
	w.logger.Info("received a request to reobserve using a custom endpoint", zap.Stringer("chainID", chainID), zap.Any("txID", txID), zap.String("url", customEndpoint))
	timeout, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	ethConn, err := connectors.NewEthereumBaseConnector(timeout, w.networkName, w.url, w.contract, w.logger)
	if err != nil {
		return 0, fmt.Errorf(`failed to connect to endpoint "%v": %w`, customEndpoint, err)
	}
	return w.handleReobservationRequest(ctx, chainID, txID, ethConn)
}
