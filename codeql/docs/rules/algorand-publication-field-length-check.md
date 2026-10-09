# Algorand Publication Field Length Check

Prove Algorand `publishMessage` nonce and sequence byte fields are exactly 8 bytes before decoding them with `binary.BigEndian.Uint64`.

## Why This Matters

Algorand watcher code converts chain data into canonical Wormhole `MessagePublication` values. The publication nonce comes from `ApplicationArgs[2]`, and the sequence comes from the first log entry. `binary.BigEndian.Uint64` requires an 8-byte input; malformed field lengths can panic the watcher before the observation is skipped. Container bounds, app ID, method-name checks, and contract-side `Itob` expectations do not prove the exact byte width of the field being decoded.

## Examples

### Violation

```go
func build(at ApplicationTransaction, ed EvalDelta) MessagePublication {
	nonce := binary.BigEndian.Uint64(at.ApplicationArgs[2])
	sequence := binary.BigEndian.Uint64([]byte(ed.Logs[0]))
	return MessagePublication{Nonce: uint32(nonce), Sequence: sequence}
}
```

### Fix

```go
func build(at ApplicationTransaction, ed EvalDelta) (MessagePublication, bool) {
	if len(at.ApplicationArgs[2]) != 8 || len([]byte(ed.Logs[0])) != 8 {
		return MessagePublication{}, false
	}
	nonce := binary.BigEndian.Uint64(at.ApplicationArgs[2])
	sequence := binary.BigEndian.Uint64([]byte(ed.Logs[0]))
	return MessagePublication{Nonce: uint32(nonce), Sequence: sequence}, true
}
```

## What The Rule Checks

The rule reports production Go under `node/pkg/watchers/algorand/` and `pkg/watchers/algorand/` when an Algorand publication function decodes `at.ApplicationArgs[2]` or `[]byte(ed.Logs[0])` with `binary.BigEndian.Uint64` without a dominating exact `len(value) == 8` proof for the same value.

It recognizes direct `binary.BigEndian.Uint64(...)` calls, local aliases of `binary.BigEndian`, local aliases of the nonce or sequence bytes, stale guards invalidated by reassignment of the alias or exact indexed source, and thin local helper calls whose parameter reaches an internal `Uint64`. A checked helper is safe only when the helper enforces exact length and every relevant helper result published into `MessagePublication.Nonce` or `MessagePublication.Sequence` is dominated by rejection of the helper error. The rule ignores tests, generated files, non-Algorand watchers, non-publication decodes, container-bounds-only code, and typed/fixed-width values already produced by checked parsers.

## Limitations

The model is intentionally bounded to the two Wormhole Algorand publication fields and thin local helpers in watcher code. Deep interprocedural propagation, non-local abstractions, equivalent checked parsers with different shapes, and publication construction hidden behind complex containers may be missed. The query proves dominance syntactically with current guard forms; unusual but safe control flow may need additional fixtures before being accepted.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/algorand-publication-field-length-check.md): records the exact field definitions, guard contract, helper treatment, bypass risks, tests, and calibration evidence.
- [Second-gate return report](../../../.codeql-lint-builder/runs/algorand-publication-field-length-check/09-second-gate-return-2026-07-15.md): records the final checked-helper publication-use fix and zero-result recalibration.
- [Rule query](../../src/algorand-publication-field-length-check.ql): defines exact field sources, `Uint64` sinks, dominance, stale-guard invalidation, and thin-helper modeling.
- [Rule fixtures](../../test/algorand-publication-field-length-check/): encode unguarded direct decodes, non-exact checks, stale guards, `BigEndian` aliases, helper bypasses, checked-helper error handling, and exclusions.

The rule artifact cites Wormhole Algorand watcher source, malformed-length regression tests, contract code, and hardening commit `5ce968ff1638d353f9bfe8c94461f9583eaeeedf`, but this checkout does not contain the Wormhole repository, so this page cannot provide verified version-pinned links to them.

## Maintainer Notes

The CodeQL ID is `wormhole/go/algorand-publication-field-length-check`. Update query and fixtures together if Algorand watcher paths, nonce/sequence source expressions, publication construction, helper return conventions, or accepted exact-length guard shapes change.
