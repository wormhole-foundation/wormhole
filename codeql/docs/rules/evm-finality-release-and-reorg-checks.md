# EVM Finality Release And Reorg Checks

Release an EVM watcher pending message from `w.pending` only after local, fail-closed proofs establish the expected finality threshold and reject receipt reorg drift before `(*Watcher).verifyAndPublish`.

## Why This Matters

Pending EVM messages are provisional until the watcher reaches the message's effective consistency level and block-height delay. Even after the delay, the originally observed receipt can be orphaned or replaced by a reorg. The release path must therefore refetch the receipt and prove that the refetched receipt still describes the same transaction in the same block before publishing the pending message.

## Examples

### Violation

```go
for _, pending := range w.pending {
	receipt, _ := connector.TransactionReceipt(ctx, common.BytesToHash(pending.message.TxID))
	return w.verifyAndPublish(pending.message, ctx, common.BytesToHash(pending.message.TxID), receipt)
}
```

This releases a pending message without proving it came from the same watcher pending map, without checking the effective consistency level or height threshold, and without fail-closed handling for refetch errors, nil receipts, transaction-hash drift, or block-hash drift.

### Fix

```go
blockNumberU := ev.Number.Uint64()
thisConsistencyLevel := vaa.ConsistencyLevelFinalized
thisConsistencyLevel = vaa.ConsistencyLevelSafe

for key, pending := range w.pending {
	if !consistencyLevelMatches(thisConsistencyLevel, pending.effectiveCL) {
		continue
	}
	if blockNumberU < pending.height+pending.additionalBlocks {
		continue
	}

	txHash := common.BytesToHash(pending.message.TxID)
	receipt, err := connector.TransactionReceipt(ctx, txHash)
	if errors.Is(err, ethereum.NotFound) {
		continue
	}
	if err != nil {
		continue
	}
	if receipt == nil {
		continue
	}
	if receipt.TxHash != txHash {
		continue
	}
	if receipt.BlockHash != key.BlockHash {
		continue
	}

	return w.verifyAndPublish(pending.message, ctx, txHash, receipt)
}
```

## What The Rule Checks

The V1 rule reports production EVM watcher calls that release `pending.message` through `verifyAndPublish` unless the call is in the currently modeled pending-release shape: a `range` over the same watcher's `w.pending` map with a pending value and key available at the sink.

For each modeled release, all required proofs must be visible in the same function and must be CFG-dominating, fail-closed guards (`continue` or `return`) before the `verifyAndPublish` call:

- the pending entry comes from the same watcher's `w.pending` range;
- the current effective consistency level matches `pending.effectiveCL`;
- the current block height has reached `pending.height + pending.additionalBlocks`;
- the receipt is refetched with the pending message transaction hash;
- not-found/orphaned receipt errors are rejected;
- generic non-nil receipt-fetch errors are rejected;
- nil receipts are rejected;
- the refetched receipt's `TxHash` matches the pending message transaction hash; and
- the refetched receipt's `BlockHash` matches the pending range key's block hash.

Receipt execution success is intentionally delegated to the sibling rule `evm-require-successful-receipt-before-observation`; this rule checks finality-release and reorg-consistency preconditions only.

## Limitations

The V1 source model is intentionally bounded to current pending-release shapes in `node/pkg/watchers/evm/`: direct `range` loops over `w.pending`, `pending.message`, `pending.effectiveCL`, `pending.height`, `pending.additionalBlocks`, range keys with `BlockHash`, `thisConsistencyLevel`, `blockNumberU`, `TransactionReceipt`, `errors.Is(..., ErrNoResult|NotFound)`, nil/error checks, and direct transaction/block hash comparisons.

The rule does not yet model reobservation/v2 paths, helper-propagated proofs, arbitrary wrapper functions, alternate field names, interprocedural invariants, or broader receipt provenance. Add those only with fixtures that preserve the fail-closed, CFG-dominating proof requirement. It also does not prove receipt success; keep that responsibility in the sibling receipt-success rule.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/evm-finality-release-and-reorg-checks.md): records the bounded V1 policy, source evidence, and lifecycle decisions.
- [Rule query](../../src/evm-finality-release-and-reorg-checks.ql): defines the V1 source shapes, pending-release sink, required proof predicates, and diagnostic messages.
- [Rule fixtures](../../test/evm-finality-release-and-reorg-checks/): encode missing-proof cases, accepted guard shapes, and bounded-source behavior.
- [Sibling receipt-success rule](./evm-require-successful-receipt-before-observation.md): checks that a receipt succeeded before observation or publication.
- [EVM verify-and-publish gate](./evm-verify-and-publish-gate.md): documents the approved publication gate for EVM watcher messages.

## Maintainer Notes

The CodeQL ID is `wormhole/go/evm-finality-release-and-reorg-checks`. Update the query, fixtures, and this page together when pending storage, pending release loops, consistency-level calculation, height calculation, receipt refetching, or reorg checks change. Do not broaden V1 by trusting helper names or returned tuple shapes alone; model helper propagation separately and require the same CFG-dominating fail-closed proofs at the publication sink.
