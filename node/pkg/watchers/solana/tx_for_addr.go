package solana

// This code queries for Wormhole observations by reading transactions involving the core contract.
// It takes the start and end blocks to be checked, using one plus the previous current slot as the start,
// and the current slot as the end. It gets the first transaction in the oldest block and the last transaction
// in the newest block and uses the `getSignaturesForAddress` RPC call to query for all transactions in that
// range that involved the Wormhole core contract. It then reads each of those transactions and uses the standard
// transaction processing code to observe any messages found in those transactions.

// TODO: Get rid of "TEST:" log messages.

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/jsonrpc"
	"go.uber.org/zap"
)

const (
	// MaxSignaturesPerQuery is the maximum number of transactions to be returned by `GetSignaturesForAddressWithOpts`.
	// The max / default is 1000. You can set this to something smaller (like 5) to test pagination.
	MaxSignaturesPerQuery = 1000

	// NumGetBlockRetries is how many times we will try to query for a slot, allowing for skipped / missing slots.
	NumGetBlockRetries = 25
)

/* TODO: Delete this code if we end up not needing it.
// transactionProcessor is the entry point of the runnable that periodically queries for new Wormhole observations.
// It uses the standard `DefaultPollDelay`, although the timing will vary based on query delays. Each interval, it gets
// the latest slot. It uses that slot and one plus the latest slot of the previous interval to determine a range of slots.
// It then invokes the function that queries for Wormhole transactions in slots in that range.
func (s *SolanaWatcher) transactionProcessor(ctx context.Context) error {
	timer := time.NewTicker(DefaultPollDelay)
	defer timer.Stop()

	// Keep track of the last slot of the previous interval which determines the oldest slot we need to query next time.
	var oldestSlot uint64

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			if newestSlot, err := s.processLatestTransactions(ctx, oldestSlot); err != nil {
				s.logger.Error("failed to get transactions", zap.Error(err))
				s.errC <- err
				return err
			} else {
				oldestSlot = newestSlot
			}
		}
	}
}

// processLatestTransactions gets the latest slot and then invokes the function that queries for Wormhole transactions in slots in the specified range.
func (s *SolanaWatcher) processLatestTransactions(ctx context.Context, oldestSlot uint64) (uint64, error) {
	rCtx, cancel := context.WithTimeout(ctx, rpcTimeout)
	newestSlot, err := s.rpcClient.GetSlot(rCtx, s.commitment)
	cancel()
	if err != nil {
		return 0, fmt.Errorf("failed to get latest slot: %w", err)
	}

	if oldestSlot == 0 {
		// This is the startup scenario.
		oldestSlot = newestSlot - 1
	} else if oldestSlot == newestSlot {
		s.logger.Debug("not getting transactions, slot number has not advanced", zap.Uint64("slotNum", oldestSlot))
		return oldestSlot, nil
	} else {
		oldestSlot++
	}

	err = s.processTransactionsForSlots(oldestSlot, newestSlot)
	if err != nil {
		return 0, err
	}

	return newestSlot, nil
}
*/

// processTransactionsForSlots queries for the transactions for a range of slots and processes any core events.
func (s *SolanaWatcher) processTransactionsForSlots(oldestSlot uint64, newestSlot uint64) error {
	if newestSlot < oldestSlot {
		// We probably got load balanced. Just wait until next time.
		s.logger.Debug("not getting transactions, slot number went backwards", zap.Uint64("lastSlot", oldestSlot), zap.Uint64("newestSlot", newestSlot))
		return nil
	}

	newestBlock, err := s.findNextValidBlock(newestSlot, true, NumGetBlockRetries)
	if err != nil {
		return fmt.Errorf("failed to get newestBlock: %w", err)
	}

	oldestBlock, err := s.findNextValidBlock(oldestSlot, false, NumGetBlockRetries)
	if err != nil {
		return fmt.Errorf("failed to get oldestBlock: %w", err)
	}

	// We query for transactions in reverse order, so `fromSig`` is newest and `toSig` is oldest.
	fromSig, err := getLastSignature(newestBlock)
	if err != nil {
		return fmt.Errorf("failed to get fromSig: %w", err)
	}
	toSig, err := getFirstSignature(oldestBlock)
	if err != nil {
		return fmt.Errorf("failed to get toSig: %w", err)
	}

	s.logger.Info("TEST: got blocks",
		zap.Uint64("oldestSlot", oldestSlot),
		zap.Uint64("newestSlot", newestSlot),
		zap.Stringer("fromSig", fromSig),
		zap.Stringer("toSig", toSig),
	)

	if err := s.queryAndProcessTransactions(fromSig, toSig); err != nil {
		return fmt.Errorf("failed to query transactions for sigs: %w", err)
	}
	return nil
}

