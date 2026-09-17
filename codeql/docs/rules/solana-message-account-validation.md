# Solana Message Account Validation

Create Solana watcher message account data with `NewMessageAccountData` and reject its error before parsing or processing the value.

## Why This Matters

A Solana account stores program state as bytes. In the Wormhole Solana watcher, `NewMessageAccountData` is the validation boundary that checks those bytes before they become `MessageAccountData`. Constructing that type directly, or using the constructor result after an error, lets unvalidated data reach parsing or processing code without the constructor's discriminator and length checks.

## Examples

### Violation

```go
func process(raw []byte) {
	data := MessageAccountData{Data: raw}
	ParseMessagePublicationAccount(data)
}
```

### Fix

```go
func process(raw []byte) error {
	data, err := NewMessageAccountData(raw)
	if err != nil {
		return err
	}
	ParseMessagePublicationAccount(data)
	return nil
}
```

## What The Rule Checks

The rule reports calls to `ParseMessagePublicationAccount` and `processMessageAccount` in production files under `node/pkg/watchers/solana/` when the relevant argument cannot be traced to a successful `NewMessageAccountData` call. It accepts local aliases, one pointer round trip, and checked factory wrappers up to the modeled depth. Fix an alert by routing the raw bytes through the constructor and ensuring its error is rejected on every path to the reported call.

The rule deliberately ignores tests, generated files, similarly named functions outside the Solana watcher, and calls inside the parser or processor implementations themselves.

## Limitations

The model is intentionally local and recognizes at most two levels of safe factory wrapping. More complex interprocedural flows, container storage, or equivalent validators with another name may be reported. An accepted error guard must start on a later source line than the constructor assignment; this matches gofmt-normalized watcher code but can conservatively report a semicolon-separated valid guard. Conversely, the rule proves use of the designated constructor and its error guard; it does not independently verify the constructor's implementation.

## Learn More

- [Solana accounts](https://solana.com/docs/core/accounts): mutable Solana program state is stored in account data; this documentation is maintained online and is not version-pinned.
- [Rule contract](../../../.codeql-lint-builder/rules/solana-message-account-validation.md): records the project policy, architecture intent, source references, test matrix, and calibration evidence.
- [Acceptance review](../../../.codeql-lint-builder/runs/solana-message-account-validation/08-review-and-learn-acceptance-2026-07-13.md): records the final stale-guard, source-order, performance, and acceptance gate.
- [Rule query](../../src/solana-message-account-validation.ql): defines the enforced constructor provenance, error guard, scope, and report location.
- [Rule fixtures](../../test/solana-message-account-validation/): encode accepted constructor flows, violations, and boundary cases.

The rule artifact cites Wormhole source and hardening commit `e889d725f`, but this checkout does not contain the Wormhole repository, so this page cannot provide verified version-pinned links to them.

## Maintainer Notes

The CodeQL ID is `wormhole/go/solana-message-account-validation`. The query depends on local data flow, global value numbering, and dominating error guards. Update the query and all three fixture categories together if parser entry points, the constructor name, watcher paths, or the allowed factory depth changes.
