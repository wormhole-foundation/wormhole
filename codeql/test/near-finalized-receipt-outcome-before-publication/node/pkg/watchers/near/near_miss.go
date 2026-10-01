package near

import "context"

func nearMissExistsOnly(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	if !outcomeBlockHash.Exists() {
		return errNotFinalized()
	}
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, Header{}, "", log)
	}
	return nil
}

func nearMissCacheMutationOnly(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	e.finalizer.setFinalized(outcomeBlockHash.String())
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, Header{}, "", log)
	}
	return nil
}
