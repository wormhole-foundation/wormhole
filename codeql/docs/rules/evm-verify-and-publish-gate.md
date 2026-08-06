# EVM Verify-And-Publish Gate

Publish EVM watcher `MessagePublication` values only through `(*Watcher).verifyAndPublish`.

## Why This Matters

The EVM watcher’s `msgC` channel is the handoff from chain observation code to the shared processor. `verifyAndPublish` is the security gate that rejects nil messages, applies transfer-verifier logic when configured, updates `verificationState`, and then performs the channel send. A direct send to `w.msgC` can publish an EVM token-bridge message without the receipt-backed transfer-verifier state update that downstream code relies on.

## Examples

### Violation

```go
func publish(w *Watcher, msg *common.MessagePublication) {
	w.msgC <- msg
}
```

```go
func publish(w *Watcher, msg *common.MessagePublication) {
	out := w.msgC
	out <- msg
}
```

### Fix

```go
func publish(w *Watcher, msg *common.MessagePublication, receipt *Receipt) error {
	return w.verifyAndPublish(msg, context.Background(), msg.TxID, receipt)
}
```

## What The Rule Checks

The rule reports production Go sends under `node/pkg/watchers/evm/` when the send target is the EVM watcher's `msgC` publication channel and the send is not inside `func (w *Watcher) verifyAndPublish(...)`. It recognizes direct field sends, ordered same-function local alias chains, and directly called thin helpers whose channel parameter is passed `w.msgC` or a supported alias.

The rule ignores tests, generated files, non-EVM watcher channels, channel wiring in constructors, channel reads, unrelated same-typed channels, and helpers that only construct or return `*common.MessagePublication` without publishing.

## Limitations

The accepted model is intentionally bounded. It does not follow channel identity through closures, wrapper structs, function values, method expressions, interfaces, helper factories, reflection, unsafe pointers, package globals, or heterogeneous containers. Production calibration found no such bypass path. The rule also trusts the entire `verifyAndPublish` method body; it does not prove that the method internally verifies before its approved send.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/evm-verify-and-publish-gate.md): policy, source evidence, test matrix, calibration, and accepted unsupported boundaries.
- [Acceptance review](../../../.codeql-lint-builder/runs/evm-verify-and-publish-gate/08-review-and-learn-acceptance-2026-07-14.md): final gate checklist and residual risk.
- [Rule query](../../src/evm-verify-and-publish-gate.ql): scope, channel identity, alias, helper, and approved-body predicates.
- [Rule fixtures](../../test/evm-verify-and-publish-gate/): direct, alias, helper, precision, and unsupported-boundary cases.

The rule artifact cites Wormhole source and hardening commit `983dd07551557530a337dcbff5bd579564e57426`, but this checkout does not contain the Wormhole repository, so this page cannot provide verified version-pinned links to those files.

## Maintainer Notes

The CodeQL ID is `wormhole/go/evm-verify-and-publish-gate`. Update the query, fixtures, and this page together if the EVM watcher publication channel, `Watcher` package path, `MessagePublication` type, or approved publication gate changes. Do not broaden the exception to a second helper by name alone; first decide whether the architecture now has multiple approved gates.
