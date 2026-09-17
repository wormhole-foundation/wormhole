package near

import "context"

func errNotFinalized() error { return nil }

func negativeSameOutcomeFinality(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	logs := outcome.Get("logs")
	blockHeader, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, blockHeader, "", log)
	}
	return nil
}

func negativeTrueBranch(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	logs := outcome.Get("logs")
	blockHeader, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if isFinalized {
		for _, log := range logs.Array() {
			return e.processWormholeLog(logger, ctx, job, blockHeader, "", log)
		}
	}
	return nil
}

func negativeDelayedFinalityRetry(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	logs := outcome.Get("logs")
	blockHeader, isFinalized := e.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}
	for _, log := range logs.Array() {
		if err := e.processWormholeLog(logger, ctx, job, blockHeader, "", log); err != nil {
			return err
		}
	}
	return nil
}

func negativeReobservationSamePath(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	job.isReobservation = true
	return negativeSameOutcomeFinality(e, logger, ctx, job, receiptOutcome)
}
