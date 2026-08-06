# Require A Validated XRPL Transaction

Prove that an XRPL transaction's `Validated` field is true before passing that same transaction to a Wormhole parser entry point.

## Why This Matters

An XRPL API response may describe a transaction before its result is final. The Wormhole parser can turn a transaction into an observation, so parsing a response that is not from a validated ledger can make provisional transaction data eligible for downstream processing. A check after parsing, a check of another transaction, or a check invalidated by reassignment does not establish the required precondition.

## Examples

### Violation

```go
func parse(parser *Parser, tx TxResponse) {
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}
```

### Fix

```go
func parse(parser *Parser, tx TxResponse) {
	if !tx.Validated {
		return
	}
	parser.ParseTxResponse(&txResponseV2{TxResponse: tx})
}
```

## What The Rule Checks

The rule reports calls to `Parser.ParseTransactionStream` and `Parser.ParseTxResponse` in production files under `node/pkg/watchers/xrpl/` unless a dominating branch proves `Validated == true` for the parsed transaction. It recognizes a direct field check, a local boolean alias, stream transactions, and the `txResponseV2` wrapper used by `ParseTxResponse`. It rejects proofs invalidated by replacing the transaction, wrapper, wrapped transaction, or validated field before the call.

Fix an alert by returning or continuing when `Validated` is false before invoking the parser. The rule ignores tests, generated files, same-named methods on other receiver types, parser methods outside the XRPL watcher, and checks of unrelated transaction values.

## Limitations

The proof must be visible in the same function and match the modeled parser and wrapper shapes. Validation performed in a helper, encoded through a different wrapper, or represented by another API may be reported even if semantically safe. Mutation invalidation uses source position and can conservatively reject a mutation on a branch that cannot reach the parser call. Taking the transaction's address also conservatively invalidates the proof, even if a pointer is rebound before use. The rule checks ledger validation only; it does not prove transaction success or any other parser precondition.

## Learn More

- [XRPL transaction finality](https://xrpl.org/docs/concepts/transactions/finality-of-results): explains validated ledgers and when transaction outcomes are final; this documentation is maintained online and is not version-pinned.
- [Rule contract](../../../.codeql-lint-builder/rules/xrpl-require-validated-transaction.md): records the project policy, architecture intent, source references, test matrix, and calibration evidence.
- [Acceptance review](../../../.codeql-lint-builder/runs/xrpl-require-validated-transaction/08-review-and-learn-acceptance-2026-07-13.md): records the final mutation, pointer, precision, and acceptance gate.
- [Rule query](../../src/xrpl-require-validated-transaction.ql): defines parser entry points, accepted proof shapes, mutation invalidation, and scope.
- [Rule fixtures](../../test/xrpl-require-validated-transaction/): encode dominating checks, invalid checks, mutation cases, wrappers, and near misses.

The rule artifact cites Wormhole source for the caller contract and finality policy, but this checkout does not contain the Wormhole repository, so this page cannot provide verified version-pinned links to it.

## Maintainer Notes

The CodeQL ID is `wormhole/go/xrpl-require-validated-transaction`. The query uses global value numbering and control-flow dominance to connect the parsed value to its validation proof. Update the query and fixtures together if parser methods, receiver types, response wrappers, transaction fields, or watcher paths change.
