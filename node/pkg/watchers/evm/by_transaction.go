package evm

import (
	"context"
	"fmt"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors"
	"github.com/certusone/wormhole/node/pkg/watchers/evm/connectors/ethabi"

	"github.com/certusone/wormhole/node/pkg/common"
	eth_common "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	gethTypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
)

var (
	// SECURITY: Hardcoded ABI identifier for the LogMessagePublished topic. When using the watcher, we don't need this
	// since the node will only hand us pre-filtered events. In this case, we need to manually verify it
	// since ParseLogMessagePublished will only verify whether it parses.
	LogMessagePublishedTopic = eth_common.HexToHash("0x6eb224fb001ed210e379b335e35efe88672a8ce935d981a6896b27ffdf52a3b2")
)

// isValidCoreBridgeMessagePublicationLog checks that a log entry was emitted by the expected contract,
// has the expected LogMessagePublished event topic, and has not been removed
// due to a chain reorganization. This is called from both the real-time
// subscription path, postMessage, and the reobservation path,
// MessageEventsForTransaction, to ensure consistent validation.
func isValidCoreBridgeMessagePublicationLog(l types.Log, contract eth_common.Address) bool {
	// SECURITY: Reject logs that have been flagged as removed due to a chain reorg.
	if l.Removed {
		return false
	}
	// SECURITY: Verify the log was produced by the supplied contract.
	if l.Address != contract {
		return false
	}
	// SECURITY: Verify the event is LogMessagePublished.
	if len(l.Topics) == 0 || l.Topics[0] != LogMessagePublishedTopic {
		return false
	}
	return true
}

// validateTransactionReceipt checks that a transaction receipt was successfully retrieved and that
// the transaction executed successfully. It returns a non-nil error describing the problem when the
// receipt is unusable. This is the shared validation used by both the real-time subscription path
// (postMessage) and the reobservation path (MessageEventsForTransaction).
//
// SECURITY: Bail early when the receipt status is anything other than 1 (success). In theory this
// check isn't strictly necessary - a failed transaction cannot emit logs and will trigger neither
// subscription messages nor have log messages in its receipt. However, relying on that invariant is
// brittle - we connect to a lot of EVM-compatible chains which might accidentally break this API
// contract and return logs for failed transactions. Check explicitly instead.
func validateTransactionReceipt(receipt *types.Receipt, err error) error {
	if receipt == nil || err != nil {
		return fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	if receipt.Status != gethTypes.ReceiptStatusSuccessful {
		return fmt.Errorf("non-success transaction status: %d", receipt.Status)
	}
	return nil
}

// newMessagePublication builds a MessagePublication from a parsed LogMessagePublished event and
// the timestamp of the block that included it. This is the single place where the EVM watcher
// translates on-chain event data (the ethabi.AbiLogMessagePublished struct returned by the EVM
// libraries) into a MessagePublication, so the real-time subscription path (postMessage) and the
// reobservation path (MessageEventsForTransaction) stay consistent.
//
// isReobservation must be true when the message is being reconstructed from a reobservation
// request and false for live observations; the caller states this explicitly so the field is set
// at construction rather than mutated afterward. The emitter chain id is supplied by the caller
// and is always the watcher's own hardcoded chain id (SECURITY: it must never be derived from
// untrusted event data).
func newMessagePublication(ev *ethabi.AbiLogMessagePublished, blockTime uint64, chainId vaa.ChainID, isReobservation bool) *common.MessagePublication {
	return &common.MessagePublication{
		TxID:             ev.Raw.TxHash.Bytes(),
		Timestamp:        time.Unix(int64(blockTime), 0), // #nosec G115 -- This conversion is safe indefinitely
		Nonce:            ev.Nonce,
		Sequence:         ev.Sequence,
		EmitterChain:     chainId, // SECURITY: Hardcoded chain id from watcher
		EmitterAddress:   PadAddress(ev.Sender),
		Payload:          ev.Payload,
		ConsistencyLevel: ev.ConsistencyLevel,
		IsReobservation:  isReobservation,
		Unreliable:       false,
	}
}

// MessageEventsForTransaction returns the lockup events for a given transaction.
// Returns the block number and a list of MessagePublication events.
//
// isReobservation labels the returned messages: pass true when servicing a reobservation
// request and false for a plain parse (e.g. debug tooling). It is threaded through to
// newMessagePublication so IsReobservation is set at construction rather than mutated later.
func MessageEventsForTransaction(
	ctx context.Context,
	ethConn connectors.Connector,
	contract eth_common.Address,
	chainId vaa.ChainID,
	tx eth_common.Hash,
	isReobservation bool) (*types.Receipt, uint64, []*common.MessagePublication, error) {

	// Get transactions logs from transaction
	// API only returns transactions that have been included in a block. Nothing in the mempool
	receipt, err := ethConn.TransactionReceipt(ctx, tx)
	// SECURITY: Do not trust the logs of a transaction whose receipt could not be fetched or whose
	// execution did not succeed. A failed transaction cannot emit logs, so a non-success receipt
	// here means the RPC node is misbehaving; bail before we parse any events from it. See
	// validateTransactionReceipt for the full rationale.
	if valErr := validateTransactionReceipt(receipt, err); valErr != nil {
		return nil, 0, nil, valErr
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

		if !isValidCoreBridgeMessagePublicationLog(*l, contract) {
			continue
		}

		ev, err := ethConn.ParseLogMessagePublished(*l)
		if err != nil {
			return nil, 0, nil, fmt.Errorf("failed to parse log: %w", err)
		}

		msgs = append(msgs, newMessagePublication(ev, blockTime, chainId, isReobservation))
	}

	return receipt, receipt.BlockNumber.Uint64(), msgs, nil
}
