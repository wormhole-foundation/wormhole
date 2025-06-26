package solana

// This code queries for Wormhole observations by reading transactions involving the core contract.
// It uses the `getSignaturesForAddress` RPC call to query for all transactions since the last observed
// transaction. It then reads each of those transactions and uses the standard transaction processing code
// to observe any messages found in those transactions.
//
// On guardian startup, we read the most recent transaction involving the core contract and store that in
// the Watcher object. This is our starting point for subsequent poll intervals. This is updated each time
// we observe a transaction. By storing it in the Watcher object, we can continue where we left off on a watcher
// restart (but not on a guardian restart).

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap"
)

// MaxSignaturesPerQuery is the maximum number of transactions to be returned by `GetSignaturesForAddressWithOpts`.
// The max / default is 1000. You can set this to something smaller (like 5) to test pagination.
const MaxSignaturesPerQuery = 1000

// transactionProcessor is the entry point of the runnable that periodically queries for new Wormhole observations.
// It uses the standard `DefaultPollDelay`, although the timing will vary based on query delays. Each interval, it
// looks for new transactions involving the core contract by using the `GetSignaturesForAddressWithOpts` RPC call.
// Any transactions that are detected are processed using the standard transaction processing code.
// Note: This is a separate runnable so that query delays don't impact the standard block height reporting.
func (s *SolanaWatcher) transactionProcessor(ctx context.Context) error {
	// Initialize our starting point. If we already have a previous signature, that means there has been a watcher restart
	// (rather than a guardian restart), so we want to preserve that value and start where we left off.
	if s.pollPrevWormholeSignature.IsZero() {
		var err error
		s.pollPrevWormholeSignature, err = s.getPrevWormholeSignature()
		if err != nil {
			s.logger.Error("failed to get the last wormhole signature on start up", zap.Error(err))
			s.errC <- err
			return err
		}
	}

	s.logger.Info("starting from previous wormhole signature", zap.Stringer("prevSig", s.pollPrevWormholeSignature))

	timer := time.NewTicker(DefaultPollDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-timer.C:
			//nolint:contextcheck // Passed via the 's' object instead of as a parameter.
			err := s.processNewTransactions()
			if err != nil {
				s.logger.Error("failed to get transactions", zap.Error(err))
				s.errC <- err
				return err
			}
		}
	}
}

// getPrevWormholeSignature reads the most recent transaction involving the Wormhole core contract and returns it.
func (s *SolanaWatcher) getPrevWormholeSignature() (solana.Signature, error) {
	limit := int(1)
	signatures, err := s.rpcClient.GetSignaturesForAddressWithOpts(s.ctx, s.contract, &rpc.GetSignaturesForAddressOpts{
		Commitment: s.commitment,
		Limit:      &limit,
	})

	if err != nil || len(signatures) == 0 {
		return solana.Signature{}, err
	}

	return signatures[0].Signature, nil
}

// processNewTransactions checks for new transactions involving the core contract and processes them.
func (s *SolanaWatcher) processNewTransactions() error {
	transactions, err := s.getTransactionSignatures()
	if err != nil {
		return fmt.Errorf("failed to query for transactions: %w", err)
	}

	if len(transactions) == 0 {
		return nil
	}

	for _, tx := range transactions {
		// s.logger.Info("TEST: processing transaction", zap.Stringer("sig", tx.Signature))
		go s.processTransactionWithRetry(tx.Signature)
	}

	s.pollPrevWormholeSignature = transactions[len(transactions)-1].Signature
	return nil
}

// getTransactionSignatures uses the `getSignaturesForAddress` RPC call to query for all transactions involving the core contract
// since the last processed transaction. Since the API call might not hold all transactions (which is very unlikely, since the max is 1000),
// it handles pagination. After building the set of transactions, it reverses them so they are returned in chronological order.
func (s *SolanaWatcher) getTransactionSignatures() ([]*rpc.TransactionSignature, error) {
	results := []*rpc.TransactionSignature{}
	limit := MaxSignaturesPerQuery
	numSignatures := MaxSignaturesPerQuery
	currSig := solana.Signature{}

	for numSignatures == MaxSignaturesPerQuery {
		batchSignatures, err := s.rpcClient.GetSignaturesForAddressWithOpts(s.ctx, s.contract, &rpc.GetSignaturesForAddressOpts{
			Before:     currSig,
			Until:      s.pollPrevWormholeSignature,
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