// getFirstSignature returns the first signature in the block.
func getFirstSignature(block *rpc.GetBlockResult) (solana.Signature, error) {
	if len(block.Transactions) == 0 {
		return solana.Signature{}, errors.New("block does not contain any transactions")
	}
	tx, err := block.Transactions[0].GetTransaction()
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to extract tx: %w", err)
	}
	if len(tx.Signatures) == 0 {
		return solana.Signature{}, errors.New("transaction contains no signatures")
	}

	return tx.Signatures[0], nil
}

// getLastSignature returns the last signature in the block.
func getLastSignature(block *rpc.GetBlockResult) (solana.Signature, error) {
	if len(block.Transactions) == 0 {
		return solana.Signature{}, errors.New("block does not contain any transactions")
	}
	tx, err := block.Transactions[len(block.Transactions)-1].GetTransaction()
	if err != nil {
		return solana.Signature{}, fmt.Errorf("failed to extract tx: %w", err)
	}
	if len(tx.Signatures) == 0 {
		return solana.Signature{}, errors.New("transaction contains no signatures")
	}
	return tx.Signatures[len(tx.Signatures)-1], nil
}

// findNextValidBlock queries for a block associated with a slot. If the slot was skipped or is missing,
// it looks for the "next" one, based on the `decrement` flag. If that is true, we decrement the slot number.
// If it is false, we increment it. This process continues until we find a valid block or the number of
// retries is exhausted.
func (s *SolanaWatcher) findNextValidBlock(slot uint64, decrement bool, retries int) (*rpc.GetBlockResult, error) {
	// identify block range by fetching signatures of the first and last transactions
	// getSignaturesForAddress walks backwards so fromSignature occurs after toSignature
	if retries == 0 {
		return nil, errors.New("no block found after exhausting retries")
	}

	// Get the block for the slot, retrying if the block is not yet available (probably because we are behind a proxy).
	var block *rpc.GetBlockResult
	var err error
	for retries := maxRetries; retries > 0; retries -= 1 {
		rewards := false
		maxSupportedTransactionVersion := uint64(0)
		block, err = s.rpcClient.GetBlockWithOpts(s.ctx, uint64(slot), &rpc.GetBlockOpts{
			Encoding:                       solana.EncodingBase64, // solana-go doesn't support json encoding.
			TransactionDetails:             "full",
			Rewards:                        &rewards,
			Commitment:                     s.commitment,
			MaxSupportedTransactionVersion: &maxSupportedTransactionVersion,
		})

		if err == nil || retries <= 1 {
			break
		}

		var rpcErr *jsonrpc.RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32004 /* BLOCK_NOT_AVAILABLE */ {
			break
		}

		time.Sleep(retryDelay)
	}

	if err != nil {
		var rpcErr *jsonrpc.RPCError
		if errors.As(err, &rpcErr) && (rpcErr.Code == -32007 /* SLOT_SKIPPED */ || rpcErr.Code == -32009 /* BLOCK_NOT_AVAILABLE */) {

			// failed to get confirmed block: slot was skipped or missing in long-term storage
			return s.findNextValidBlock(updateSlot(slot, decrement), decrement, retries-1)
		} else {
			return nil, err
		}
	}

	if block == nil || block.BlockTime == nil || len(block.Transactions) == 0 {
		return s.findNextValidBlock(updateSlot(slot, decrement), decrement, retries-1)
	}

	return block, nil
}

