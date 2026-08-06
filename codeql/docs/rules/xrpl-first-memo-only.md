# XRPL First Memo Only

Wormhole Core and NTT parsing in the XRPL watcher must inspect only `Memos[0]`, never scan or select a later memo.

## Why This Matters

An XRP Ledger transaction can carry an ordered array of arbitrary memos. Wormhole assigns canonical meaning to the first memo for Core and Native Token Transfer (NTT) messages. If the watcher scans the array, a malformed or unrelated first memo can be bypassed by a later memo that looks like a Wormhole message, changing which payload the watcher treats as canonical.

## Examples

### Violation

```go
func parse(tx Transaction) bool {
	for _, memo := range tx.Memos {
		if memo.MemoFormat == coreMemoFormat {
			return true
		}
	}
	return false
}
```

### Fix

```go
func parse(tx Transaction) bool {
	if len(tx.Memos) == 0 {
		return false
	}
	return tx.Memos[0].MemoFormat == coreMemoFormat
}
```

## What The Rule Checks

The rule reports range loops, nonzero or dynamic indexing, and scanning-helper calls that consume an XRPL transaction's `Memos` collection while recognizing `coreMemoFormat` or `nttMemoFormat`. It covers typed `Transaction.Memos` values and the `"Memos"` entry of `FlatTransaction` values in production files under `node/pkg/watchers/xrpl/`.

Fix an alert by checking that the memo collection is nonempty and inspecting index zero directly. A malformed first memo must cause rejection rather than a search for a later fallback. The rule ignores tests, generated files, unrelated memo formats, unrelated types with a `Memos` field, local memo arrays, and iteration over the selected first memo's data bytes.

## Limitations

Recognition is tied to the current watcher path, transaction type names, and the identifiers `coreMemoFormat` and `nttMemoFormat`. Collection flow through unmodeled helpers or renamed protocol constants can evade the rule, while a helper that scans for other purposes may be reported when called with one of the recognized formats.

## Learn More

- [XRPL transaction Memos field](https://xrpl.org/docs/references/protocol/transactions/common-fields#memos-field): specifies that `Memos` is an array carrying arbitrary messaging data; this documentation is maintained online and is not version-pinned.
- [Rule contract](../../../.codeql-lint-builder/rules/xrpl-first-memo-only.md): records the Wormhole first-memo policy, source references, test matrix, and calibration evidence.
- [Acceptance review](../../../.codeql-lint-builder/runs/xrpl-first-memo-only/08-review-and-learn-acceptance-2026-07-13.md): records the final wrapper-model gate and acceptance decision.
- [Rule query](../../src/xrpl-first-memo-only.ql): defines the recognized formats, collection sources, scanning behavior, and exclusions.
- [Rule fixtures](../../test/xrpl-first-memo-only/): encode direct scans, helper scans, first-index access, and near misses.

The rule artifact cites Wormhole source, regression tests, and fix commit `0d2738c68`, but this checkout does not contain the Wormhole repository, so this page cannot provide verified version-pinned links to them.

## Maintainer Notes

The CodeQL ID is `wormhole/go/xrpl-first-memo-only`. The query uses local data flow and a transitive local call relation to connect format recognizers to memo access. Update the query and fixtures together when XRPL transaction representations, memo-format constants, helper structure, or watcher paths change.
