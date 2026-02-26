package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"

	"github.com/certusone/wormhole/node/pkg/common"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	// SECURITY: Hardcoded ABI identifier for the LogMessagePublished topic. When using the watcher, we don't need this
	// since the node will only hand us pre-filtered events. In this case, we need to manually verify it
	// since ParseLogMessagePublished will only verify whether it parses.
	LogMessagePublishedTopic = eth_common.HexToHash("0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2")
)

// isLogValid checks that a log entry was emitted by the expected contract,
// has the expected LogMessagePublished event topic, and has not been removed
// due to a chain reorganization. This is called from both the real-time
// subscription path (postMessage) and the reobservation path
// (MessageEventsForTransaction) to ensure consistent validation.
func isLogValid(l types.Log, contract eth_common.Address) bool {
	// SECURITY: Reject logs that have been flagged as removed due to a chain reorg.
	if l.Removed {
		return false
	}
	// SECURITY: Verify the log was produced by our contract.
	if l.Address != contract {
		return false
	}
	// SECURITY: Verify the event is LogMessagePublished.
	if len(l.Topics) == 0 || l.Topics[0] != LogMessagePublishedTopic {
		return false
	}
	return true
}

// MessageEventsForTransaction returns the lockup events for a given transaction.
// Returns the block number and a list of MessagePublication events.
func MessageEventsForTransaction(
	ctx context.Context,
	ethConn connectors.Connector,
	contract eth_common.Address,
	chainId vaa.ChainID,
	tx eth_common.Hash) (*types.Receipt, uint64, []*common.MessagePublication, error) {

	// Get transactions logs from transaction
	receipt, err := ethConn.TransactionReceipt(ctx, tx)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// SECURITY
	// Bail early when the transaction receipt status is anything other than
	// 1 (success). In theory, this check isn't strictly necessary - a failed
	// transaction cannot emit logs and will trigger neither subscription
	// messages nor have log messages in its receipt.
	//
	// However, relying on that invariant is brittle - we connect to a lot of
	// EVM-compatible chains which might accidentally break this API contract
	// and return logs for failed transactions. Check explicitly instead.
	if receipt.Status != 1 {
		return nil, 0, nil, fmt.Errorf("non-success transaction status: %d", receipt.Status)
	}

	// Get block
	blockTime, err := ethConn.TimeOfBlockByHash(ctx, receipt.BlockHash)
	if err != nil {
		return nil, 0, nil, fmt.Errorf("failed to get block time: %w", err)
	}

	msgs := make([]*common.MessagePublication, 0, len(receipt.Logs))

	// Extract logs
	for _, l := range receipt.Logs {
		if l == nil {
			continue
		}

		if !isLogValid(*l, contract) {
			continue
		}

		ev, err := ethConn.ParseLogMessagePublished(*l)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to parse log: %w", err)
		}

		message := &common.MessagePublication{
			TxID:             ev.Raw.TxHash.Bytes(),
			Timestamp:        time.Unix(int64(blockTime), 0), // #nosec G115 -- This conversion is safe indefinitely
			Nonce:            ev.Nonce,
			Sequence:         ev.Sequence,
			EmitterChain:     chainId, // SECURITY: Hardcoded to watcher chain id
			EmitterAddress:   PadAddress(ev.Sender),
			Payload:          ev.Payload,
			ConsistencyLevel: ev.ConsistencyLevel,
		}

		msgs = append(msgs, message)
	}

	return receipt, receipt.BlockNumber.Uint64(), msgs, nil
}
