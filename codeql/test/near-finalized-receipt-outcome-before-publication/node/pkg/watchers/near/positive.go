package near

import "context"

func positiveNoFinality(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, Header{}, "", log)
	}
	return nil
}

func positiveFinalityAfterPublication(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		if err := e.processWormholeLog(logger, ctx, job, Header{}, "", log); err != nil {
			return err
		}
	}
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	_, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	return nil
}

func positiveWrongReceiptFinality(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result, otherReceiptOutcome Result) error {
	outcomeBlockHash := otherReceiptOutcome.Get("block_hash")
	_, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, Header{}, "", log)
	}
	return nil
}

func positiveTxBlockFinalityOnly(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	_, isFinalized := e.finalizer.isFinalized(logger, ctx, job.blockHash)
	if !isFinalized {
		return errNotFinalized()
	}
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, Header{}, "", log)
	}
	return nil
}

func positiveIgnoredBoolean(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, Header{}, "", log)
	}
	return nil
}

func positiveReceiptReassignedAfterProof(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result, replacement Result) error {
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	_, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	receiptOutcome = replacement
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, Header{}, "", log)
	}
	return nil
}

func positiveRightBooleanWrongHeader(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	_, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, Header{}, "", log)
	}
	return nil
}

func positiveHeaderReassignedAfterProof(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	blockHeader, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	blockHeader = Header{}
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, blockHeader, "", log)
	}
	return nil
}

func positiveSeparatelyFetchedHeader(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	_, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	blockHeader, _ := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, blockHeader, "", log)
	}
	return nil
}

func positiveInvertedFinalityGuard(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	logs := outcome.Get("logs")
	blockHeader, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if isFinalized {
		return nil
	}
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, blockHeader, "", log)
	}
	return nil
}

func positiveUnrelatedSameNameFinalizer(e *Watcher, other OtherFinalizer, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	logs := outcome.Get("logs")
	blockHeader, isFinalized := other.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, blockHeader, "", log)
	}
	return nil
}

func positiveDifferentWatcherFinalizer(e *Watcher, other *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	logs := outcome.Get("logs")
	blockHeader, isFinalized := other.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, blockHeader, "", log)
	}
	return nil
}

func positiveDifferentFinalizerValue(e *Watcher, finalizer Finalizer, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	logs := outcome.Get("logs")
	blockHeader, isFinalized := finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, blockHeader, "", log)
	}
	return nil
}
