# Canonical Chain ID Parsing

Boundary-derived Wormhole chain IDs must be converted with the SDK chain-ID helpers instead of local range checks and direct `vaa.ChainID(...)` casts.

## Why This Matters

Wormhole chain IDs are `uint16` wire values, but many uses also require SDK-registered-chain semantics. Local conversions blur those two decisions and duplicate validation in admin RPC, public RPC, governance, watcher, IBC, manager, Governor, Accountant, and txverifier paths. The SDK helpers make the intended boundary explicit: wire-valid, registered numeric, or registered string.

## Examples

### Violation

```go
func submit(req *proto.GovernanceRequest) BodyContractUpgrade {
	if req.ChainId > math.MaxUint16 {
		return BodyContractUpgrade{}
	}
	return BodyContractUpgrade{
		TargetChainID: vaa.ChainID(req.ChainId),
	}
}
```

### Fix

```go
func submit(req *proto.GovernanceRequest) (BodyContractUpgrade, error) {
	chain, err := vaa.ChainIDFromNumber[uint32](req.ChainId)
	if err != nil {
		return BodyContractUpgrade{}, err
	}
	return BodyContractUpgrade{TargetChainID: chain}, nil
}
```

Use `vaa.KnownChainIDFromNumber` instead when the value selects a local watcher, reobserver, chain-specific policy, or supported-chain map. Use `vaa.StringToKnownChainID` for string configuration or text input that must name an SDK-known chain.

## What The Rule Checks

The rule reports direct conversions to SDK `vaa.ChainID` when a modeled boundary source flows into a chain-ID use context without first passing through `vaa.ChainIDFromNumber`, `vaa.KnownChainIDFromNumber`, or `vaa.StringToKnownChainID`. Reported contexts include struct fields, call arguments, indexes, comparisons, and assignments.

The exact checked scope is production Go under `node/`, `pkg/`, or `cmd/` path shapes used by Wormhole CodeQL databases, excluding tests, protobuf/gRPC generated files, generated ABI bindings, and CodeQL generated files. Modeled sources are intentionally bounded: generated Wormhole protobuf field/getter reads whose base comes from a parameter or channel receive, IBC `WasmAttributes.GetAsUint("message.chain_id", 16)`, IBC `ChannelChains` JSON entries at index `1`, and txverifier parameter-backed range values. The query also recognizes both `github.com/wormhole-foundation/wormhole/sdk/vaa` and `github.com/certusone/wormhole/sdk/vaa` import paths.

Fix an alert by choosing the helper that matches the surrounding semantics. Do not blindly replace every alert with known-chain validation: governance payloads, historical VAA lookup, peer/version-skew data, and other wire-compatible contexts may need `ChainIDFromNumber` rather than `KnownChainIDFromNumber`.

## Limitations

The query does not prove arbitrary external provenance. Its IBC and JSON coverage is a shape-based model tied to concrete production patterns and same-file near-miss fixtures. It does not cover broad string parsing, generic HTTP/config/database sources, interprocedural wrapper helpers, full serialized `chain/address/sequence` parsing owned by `canonical-vaa-id-parsing`, or chain-native EVM `chainId` before it is mapped to a Wormhole chain. Already-typed `vaa.ChainID` values, SDK constants, non-chain-ID numeric fields, SDK wire serialization internals, tests, and generated code are out of scope.

## Learn More

- [Rule contract](../../../.codeql-lint-builder/rules/canonical-chain-id-parsing.md): records helper semantics, source model, calibration evidence, and the wire-valid versus registered-chain distinction.
- [Second-gate acceptance report](../../../.codeql-lint-builder/runs/canonical-chain-id-parsing/09-second-gate-return-2026-07-15.md): documents the final bounded IBC/JSON model and validation results.
- [Rule query](../../src/canonical-chain-id-parsing.ql): defines production scope, boundary sources, SDK-helper recognition, and direct-cast sinks.
- [Rule fixtures](../../test/canonical-chain-id-parsing/): encode protobuf, IBC, JSON, txverifier, helper, internal-value, and near-miss cases.
- [Calibration SARIF](../../../.codeql-lint-builder/runs/canonical-chain-id-parsing/07-calibrate-2026-07-14.sarif): stores representative production findings used during calibration.

The rule artifact cites Wormhole source revisions and helper-introduction commits, but this checkout does not contain the Wormhole repository source, so this page links only to local artifacts and run evidence.

## Maintainer Notes

The CodeQL ID is `wormhole/go/canonical-chain-id-parsing`. Keep the source model evidence-based; do not broaden by variable names alone. When adding a new boundary source, add production evidence plus negative fixtures for same-file lookalikes. Maintainers reviewing findings should classify the intended remediation as wire-valid or registered-chain before proposing a fix.
