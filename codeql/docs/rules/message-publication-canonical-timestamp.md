# MessagePublication Canonical Timestamp

Convert chain-derived Unix-second timestamps with `vaa.TimeFromUnix` before assigning them to `common.MessagePublication.Timestamp`.

## Why This Matters

`MessagePublication.Timestamp` is part of the VAA signing and wire-format path, where timestamps serialize at `uint32` precision. Direct `time.Unix(...)` conversion, often after `int64(...)` or `uint32(...)` casts, bypasses the SDK-owned range check that rejects negative and above-`math.MaxUint32` values. `vaa.TimeFromUnix` centralizes that protocol bound and forces callers to fail closed on invalid chain timestamps.

## Examples

### Violation

```go
func build(blockTime uint64) common.MessagePublication {
	return common.MessagePublication{
		Timestamp: time.Unix(int64(blockTime), 0),
	}
}
```

```go
func build(blockTime uint64) common.MessagePublication {
	timestamp, _ := vaa.TimeFromUnix(blockTime)
	return common.MessagePublication{Timestamp: timestamp}
}
```

### Fix

```go
func build(blockTime uint64) (common.MessagePublication, error) {
	timestamp, err := vaa.TimeFromUnix(blockTime)
	if err != nil {
		return common.MessagePublication{}, err
	}
	return common.MessagePublication{Timestamp: timestamp}, nil
}
```

## What The Rule Checks

The rule reports production Go under `node/` when a keyed composite literal for `common.MessagePublication` assigns `Timestamp:` from unsafe Unix-second conversion provenance. Covered positives include direct `time.Unix`, local temporaries, a thin helper returning only `time.Time`, and `vaa.TimeFromUnix` results whose paired error is ignored, overwritten, not checked, or checked without preventing publication.

It accepts `vaa.TimeFromUnix` when the returned timestamp reaches the field and the paired error is rejected before publication, including `err != nil` fail-closed branches and `err == nil` guarded publication. It ignores tests, generated files, non-publication timestamps, local wall-clock round trips such as `time.Unix(time.Now().Unix(), 0)`, typed `time.Time` inputs from parsers, deserialization/rehydration, and unrelated same-named `MessagePublication` types outside `node/pkg/common`.

## Limitations

This rule currently checks `Timestamp:` keyed composite literals only. Post-construction field assignments such as `msg.Timestamp = time.Unix(...)` are an honest unsupported future scope; the known IBC-style field-mutation candidate should be modeled separately before this contract expands. The rule also avoids broad semantic source classification for every RPC/event field and does not handle deep interfaces, reflection, function values, or generic helper factories.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/message-publication-canonical-timestamp.md): policy, exact narrowed scope, pinned positives, test matrix, and calibration.
- [Final review/fix report](../../../.codeql-lint-builder/runs/message-publication-canonical-timestamp/08-return-fix-2026-07-15.md): final scope narrowing, type-identity fix, and test results.
- [Rule query](../../src/message-publication-canonical-timestamp.ql): composite-literal sink, `time.Unix`, `vaa.TimeFromUnix`, and error-guard predicates.
- [Rule fixtures](../../test/message-publication-canonical-timestamp/): direct conversions, unchecked helper errors, fail-closed guards, wall-clock exclusions, unrelated-type exclusion, and unsupported boundaries.

The rule artifact cites Wormhole source plus timestamp hardening commits `5e0920b281e7e57ea53857f7c8e23e3134505149` and `f4f745660`, but this checkout does not contain the Wormhole repository, so this page cannot provide verified version-pinned links to those files.

## Maintainer Notes

The CodeQL ID is `wormhole/go/message-publication-canonical-timestamp`. Preserve the resolved type check for `node/pkg/common.MessagePublication`; selector-name-only matching caused a known false-positive risk. If field-assignment support is added, add dedicated probes, fixtures, and recalibration rather than silently changing this composite-literal-only rule.
