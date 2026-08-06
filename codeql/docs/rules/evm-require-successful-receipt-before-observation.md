# EVM Successful Receipt Before Observation

Require proof that an EVM transaction receipt succeeded before parsing its publication logs or passing it to `(*Watcher).verifyAndPublish`. The proof may be local or inherited from the checked successful return of the modeled `MessageEventsForTransaction` helper.

## Why This Matters

Transaction inclusion, block finality, a non-nil receipt, and a matching transaction hash do not prove successful EVM execution. Observation code should reject failed receipts before deriving or publishing Wormhole messages.

## Examples

### Violation

```go
receipt, err := connector.TransactionReceipt(ctx, txHash)
if err != nil || receipt == nil {
	return err
}
return w.verifyAndPublish(msg, ctx, txHash, receipt)
```

### Fix

```go
receipt, err := connector.TransactionReceipt(ctx, txHash)
if err != nil || receipt == nil {
	return err
}
if receipt.Status != gethTypes.ReceiptStatusSuccessful {
	return fmt.Errorf("transaction failed")
}
return w.verifyAndPublish(msg, ctx, txHash, receipt)
```

## What The Rule Checks

The rule reports production EVM watcher calls to `verifyAndPublish` whose receipt argument lacks a dominating success-status proof. It also reports direct, indexed, aliased, and supported one-hop-helper parsing of `receipt.Logs` before the same receipt is proven successful.

Tests, generated files, non-EVM watcher code, and uses dominated by a same-receipt `Status == ReceiptStatusSuccessful` proof are excluded. A checked successful `MessageEventsForTransaction` tuple also proves its returned receipt successful and binds its returned messages to that receipt. Reassigning the receipt or message slice, mixing tuple values from different calls, or failing to reject the paired error invalidates the inherited proof.

## Helper Summary

The query models the canonical `MessageEventsForTransaction` implementation in `node/pkg/watchers/evm/by_transaction.go`. Every nil-error return must remain dominated by a successful-status proof for the returned receipt before the summary activates.

At the caller, the returned receipt, message slice, and checked error must come from the same tuple. The published message must be read from that slice. Unchecked or overwritten errors, mixed helper calls, reassigned receipts or slices, reassigned range values, and mutated slice elements remain findings. Arbitrary wrappers remain unsupported unless they receive an equally precise semantic summary.

Existing-database calibration reports zero findings in 16.7 seconds, removing the former `reobserve.go` false positives without requiring redundant local receipt-status checks.

Receipt-log provenance is deliberately shallow and covers the supported range, index, local alias, and one-hop parse-helper shapes encoded by the fixtures.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/evm-require-successful-receipt-before-observation.md)
- [Acceptance review](../../../.codeql-lint-builder/runs/evm-require-successful-receipt-before-observation/10-final-review-and-acceptance-2026-07-15.md)
- [Tuple-postcondition bugfix](../../../.codeql-lint-builder/runs/evm-require-successful-receipt-before-observation/11-tuple-postcondition-bugfix-2026-07-17.md)
- [Rule query](../../src/evm-require-successful-receipt-before-observation.ql)
- [Rule fixtures](../../test/evm-require-successful-receipt-before-observation/)

## Maintainer Notes

The CodeQL ID is `wormhole/go/evm-require-successful-receipt-before-observation`. Update the query, fixtures, contract, and this page together when receipt acquisition, log parsing, or `verifyAndPublish` call paths change. The helper summary is intentionally restricted to the canonical implementation path and proves its successful-return behavior; matching only a function name or tuple shape is insufficient.
