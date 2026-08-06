# Solana Commitment Match Before Publication

Check a decoded Solana message commitment against the watcher's configured commitment before publishing the message or scheduling an instruction-account retry.

## Why This Matters

The Solana watcher decodes a requested consistency level from account and shim message data, converts it into a watcher commitment, and then decides whether the observation should be published. Publishing to `msgC`, or scheduling `retryFetchMessageAccount`, before proving that decoded commitment is acceptable can let a lower-commitment observation flow through a watcher that is configured for a stricter commitment.

## Examples

### Violation

```go
func process(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}

	s.msgC <- &MessagePublication{}
}
```

Instruction retry scheduling has the same requirement:

```go
func schedule(s *SolanaWatcher, ctx Context, rpcClient *RPCClient, acc PublicKey, sig Signature, data *ShimPostMessageData, isReobservation bool) {
	commitment, err := data.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	_ = commitment

	RunWithScissors(ctx, nil, "retryFetchMessageAccount", func(ctx Context) error {
		s.retryFetchMessageAccount(ctx, rpcClient, acc, 0, 0, isReobservation, sig)
		return nil
	})
}
```

### Fix

```go
func process(s *SolanaWatcher, proposal *MessagePublicationAccount, isReobservation bool) {
	commitment, err := accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}

	s.msgC <- &MessagePublication{}
}
```

For instruction retry scheduling, guard the converted instruction commitment before the direct retry call or before the `RunWithScissors` scheduling call:

```go
func schedule(s *SolanaWatcher, ctx Context, rpcClient *RPCClient, acc PublicKey, sig Signature, data *ShimPostMessageData, isReobservation bool) {
	commitment, err := data.ConsistencyLevel.Commitment()
	if err != nil {
		return
	}
	if !s.checkCommitment(commitment, isReobservation) {
		return
	}

	RunWithScissors(ctx, nil, "retryFetchMessageAccount", func(ctx Context) error {
		s.retryFetchMessageAccount(ctx, rpcClient, acc, 0, 0, isReobservation, sig)
		return nil
	})
}
```

## What The Rule Checks

The rule reports production Go code under `node/pkg/watchers/solana/` when a decoded commitment reaches a publication or instruction-retry sink without both required proofs:

- the conversion error from the same decoded commitment conversion is rejected after the conversion; and
- the same `SolanaWatcher` receiver that publishes or schedules the observation has proven `checkCommitment(commitment, isReobservation)` true before the sink.

The account and shim publication sink is a send to a watcher's `msgC` field, such as `s.msgC <- observation`. Account commitments are conversions from `accountConsistencyLevelToCommitment(proposal.ConsistencyLevel)`. Shim commitments are conversions from `postMessage.ConsistencyLevel.Commitment()`.

The instruction-retry sink is either a direct call to `s.retryFetchMessageAccount(...)` or a direct `RunWithScissors(ctx, errC, "retryFetchMessageAccount", func(...) { s.retryFetchMessageAccount(...) })` scheduling call. For those paths, the relevant decoded commitment is the instruction `PostMessageData.ConsistencyLevel.Commitment()` conversion.

Accepted `checkCommitment` proof shapes are intentionally narrow:

- `if !s.checkCommitment(commitment, isReobservation) { return }` before the sink;
- `if s.checkCommitment(commitment, isReobservation) { ... sink ... }`; and
- `if !s.checkCommitment(commitment, isReobservation) { ... } else { ... sink ... }`.

The proof must call the `checkCommitment` method on a `SolanaWatcher`, pass the converted commitment as the first argument, and use the same watcher receiver as the later `msgC` send or `retryFetchMessageAccount` receiver. A helper with the same method name, a different watcher instance, an ignored check result, or a check after the sink does not satisfy the rule.

The conversion error guard must occur after the conversion and before the sink. A stale guard from an earlier assignment, an ignored conversion error, or a reassignment of the error variable before the sink invalidates the conversion proof. Reassigning the commitment after `checkCommitment` also invalidates the commitment proof for later publication.

The rule deliberately ignores tests, generated files, non-Solana-watcher files, unrelated `RunWithScissors` jobs, close-event paths that delegate through already modeled account processing, and manual comparisons such as `commitment != CommitmentConfirmed` that do not use the watcher's canonical `checkCommitment` policy.

## Limitations

The model is local and syntax-bounded. It does not prove arbitrary interprocedural wrappers, local function variables passed to `RunWithScissors`, non-inline retry scheduling, deep receiver aliasing, container-stored commitments, or custom helper predicates equivalent to `checkCommitment`. Branch proof recognition is limited to the supported shapes above and requires a direct `return` in the failing branch. Receiver matching is exact enough to avoid accepting checks on another watcher, but it may miss complex aliases that are semantically equivalent.

The query ties instruction scheduling to direct `retryFetchMessageAccount` calls and direct `RunWithScissors` calls named `"retryFetchMessageAccount"`. Thin lifecycle wrappers, asynchronous helper boundaries, and renamed scheduling jobs are out of scope unless the query and fixtures are extended together.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/solana-commitment-match-before-publication.md): records the intended project policy, source references, test matrix, and calibration notes when present in the lint-builder artifacts.
- [Rule query](../../src/solana-commitment-match-before-publication.ql): defines decoded commitment conversion, conversion-error proof, receiver-matched `checkCommitment` proof, publication sinks, and instruction retry scheduling sinks.
- [Rule fixtures](../../test/solana-commitment-match-before-publication/): encode account and shim `msgC` sends, direct and scheduled instruction retries, accepted branch shapes, receiver mismatches, stale guards, reassignment invalidation, and near-miss carve-outs.

## Maintainer Notes

The CodeQL ID is `wormhole/go/solana-commitment-match-before-publication`. Keep this page, the query, and the fixtures synchronized if `checkCommitment`, `accountConsistencyLevelToCommitment`, `PostMessageData.ConsistencyLevel.Commitment`, `msgC`, `retryFetchMessageAccount`, or `RunWithScissors` scheduling conventions change. Add fixtures before broadening receiver aliasing, branch-shape recognition, or wrapper support so the rule remains high precision.
