# Wormhole Rust SDK

Rust implementation of Wormhole core types and utilities.

## Packages

- **`supported-chains`** - Chain ID definitions and conversions
- **`vaas-serde`** - VAA parsing and serialization
- **`serde_wormhole`** - Serde data format for Wormhole payloads

## Code Generation

The `supported-chains` crate auto-generates the `Chain` enum and trait implementations from the Go SDK source of truth at compile time.

**Important:** The Rust SDK depends on `sdk/vaa/structs.go`. Chain names, formatting, and IDs are extracted directly from Go constant definitions. When modifying the Go SDK, preserve the existing format of chain constant declarations.

### How It Works

`supported-chains/build.rs` parses `sdk/vaa/structs.go` and generates:
- Chain enum variants (preserving Go name casing)
- `From<u16>` and `From<Chain> for u16` conversions
- `Display` and `FromStr` implementations

The generated code includes all active chains and marks obsolete chain IDs as comments.

The approach simply writes raw strings out to a Rust file rather than use more complicated AST mechanisms.
However, the Rust compiler will include this file during compilation so we benefit from all the usual protections this offers.
The generation is also supplemented by a linting script under the repo's `scripts/` directory which checks
for parity between the Go and Rust SDKs. This is technically redundant but should help catch potential problems
in `build.rs`.

### Updating Chains

1. Update `sdk/vaa/structs.go` (the canonical source)
2. Rebuild the Rust SDK - code regenerates automatically

No manual synchronization needed.

### Breaking Changes

Version 1.0.0 introduces breaking changes:
- Removed obsolete chain variants (Oasis, Aurora, Karura, Acala, Neon, Xpla)
- Chain names now match Go SDK exactly (e.g., `BSC` not `Bsc`, `PythNet` not `Pythnet`)
- Obsolete chain IDs map to `Chain::Unknown(n)`

### Testing

```bash
cd sdk/rust/supported-chains
cargo test
```

To verify sync with Go SDK:
```bash
./scripts/check-rust-chain-sync.sh
```

To view generated code:
```bash
cargo build
cat target/debug/build/wormhole-supported-chains-*/out/chains_generated.rs
```

### Build Script Behavior

- Reruns when `sdk/vaa/structs.go` changes
- Parses patterns like:
  - `ChainIDSolana ChainID = 1` → Active chain
  - `// OBSOLETE: ChainIDOasis ChainID = 7` → Maps to `Chain::Unknown(7)`
  - `// WARNING: ChainIDTerra...` → Includes with doc comment
- Preserves exact casing from Go (e.g., `ChainIDBSC` → `Chain::BSC`)
