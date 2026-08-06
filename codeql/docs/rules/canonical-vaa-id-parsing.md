# Canonical VAA ID Parsing

Serialized Wormhole VAA IDs in the form `chain/address/sequence` must be parsed with the canonical VAA-ID parser before constructing storage or lookup identities.

## Why This Matters

VAA IDs are used to decide whether a signed VAA already exists and to drive repair, reobservation, admin, and storage flows. Manual splitting can disagree with the canonical parser on tuple shape, chain width, sequence width, and especially emitter-address decoding. The preserved regression is an admin RPC path that split a VAA key and built `vaa.Address([]byte(parts[1]))`, treating hex text as ASCII bytes and therefore looking up a different VAA ID than the one stored canonically.

## Examples

### Violation

```go
func hasVAA(db database, vaaKey string) (bool, error) {
	parts := strings.Split(vaaKey, "/")
	chain, err := strconv.ParseUint(parts[0], 10, 16)
	if err != nil {
		return false, err
	}
	seq, err := strconv.ParseUint(parts[2], 10, 64)
	if err != nil {
		return false, err
	}

	id := guardianDB.VAAID{
		EmitterChain:   vaa.ChainID(chain),
		EmitterAddress: vaa.Address([]byte(parts[1])),
		Sequence:       seq,
	}
	return db.HasVAA(id)
}
```

### Fix

```go
func hasVAA(db database, vaaKey string) (bool, error) {
	id, err := guardianDB.VaaIDFromString(vaaKey)
	if err != nil {
		return false, err
	}
	return db.HasVAA(*id)
}
```

## What The Rule Checks

The rule reports production Go under `node/` when a slash split of a serialized VAA/message ID flows into a `VAAID` `EmitterAddress` field instead of routing the complete ID string through the canonical parser. It covers `VaaIDFromString` in `node/pkg/db` / `pkg/db` and future `VAAIDFromString` in `sdk/vaa` as canonical parsers, excludes their bodies, and recognizes both direct `vaa.Address([]byte(parts[1]))` reconstruction and component-level `vaa.StringToAddress(parts[1])` followed by manual tuple assembly.

Fix an alert by parsing the complete serialized ID with `db.VaaIDFromString` on the pinned DB-backed revision, or `vaa.VAAIDFromString` on SDK-migrated revisions, and by handling parse errors before using the identity. Component-level parsing is not a substitute for the whole-ID parser.

The exact checked scope is production Go files under `node/`, excluding `*_test.go`, `*.pb.go`, and CodeQL generated files. Current fixtures cover the admin RPC ASCII-vs-hex regression, renamed local aliases, helper bypasses, ignored canonical parser results followed by manual reconstruction, canonical parser use, typed-field construction, string production, txverifier near misses, CLI near misses, tests, and generated files.

## Limitations

This query is intentionally narrow. It does not try to prove every semantic VAA-ID string by name; it requires local split-component flow into `VAAID.EmitterAddress`. Deep interprocedural parsing, reflection, or opaque helper libraries can be missed. Parsing a package-local non-storage tuple, producing a VAA-ID string from typed fields, splitting CLI arguments into request fields, or parsing a standalone address belongs outside this rule. Standalone address normalization is owned by `canonical-vaa-address-parsing`; full `chain/address/sequence` reconstruction is owned here.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/canonical-vaa-id-parsing.md): records the standard, architecture boundary, test matrix, calibration evidence, and ownership decisions.
- [Acceptance review](../../../.codeql-lint-builder/runs/canonical-vaa-id-parsing/08-review-and-learn-acceptance-2026-07-14.md): records the final accepted state after tightening parser identity and duplicate-report boundaries.
- [Rule query](../../src/canonical-vaa-id-parsing.ql): implements the production scope, canonical-parser recognition, split-flow model, and exclusions.
- [Rule fixtures](../../test/canonical-vaa-id-parsing/): encode positives, canonical fixes, near misses, generated/test exclusions, and parser-alias regressions.
- [Calibration results](../../../.codeql-lint-builder/results/canonical-vaa-id-parsing/workflow07-20260714.csv): records the representative production result reviewed during calibration.

The rule artifact cites Wormhole source revisions and migration commits, but this checkout does not contain the Wormhole repository source, so this page links only to local rule artifacts, query code, fixtures, and run reports.

## Maintainer Notes

The CodeQL ID is `wormhole/go/canonical-vaa-id-parsing`. Preserve this rule's precedence over address parsing for complete serialized VAA IDs. Update the query and fixtures together if the canonical parser moves, if `VAAID` type names or package paths change, or if production introduces a reviewed parser wrapper that delegates completely to the canonical parser.
