# Canonical VAA Address Parsing

External Wormhole address data must be normalized with `vaa.StringToAddress` or `vaa.BytesToAddress` before it is used as a `vaa.Address` protocol identity.

## Why This Matters

`vaa.Address` is Wormhole's 32-byte identity form for emitters, VAA IDs, publications, Governor/Accountant state, delegated observations, relayer keys, and lookup/storage tuples. Manual casts, `hex.DecodeString` plus `copy`, or package-local address parsers can disagree with SDK behavior for left-padding, optional `0x`, overlength rejection, and ASCII-versus-hex handling.

## Examples

### Violation

```go
func publication(emitter string) common.MessagePublication {
	decoded, _ := hex.DecodeString(emitter)
	addr := vaa.Address{}
	copy(addr[:], decoded)
	return common.MessagePublication{EmitterAddress: addr}
}
```

### Fix

```go
func publication(emitter string) (common.MessagePublication, error) {
	addr, err := vaa.StringToAddress(emitter)
	if err != nil {
		return common.MessagePublication{}, err
	}
	return common.MessagePublication{EmitterAddress: addr}, nil
}
```

For raw bytes, use `vaa.BytesToAddress` and handle the error before constructing the identity sink.

## What The Rule Checks

The rule reports non-canonical `vaa.Address` construction that reaches Wormhole identity sinks. It covers direct `vaa.Address(...)` conversions from byte-like data, `copy` into a `vaa.Address`, and unsafe helper calls whose result flows to identity fields or returns. Identity sinks include `MessagePublication`, `VAA`, `VAAID`, token bridge keys, `EmitterAddress`, `emitterAddr`, `targetAddress`, and `vaa.Address` returns used by those paths.

The exact checked scope is production Go under `node/` or `pkg/`, excluding tests and CodeQL generated files. The query recognizes canonical `vaa.StringToAddress` and `vaa.BytesToAddress` results, safe thin wrappers that delegate to those calls, already-typed `VAA` / `MessagePublication` / `VAAID` propagation, and explicit reviewed adapter exceptions. Current explicit exceptions include EVM `PadAddress(common.Address)`, Aptos `uint64` sender encoding, NEAR exact-32 emitter digest copy, XRPL account helpers, Sui typed array conversion, compile-time known-tokenbridge synthetic notary construction, and the narrow public RPC `GetSignedVAA` `MessageId.EmitterAddress` exact-32 lookup path.

Fix an alert by routing string input through `vaa.StringToAddress`, byte input through `vaa.BytesToAddress`, or a whole serialized VAA ID through the canonical whole-ID parser owned by `canonical-vaa-id-parsing`. A string helper that decodes locally and then calls `BytesToAddress` is still intentionally reported because it changes string parsing policy.

## Limitations

The query is conservative about source provenance. It reports the current CosmWasm local `StringToAddress` copy because the accepted query cannot soundly prove the authenticated core-event producer path. It keeps a full-VAA-ID address-component result as overlap evidence in calibration, but root-cause ownership for complete `chain/address/sequence` parsing belongs to `canonical-vaa-id-parsing`. Hashes, transaction IDs, `common.Hash`, chain-ID parsing, tests, generated files, and already-typed internal address propagation are out of scope.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/canonical-vaa-address-parsing.md): records the address normalization standard, exceptions, test matrix, calibration results, and ownership boundaries.
- [Final acceptance-blocker report](../../../.codeql-lint-builder/runs/canonical-vaa-address-parsing/10-final-acceptance-blockers-2026-07-15.md): documents the final public RPC and CosmWasm decisions plus retained production findings.
- [Rule query](../../src/canonical-vaa-address-parsing.ql): implements unsafe conversions, copy sinks, helper summaries, canonical sanitizer recognition, and explicit exceptions.
- [Rule fixtures](../../test/canonical-vaa-address-parsing/): encode direct conversions, manual copies, unsafe wrappers, canonical wrappers, typed propagation, whole-ID ownership, hash exclusions, and chain-adapter exceptions.
- [Calibration SARIF](../../../calibration/canonical-vaa-address-parsing/results.sarif): stores the latest production calibration results referenced by the acceptance report.

The rule artifact cites Wormhole source revisions and examples, but this checkout does not contain the Wormhole repository source, so this page links only to local artifacts, query code, fixtures, and calibration outputs.

## Maintainer Notes

The CodeQL ID is `wormhole/go/canonical-vaa-address-parsing`. Add new chain-specific adapter exceptions only after documenting source type, width, and semantics in fixtures and the rule artifact. Keep duplicate-report boundaries with `canonical-vaa-id-parsing` explicit: complete serialized VAA IDs are whole-ID parser issues; standalone address components are address-normalization issues.
