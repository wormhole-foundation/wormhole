# NEAR Finalized Receipt Outcome Before Publication

Require NEAR watcher receipt-outcome logs to be published only after proving that the same `receipt_outcome.block_hash` is finalized.

## Why This Matters

NEAR receipt outcomes can expose Wormhole logs before the receipt outcome's block is finalized. A watcher that publishes logs from an unfinalized receipt outcome can observe data that is later invalidated by consensus reorganization or finality lag. The local publication path should therefore bind the parsed logs, the finality proof, and the header passed to publication to the same receipt outcome.

## Examples

### Violation

```go
func observe(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, Header{}, "", log)
	}
	return nil
}
```

```go
func observe(e *Watcher, other *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
	outcomeBlockHash := receiptOutcome.Get("block_hash")
	blockHeader, isFinalized := other.finalizer.isFinalized(logger, ctx, outcomeBlockHash.String())
	if !isFinalized {
		return errNotFinalized()
	}

	outcome := receiptOutcome.Get("outcome")
	logs := outcome.Get("logs")
	for _, log := range logs.Array() {
		return e.processWormholeLog(logger, ctx, job, blockHeader, "", log)
	}
	return nil
}
```

### Fix

```go
func observe(e *Watcher, logger *Logger, ctx context.Context, job *Job, receiptOutcome Result) error {
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
```

## What The Rule Checks

The rule reports production calls to `(*Watcher).processWormholeLog` in `node/pkg/watchers/near/*.go` when the published log is the direct range-loop value from `logs.Array()` and `logs` was derived from `receiptOutcome.Get("outcome").Get("logs")` in the same function.

The implementation accepts that publication only when all of these local correlations hold:

- the finality proof is a `(*Finalizer).isFinalized` call reached through the same watcher receiver as the `processWormholeLog` sink, namely `watcher.finalizer.isFinalized(...)` for the same `watcher.processWormholeLog(...)` receiver;
- the proof's block hash argument is the `String()` value of `receiptOutcome.Get("block_hash")` for the same receipt outcome that supplied the iterated logs;
- the boolean result returned by that proof dominates the sink on the true/finalized branch, including fail-closed `if !isFinalized { return ... }` and guarded `if isFinalized { publish(...) }` shapes; and
- the block header passed to `processWormholeLog` is the header result returned by that same `isFinalized` call and has not been reassigned before publication.

Tests, generated files, non-NEAR watcher files, unrelated `isFinalized` methods, different watcher receivers, wrong receipt outcomes, proofs performed after publication, inverted guards, reassigned receipt outcomes, and reassigned or separately fetched headers are not accepted as proofs for this sink.

## Limitations

This is an intentionally bounded, intra-function rule. It does not propagate receipt-outcome log provenance or finality proofs through arbitrary helpers, wrappers, method calls, interfaces, closures, goroutines, or stored state. The modeled sink is the production `Watcher.processWormholeLog` call reached while directly iterating logs derived from `receiptOutcome.Get("outcome").Get("logs")`; direct alternate publication sinks are out of scope.

The loop model is also direct: the published argument must be the range value from `logs.Array()` in the same loop body. If future production code publishes logs after helper parsing, slice aliasing beyond the current local assignments, callback dispatch, or another publication API, add fixtures and recalibrate the query before expanding this contract.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/near-finalized-receipt-outcome-before-publication.md)
- [Rule query](../../src/near-finalized-receipt-outcome-before-publication.ql)
- [Rule fixtures](../../test/near-finalized-receipt-outcome-before-publication/)

## Maintainer Notes

The CodeQL ID is `wormhole/go/near-finalized-receipt-outcome-before-publication`. Keep the same-receiver, same-receipt-outcome, true-branch dominance, and same-proof-header requirements together when changing this rule; weakening any one of them can turn unrelated finality checks into false negatives. Update the query, fixtures, rule contract, and this page together when NEAR watcher publication code changes.
