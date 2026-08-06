# RunWithScissors Error Return

Return fatal runnable errors from `common.RunWithScissors` runnables instead of sending directly to the same wrapper error channel.

## Why This Matters

`RunWithScissors` centralizes watcher goroutine lifecycle handling: panic recovery, metrics, and nonblocking forwarding of returned errors to `errC`. A runnable that sends directly to the same `errC` bypasses that nonblocking wrapper path and can hang shutdown or cleanup when the receiver has stopped. If it also returns the same error, the failure may be delivered twice.

## Examples

### Violation

```go
common.RunWithScissors(ctx, errC, "poller", func() error {
	if err := poll(); err != nil {
		errC <- fmt.Errorf("poll failed: %w", err)
		return nil
	}
	return nil
})
```

### Fix

```go
common.RunWithScissors(ctx, errC, "poller", func() error {
	if err := poll(); err != nil {
		return fmt.Errorf("poll failed: %w", err)
	}
	return nil
})
```

## What The Rule Checks

The rule reports production Go sends under `node/` that execute in a runnable passed directly to `common.RunWithScissors(ctx, errC, name, runnable)` and target the same error channel as argument 1. It supports both `github.com/wormhole-foundation/wormhole/node/pkg/common` and `github.com/certusone/wormhole/node/pkg/common` package paths.

Supported runnable forms are inline function literals, local function values with a reaching pre-call function-literal assignment that is not overwritten, method values, receiver-method bodies that send to the same receiver field, and one synchronous helper hop where the helper receives the same channel parameter or sends through the same receiver field. The rule ignores tests, generated files, wrapper-owned forwarding inside `RunWithScissors`/`StartRunnable`/`startRunnable`, sibling reads from `errC`, unrelated channels, non-`RunWithScissors` goroutines, and helper calls launched under `go`.

## Limitations

This calibrated rule starts only from direct `common.RunWithScissors` calls. Thin lifecycle wrappers around `RunWithScissors` are intentionally deferred/unsupported, even if they appear to forward `(ctx, errC, name, runnable)` unchanged. Async helpers and goroutine boundaries are also deferred: a helper called with `go report(errC, err)` from the runnable is out of scope. Local channel aliases such as `ch := errC; ch <- err`, container-stored channels, recursive helper chains, interface dispatch, and deep interprocedural channel propagation are tolerated false negatives rather than name-matched approximations.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/run-with-scissors-error-return.md): records the lifecycle standard, direct-call-only scope, unsupported boundaries, test matrix, and calibration evidence.
- [Acceptance / return-fix report](../../../.codeql-lint-builder/runs/run-with-scissors-error-return/08-review-and-learn-return-fixes-2026-07-15.md): records async-helper exclusion, local function-value reaching-definition fixes, direct-call-only scope, and final calibration.
- [Rule query](../../src/run-with-scissors-error-return.ql): defines package paths, runnable resolution, same-channel correlation, helper modeling, and report location.
- [Rule fixtures](../../test/run-with-scissors-error-return/): encode supported positives, returned-error negatives, sibling reads, unrelated channels, unsupported aliases, async helpers, and thin-wrapper boundaries.
- [Certusone module fixtures](../../test/run-with-scissors-error-return-certusone-module/): ensure the historical module path is recognized.

The rule artifact cites Wormhole source, tests, watcher docs, and hardening commit `f04918d1b483b772727855d1711392f7575798da`, but this checkout does not contain the Wormhole repository, so this page cannot provide verified version-pinned links to them.

## Maintainer Notes

The CodeQL ID is `wormhole/go/run-with-scissors-error-return`. Keep the direct-call-only contract, async helper exclusion, and thin-wrapper deferral synchronized across query, fixtures, and docs. If thin wrappers or aliases become required, add explicit fixtures before broadening same-channel modeling.
