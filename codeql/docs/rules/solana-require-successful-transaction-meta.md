# Solana Successful Transaction Metadata

Validate that Solana RPC transaction metadata is present and successful before parsing transactions, reading metadata that drives parsing, or calling `processTransaction`.

## Why This Matters

The Solana watcher observes finalized transaction records. Finality alone does not prove that a transaction executed successfully or that useful metadata exists. Treating failed or metadata-less transactions as observations can make the watcher parse logs or transaction contents that should have been rejected at the RPC metadata boundary.

## Examples

### Violation

```go
func observe(txRpc *rpc.GetTransactionResult) error {
	for _, log := range txRpc.Meta.LogMessages {
		_ = log
	}
	return processTransaction(txRpc, txRpc.Meta)
}
```

### Fix

```go
func observe(txRpc *rpc.GetTransactionResult) error {
	if err := validateTransactionMeta(txRpc.Meta); err != nil {
		return err
	}
	return processTransaction(txRpc, txRpc.Meta)
}
```

An equivalent direct guard is also accepted when it proves the same metadata value is non-nil and has `Err == nil` before the sink.

## What The Rule Checks

The rule reports production Solana watcher sinks under `node/pkg/watchers/solana/`: calls to `processTransaction` with transaction metadata, transaction extraction from responses tied to a metadata value, and reads of metadata fields such as `LogMessages` before parsing decisions. It requires a dominating successful-meta proof for the same metadata value.

The rule deliberately ignores tests, generated files, account-subscription/account-ID paths without `*rpc.TransactionMeta`, and `Err` field reads that participate in validation.

## Limitations

The model is intra-procedural. It recognizes `validateTransactionMeta(meta)` only when the returned error is rejected before the sink, and direct `meta != nil && meta.Err == nil` guards only for the same metadata version. It conservatively reports when the metadata or validator error is reassigned before use, when validation happens after the sink, or when an ignored/overwritten validator error breaks the proof.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/solana-require-successful-transaction-meta.md): records the project policy, architecture intent, source references, test matrix, calibration evidence, and acceptance transition.
- [Rule query](../../src/solana-require-successful-transaction-meta.ql): defines the Solana transaction-metadata proof model and sink set.
- [Rule fixtures](../../test/solana-require-successful-transaction-meta/): encode violations, accepted guards, reassignment invalidation, and out-of-scope boundaries.

## Maintainer Notes

The CodeQL ID is `wormhole/go/solana-require-successful-transaction-meta`. Update the query, fixtures, and this page together if Solana watcher transaction-entry paths, `validateTransactionMeta`, or `processTransaction` signatures change.
