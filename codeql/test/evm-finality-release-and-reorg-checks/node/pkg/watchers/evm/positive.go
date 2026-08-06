package evm

import (
	"context"
	"errors"

	"codeql/evmfinalityrelease/ethereum"
	"codeql/evmfinalityrelease/gethtypes"
	"codeql/evmfinalityrelease/rpc"
)

func positiveMissingConsistencyGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	for key, pLock := range w.pending {
		if pLock.height+pLock.additionalBlocks > ev.Number {
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

func positiveWrongConsistencySource(w *Watcher, ctx context.Context, ev NewBlock) error {
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(ConsistencyLevelFinalized, pLock.message.ConsistencyLevel) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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

func positiveMissingHeightGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
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

func positiveNoReceiptRefetch(w *Watcher, ctx context.Context, cached *gethtypes.Receipt) error {
	blockCL := ConsistencyLevelFinalized
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > 100 {
			continue
		}
		if cached == nil {
			delete(w.pending, key)
			continue
		}
		if cached.TxHash != gethtypes.BytesToHash(pLock.message.TxID) {
			delete(w.pending, key)
			continue
		}
		if cached.BlockHash != key.BlockHash {
			delete(w.pending, key)
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), cached)
	}
	return nil
}

func positiveMissingNotFoundRejection(w *Watcher, ctx context.Context) error {
	blockCL := ConsistencyLevelFinalized
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > 100 {
			continue
		}
		txreceipt, err := w.ethConn.TransactionReceipt(ctx, gethtypes.BytesToHash(pLock.message.TxID))
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

func positiveMissingTxHashMatch(w *Watcher, ctx context.Context) error {
	blockCL := ConsistencyLevelFinalized
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > 100 {
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
		if txreceipt.BlockHash != key.BlockHash {
			delete(w.pending, key)
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func positiveMissingBlockHashMatch(w *Watcher, ctx context.Context) error {
	blockCL := ConsistencyLevelFinalized
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > 100 {
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
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func positiveWrongBlockHashSource(w *Watcher, ctx context.Context, ev NewBlock) error {
	currentBlockHash := gethtypes.Hash{}
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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
		if txreceipt.BlockHash != currentBlockHash {
			delete(w.pending, key)
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func positiveWrongErrorChecked(w *Watcher, ctx context.Context, ev NewBlock, unrelated error) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
			continue
		}
		txreceipt, receiptErr := w.ethConn.TransactionReceipt(ctx, gethtypes.BytesToHash(pLock.message.TxID))
		if errors.Is(receiptErr, rpc.ErrNoResult) || errors.Is(receiptErr, ethereum.NotFound) {
			delete(w.pending, key)
			continue
		}
		if unrelated != nil {
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

func positiveWrongReceiptChecked(w *Watcher, ctx context.Context, ev NewBlock, otherReceipt *gethtypes.Receipt) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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
		if otherReceipt == nil {
			delete(w.pending, key)
			continue
		}
		if otherReceipt.TxHash != gethtypes.BytesToHash(pLock.message.TxID) {
			delete(w.pending, key)
			continue
		}
		if otherReceipt.BlockHash != key.BlockHash {
			delete(w.pending, key)
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func positivePostChecksDoNotDominate(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
			continue
		}
		txreceipt, err := w.ethConn.TransactionReceipt(ctx, gethtypes.BytesToHash(pLock.message.TxID))
		if errors.Is(err, rpc.ErrNoResult) || errors.Is(err, ethereum.NotFound) {
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
		publishErr := w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
		if err != nil {
			continue
		}
		if txreceipt == nil {
			delete(w.pending, key)
			continue
		}
		return publishErr
	}
	return nil
}

func positiveWrongPendingKey(w *Watcher, ctx context.Context, ev NewBlock, wrongKey pendingKey) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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
		if txreceipt.BlockHash != wrongKey.BlockHash {
			delete(w.pending, key)
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func positiveWrongPendingMessageFetch(w *Watcher, ctx context.Context, ev NewBlock, other *pendingMessage) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
			continue
		}
		txreceipt, err := w.ethConn.TransactionReceipt(ctx, gethtypes.BytesToHash(other.message.TxID))
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

func positiveInvertedConsistencyGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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

func positiveLogOnlyConsistencyGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			delete(w.pending, key)
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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

func positiveInvertedHeightGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks <= ev.Number {
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

func positiveLogOnlyHeightGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
			delete(w.pending, key)
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

func positiveInvertedTxHashGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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
		if txreceipt.TxHash == gethtypes.BytesToHash(pLock.message.TxID) {
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

func positiveLogOnlyTxHashGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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
		}
		if txreceipt.BlockHash != key.BlockHash {
			delete(w.pending, key)
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func positiveInvertedBlockHashGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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
		if txreceipt.BlockHash == key.BlockHash {
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func positiveLogOnlyBlockHashGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockCL := currentBlockConsistencyLevel(ev)
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(blockCL, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > ev.Number {
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
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func positiveNestedConditionalConsistencyExit(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockNumberU := ev.Number
	thisConsistencyLevel := ConsistencyLevelFinalized
	if ev.Finality == Safe {
		thisConsistencyLevel = ConsistencyLevelSafe
	}
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(thisConsistencyLevel, pLock.effectiveCL) {
			if blockNumberU > 0 {
				continue
			}
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

func positiveStaleConsistencyLevel(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockNumberU := ev.Number
	thisConsistencyLevel := ConsistencyLevelFinalized
	staleConsistencyLevel := thisConsistencyLevel
	if ev.Finality == Safe {
		thisConsistencyLevel = ConsistencyLevelSafe
	}
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(staleConsistencyLevel, pLock.effectiveCL) {
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

func positiveWrongBlockNumber(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockNumberU := ev.Number
	wrongBlockNumberU := blockNumberU + 1
	thisConsistencyLevel := ConsistencyLevelFinalized
	if ev.Finality == Safe {
		thisConsistencyLevel = ConsistencyLevelSafe
	}
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(thisConsistencyLevel, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > wrongBlockNumberU {
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

func positiveConstantBlockNumber(w *Watcher, ctx context.Context, ev NewBlock) error {
	_ = ev
	thisConsistencyLevel := ConsistencyLevelFinalized
	for key, pLock := range w.pending {
		if !consistencyLevelMatches(thisConsistencyLevel, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > 100 {
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

func positiveNonWPendingRange(w *Watcher, other *Watcher, ctx context.Context, ev NewBlock) error {
	blockNumberU := ev.Number
	thisConsistencyLevel := ConsistencyLevelFinalized
	if ev.Finality == Safe {
		thisConsistencyLevel = ConsistencyLevelSafe
	}
	for key, pLock := range other.pending {
		if !consistencyLevelMatches(thisConsistencyLevel, pLock.effectiveCL) {
			continue
		}
		if pLock.height+pLock.additionalBlocks > blockNumberU {
			continue
		}
		txreceipt, err := w.ethConn.TransactionReceipt(ctx, gethtypes.BytesToHash(pLock.message.TxID))
		if errors.Is(err, rpc.ErrNoResult) || errors.Is(err, ethereum.NotFound) {
			delete(other.pending, key)
			continue
		}
		if err != nil {
			continue
		}
		if txreceipt == nil {
			delete(other.pending, key)
			continue
		}
		if txreceipt.TxHash != gethtypes.BytesToHash(pLock.message.TxID) {
			delete(other.pending, key)
			continue
		}
		if txreceipt.BlockHash != key.BlockHash {
			delete(other.pending, key)
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}

func positiveUnrelatedNestedConsistencyGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
	blockNumberU := ev.Number
	thisConsistencyLevel := ConsistencyLevelFinalized
	if ev.Finality == Safe {
		thisConsistencyLevel = ConsistencyLevelSafe
	}
	for key, pLock := range w.pending {
		if blockNumberU > 0 {
			if !consistencyLevelMatches(thisConsistencyLevel, pLock.effectiveCL) {
				continue
			}
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

func positiveUnrelatedNestedTxHashGuard(w *Watcher, ctx context.Context, ev NewBlock) error {
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
		if blockNumberU > 0 {
			if txreceipt.TxHash != gethtypes.BytesToHash(pLock.message.TxID) {
				delete(w.pending, key)
				continue
			}
		}
		if txreceipt.BlockHash != key.BlockHash {
			delete(w.pending, key)
			continue
		}
		return w.verifyAndPublish(pLock.message, ctx, gethtypes.Hash(pLock.message.TxID), txreceipt)
	}
	return nil
}
