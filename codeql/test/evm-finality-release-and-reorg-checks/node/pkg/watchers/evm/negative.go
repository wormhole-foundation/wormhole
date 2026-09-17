package evm

import (
	"context"
	"errors"

	"codeql/evmfinalityrelease/ethereum"
	"codeql/evmfinalityrelease/gethtypes"
	"codeql/evmfinalityrelease/rpc"
)

func negativeCanonicalPendingRelease(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockNumberU := ev.Number
	thisConsistencyLevel := ConsistencyLevelFinalized
	if ev.Finality == Safe {
		thisConsistencyLevel = ConsistencyLevelSafe
	}

	for key, pLock := range w.pending {
		if !consistencyLevelMatches(thisConsistencyLevel, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > blockNumberU {
			continue
		}
		txreceipt, err := w.ethConn.TransactionReceipt(ctx, gethtypes.BytesToHash(pLock.message.TxID))
		if errors.Is(err, rpc.ErrNoResult) || errors.Is(err, ethereum.NotFound) {
			delete(w.pending, key)
			continue
		}
		if err != nil {
			continue
		}
		if txreceipt == nil {
			delete(w.pending, key)
			continue
		}
		expectedTxHash := gethtypes.BytesToHash(pLock.message.TxID)
		if txreceipt.TxHash != expectedTxHash {
			delete(w.pending, key)
			continue
		}
		if txreceipt.BlockHash != key.BlockHash {
			delete(w.pending, key)
			continue
		}
		txHash := gethtypes.Hash(pLock.message.TxID)
		if err := w.verifyAndPublish(pLock.message, ctx, txHash, txreceipt); err != nil {
			return err
		}
	}
	return nil
}

func negativeReceiptSuccessDelegated(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockNumberU := ev.Number
	thisConsistencyLevel := ConsistencyLevelFinalized
	if ev.Finality == Safe {
		thisConsistencyLevel = ConsistencyLevelSafe
	}
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(thisConsistencyLevel, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > blockNumberU {
			continue
		}
		txreceipt, err := w.ethConn.TransactionReceipt(ctx, gethtypes.BytesToHash(pLock.message.TxID))
		if errors.Is(err, rpc.ErrNoResult) || errors.Is(err, ethereum.NotFound) {
			delete(w.pending, key)
			continue
		}
		if err != nil {
			continue
		}
		if txreceipt == nil {
			delete(w.pending, key)
			continue
		}
		if txreceipt.TxHash != gethtypes.BytesToHash(pLock.message.TxID) {
			delete(w.pending, key)
			continue
		}
		if txreceipt.BlockHash != key.BlockHash {
			delete(w.pending, key)
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func negativeInstantPublicationExcluded(w *Watcher, ctx context.Context, msgTx gethtypes.Hash) error {
	msg := &struct{ TxID []byte }{}
	receipt, err := w.ethConn.TransactionReceipt(ctx, msgTx)
	if err != nil || receipt == nil {
		return err
	}
	_ = msg
	return nil
}

func negativeRenamedAliasesAndSafeHeight(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockNumberU := ev.Number
	thisConsistencyLevel := ConsistencyLevelFinalized
	if ev.Finality == Safe {
		thisConsistencyLevel = ConsistencyLevelSafe
	}
	for pendingKeyAlias, pendingAlias := range w.pending {
		if !consistencyLevelMatches(thisConsistencyLevel, pendingAlias.effectiveCL) {
			continue
		}
		if blockNumberU < pendingAlias.height+pendingAlias.additionalBlocks {
			continue
		}
		gotReceipt, fetchProblem := w.ethConn.TransactionReceipt(ctx, gethtypes.BytesToHash(pendingAlias.message.TxID))
		if errors.Is(fetchProblem, rpc.ErrNoResult) || errors.Is(fetchProblem, ethereum.NotFound) {
			delete(w.pending, pendingKeyAlias)
			continue
		}
		if fetchProblem != nil {
			continue
		}
		if gotReceipt == nil {
			delete(w.pending, pendingKeyAlias)
			continue
		}
		wantHash := gethtypes.BytesToHash(pendingAlias.message.TxID)
		if gotReceipt.TxHash != wantHash {
			delete(w.pending, pendingKeyAlias)
			continue
		}
		if gotReceipt.BlockHash != pendingKeyAlias.BlockHash {
			delete(w.pending, pendingKeyAlias)
			continue
		}
		return w.verifyAndPublish(pendingAlias.message, ctx, gethtypes.Hash(pendingAlias.message.TxID), gotReceipt)
	}
	return nil
}
