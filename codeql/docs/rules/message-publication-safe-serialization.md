# MessagePublication Safe Serialization

Use current `MessagePublication` binary serialization APIs for production data: `MarshalBinary` and `UnmarshalBinary`.

## Why This Matters

The deprecated `(*MessagePublication).Marshal` and `common.UnmarshalMessagePublication` helpers omit `Unreliable` and `verificationState`. Those fields affect reobservation behavior and transfer-verifier / txverifier decisions, including rejected or anomalous messages. Using the old format for current Governor, Notary, pending-message, or transport data can erase security state during persistence or recovery.

## Examples

### Violation

```go
func write(p *PendingTransfer) ([]byte, error) {
	return p.Msg.Marshal()
}
```

```go
func read(buf []byte) (*common.MessagePublication, error) {
	return common.UnmarshalMessagePublication(buf)
}
```

### Fix

```go
func write(p *PendingTransfer) ([]byte, error) {
	return p.Msg.MarshalBinary()
}
```

```go
func read(buf []byte) (*common.MessagePublication, error) {
	msg := &common.MessagePublication{}
	if err := msg.UnmarshalBinary(buf); err != nil {
		return nil, err
	}
	return msg, nil
}
```

## What The Rule Checks

The rule reports production Go under `node/` that calls or captures the deprecated `MessagePublication` helpers. It resolves targets rather than relying on import spelling, so aliases, type aliases, embedded/promoted `Marshal` methods, parenthesized calls, bound method values, and selector captures are covered.

The only accepted deprecated read is the bounded old Governor migration path: `node/pkg/db/governor.go`, inside `UnmarshalPendingTransfer`, under the true branch of the declared `isOld` parameter. Deprecated writes are never excepted. The rule ignores tests, generated files, non-`node/` code, helper definitions in `chainlock.go`, JSON/VAA/protobuf serialization, unrelated `Marshal` methods, and code that routes a message without serializing it.

## Limitations

The model does not prove arbitrary indirect invocations through reflection, `interface{}`, generic higher-order plumbing, or wrappers that erase the static `MessagePublication` target. Future migration exceptions should be added only when statically tied to an old-version discriminator and read-only for old bytes.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/message-publication-safe-serialization.md): policy, Governor exception, source evidence, fixtures, and calibration.
- [Acceptance review](../../../.codeql-lint-builder/runs/message-publication-safe-serialization/08-review-and-learn-acceptance-2026-07-15.md): final review notes for the migration exception and fixture coverage.
- [Rule query](../../src/message-publication-safe-serialization.ql): target-resolution, selector-capture, and old Governor branch predicates.
- [Rule fixtures](../../test/message-publication-safe-serialization/): deprecated/current calls, aliases, embedded types, captures, tests, and non-node scope cases.

The rule artifact cites Wormhole source and hardening commit `a708838a9db46c02503ce38e01442852c2d88578`, but this checkout does not contain the Wormhole repository, so this page cannot provide verified version-pinned links to those files.

## Maintainer Notes

The CodeQL ID is `wormhole/go/message-publication-safe-serialization`. Keep the Governor exception bound to the resolved `isOld` parameter and true-branch AST shape; do not weaken it to identifier spelling. Update fixtures whenever new compatibility branches, wrapper serializers, or `MessagePublication` package paths are introduced.