// updateSlot updates the slot number, incrementing or decrementing it based on the `decrement` flag.
func updateSlot(slot uint64, decrement bool) uint64 {
	if decrement {
		return slot - 1
	}
	return slot + 1
}

// queryAndProcessTransactions takes a range of signatures and queries for all transactions involving
// the core contract and processes them. Note that query being used goes in reverse, so `fromSig` is after `toSig`.
// For each transaction involving the core, it performs the standard processing to observe a message. It creates
// a separate go routine for each transaction involving the core contract.
func (s *SolanaWatcher) queryAndProcessTransactions(fromSig solana.Signature, toSig solana.Signature) error {
	transactions, err := s.getTransactionSignatures(fromSig, toSig)
	if err != nil {
		return fmt.Errorf("failed to query for transactions: %w", err)
	}

	if len(transactions) == 0 {
		return nil
	}

	s.logger.Info("TEST: found transactions", zap.Int("numItems", len(transactions)), zap.Any("out", transactions))
	for _, tx := range transactions {
		go s.processTransactionWithRetry(tx.Signature)
	}

	return nil
}

// getTransactionSignatures uses the `getSignaturesForAddress` RPC call to query all transactions involving the core contract
// between the two specified signatures (where `fromSig` occurs after `toSig`). Since the API call might not hold all transactions
// (which is very unlikely, since the max is 1000), it handles pagination. After building the set of transactions, it reverses them
// so they are returned in chronological order.
func (s *SolanaWatcher) getTransactionSignatures(fromSig solana.Signature, toSig solana.Signature) ([]*rpc.TransactionSignature, error) {
	results := []*rpc.TransactionSignature{}
	limit := MaxSignaturesPerQuery
	numSignatures := MaxSignaturesPerQuery
	currSig := fromSig

	for numSignatures == MaxSignaturesPerQuery {
		batchSignatures, err := s.rpcClient.GetSignaturesForAddressWithOpts(s.ctx, s.contract, &rpc.GetSignaturesForAddressOpts{
			Before:     currSig,
			Until:      toSig,
			Commitment: s.commitment,
			Limit:      &limit,
		})
		if err != nil {
			return results, fmt.Errorf("GetSignaturesForAddressWithOpts failed: %w", err)
		}
		if len(batchSignatures) == 0 {
			break
		}
		results = append(results, batchSignatures...)

		numSignatures = len(batchSignatures)
		currSig = batchSignatures[len(batchSignatures)-1].Signature
	}

	// Reverse to maintain chronological order.
	slices.Reverse(results)
	return results, nil
}

// processTransactionWithRetry reads a transaction and processes any core observations in it.
// It allows for retries if a "not found" error is encountered. Once the transaction is read,
// it does the standard transaction processing to observe core messages.
func (s *SolanaWatcher) processTransactionWithRetry(signature solana.Signature) {
	for count := range maxRetries {
		if count != 0 {
			time.Sleep(retryDelay)
		}

		rCtx, cancel := context.WithTimeout(s.ctx, rpcTimeout)
		version := uint64(0)
		result, err := s.rpcClient.GetTransaction(
			rCtx,
			signature,
			&rpc.GetTransactionOpts{
				MaxSupportedTransactionVersion: &version,
				Commitment:                     s.commitment,
				Encoding:                       solana.EncodingBase64,
			},
		)
		cancel()
		if err != nil {
			if errors.Is(err, rpc.ErrNotFound) {
				s.logger.Debug("not found", zap.Stringer("sig", signature))
				continue
			}

			s.logger.Error("failed to get transaction for signature", zap.Stringer("signature", signature), zap.Error(err))
			return
		}

		tx, err := result.Transaction.GetTransaction()
		if err != nil {
			s.logger.Error("failed to extract transaction for subscription event", zap.Error(err))
			return
		}

		_ = s.processTransaction(s.ctx, s.rpcClient, tx, result.Meta, result.Slot, false)
		return
	}

	s.logger.Error("failed to query transaction", zap.Stringer("signature", signature))
}
