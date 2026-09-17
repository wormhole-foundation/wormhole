# XRPL Derived Generated Emitter

Use the family-specific domain-separated emitter when the XRPL watcher synthesizes XTCF, XACK, or NTT `MessagePublication` values.

## Why This Matters

XRPL watcher-generated messages do not come from an on-chain Core bridge emitter. Their `EmitterAddress` becomes part of the VAA identity consumed by downstream protocols, so generated XRPL message families need their own namespaces. XTCF and XACK must use the generated managed-account emitter with the nonzero `"XRPL"` prefix. NTT must use `keccak256("ntt" + source manager + source token)`. Raw account emitters are reserved for generic XRPL Core payments and can collide with, or be confused for, generated-message domains.

## Examples

### Violation

```go
func parseTicketCreateTransaction(account string) *MessagePublication {
	return &MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: addressToEmitter(account), // raw Core-style emitter
		Payload:        buildXTCFPayload(),
	}
}
```

### Fix

```go
func parseTicketCreateTransaction(account string) *MessagePublication {
	return &MessagePublication{
		EmitterChain:   vaa.ChainIDXRPL,
		EmitterAddress: calculateGeneratedEmitterAddress(account),
		Payload:        buildXTCFPayload(),
	}
}
```

For NTT, use the approved NTT helper that hashes the exact `"ntt"`, source-manager, and source-token sequence.

## What The Rule Checks

The rule reports production `MessagePublication` composite literals under `node/pkg/watchers/xrpl/` whose `EmitterChain` is `ChainIDXRPL`, whose enclosing function matches the current XTCF, XACK, or NTT parser shapes, and whose `EmitterAddress` cannot be proven to use the approved family derivation.

It recognizes XTCF/XACK functions by `xtcfPrefix` or `xackPrefix` use and NTT by `parseNttTransaction` calling `buildNTTPayload`. For XTCF/XACK, the approved helper must seed from `addressToEmitter` and overlay `generatedEmitterPrefix` into bytes `[0:4]` without other writes/copies to the returned emitter. For NTT, the approved helper must return bytes derived from `Keccak256` over a buffer containing `"ntt"`, `sourceNTTManager`, and `sourceToken` in the modeled slice positions.

The rule ignores tests, generated files, non-XRPL watchers, non-XRPL publications, payload-only helpers, routing code, and the generic XRPL Core raw-emitter path.

## Limitations

Family classification is tied to the current XRPL parser shapes and helper names. The query does not prove arbitrary byte-level equivalence, interface dispatch, reflection, global mutable emitters, deeply indirect helper factories, or future generated XRPL families. It conservatively rejects multiple assignments to the local emitter value and generated-emitter helpers that perform extra returned-emitter writes. A future compliant helper with a different shape may need query and fixture updates.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/xrpl-derived-generated-emitter.md): records the XRPL emitter standard, historical fixes, model, tests, and calibration evidence.
- [Acceptance review](../../../.codeql-lint-builder/runs/xrpl-derived-generated-emitter/08-review-and-learn-acceptance-2026-07-14.md): records the final helper-binding, assignment-order, overwrite, and acceptance gate.
- [Rule query](../../src/xrpl-derived-generated-emitter.ql): defines family recognition, approved helper internals, report locations, and scope.
- [Rule fixtures](../../test/xrpl-derived-generated-emitter/): encode raw emitters, wrong NTT derivations, bad layouts, overwrites, generated/test exclusions, and near misses.

The rule artifact cites Wormhole source, README sections, regression tests, and historical fix commits, but this checkout does not contain the Wormhole repository, so this page cannot provide verified version-pinned links to those files.

## Maintainer Notes

The CodeQL ID is `wormhole/go/xrpl-derived-generated-emitter`. Update the query and fixtures together if XRPL parser function names, payload-prefix signals, generated-emitter helper layout, NTT derivation helper internals, `MessagePublication` construction style, or watcher paths change.
