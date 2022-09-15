package ethereum

import (
	"context"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/common"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	// SECURITY: Hardcoded ABI identifier for the LogMessagePublished topic. When using the watcher, we don't need this
	// since the node will only hand us pre-filtered events. In this case, we need to manually verify it
	// since ParseLogMessagePublished will only verify whether it parses.
	logMessagePublishedTopic = eth_common.HexToHash("0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2")
)

// MessageEventsForTransaction returns the lockup events for a given transaction.
// Returns the block number and a list of MessagePublication events.
func MessageEventsForTransaction(
	ctx context.Context,
	ethIntf common.Ethish,
	contract eth_common.Address,
	chainId vaa.ChainID,
	tx eth_common.Hash) (uint64, []*common.MessagePublication, error) {

	// Get transactions logs from transaction
	receipt, err := ethIntf.TransactionReceipt(ctx, tx)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}

	// Bail early when the transaction receipt status is anything other than
	// 1 (success). In theory, this check isn't strictly necessary - a failed
	// transaction cannot emit logs and will trigger neither subscription
	// messages nor have log messages in its receipt.
	//
	// However, relying on that invariant is brittle - we connect to a lot of
	// EVM-compatible chains which might accidentally break this API contract
	// and return logs for failed transactions. Check explicitly instead.
	if receipt.Status != 1 {
		return 0, nil, fmt.Errorf("non-success transaction status: %d", receipt.Status)
	}

	// Get block
	blockTime, err := ethIntf.TimeOfBlockByHash(ctx, receipt.BlockHash)
	if err != nil {
		return 0, nil, fmt.Errorf("failed to get block time: %w", err)
	}

	msgs := make([]*common.MessagePublication, 0, len(receipt.Logs))

	// Extract logs
	for _, l := range receipt.Logs {
		if l == nil {
			continue
		}

		// SECURITY: Skip logs not produced by our contract.
		if l.Address != contract {
			continue
		}

		if l.Topics[0] != logMessagePublishedTopic {
			continue
		}

		ev, err := ethIntf.ParseLogMessagePublished(*l)
		if err != nil {
			return 0, nil, fmt.Errorf("failed to parse log: %w", err)
		}

		message := &common.MessagePublication{
			TxHash:           ev.Raw.TxHash,
			Timestamp:        time.Unix(int64(blockTime), 0),
			Nonce:            ev.Nonce,
			Sequence:         ev.Sequence,
			EmitterChain:     chainId,
			EmitterAddress:   PadAddress(ev.Sender),
			Payload:          ev.Payload,
			ConsistencyLevel: ev.ConsistencyLevel,
		}

		msgs = append(msgs, message)
	}

	return receipt.BlockNumber.Uint64(), msgs, nil
}
