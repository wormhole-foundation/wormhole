#!/usr/bin/env bash
# This is a safety check for the build.rs code generation for the Rust SDK.
# More information on this process can be found in the READMEs for the Go and Rust SDKs.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

# The Go SDK is the ultimate source of truth
GO_SOURCE="$REPO_ROOT/sdk/vaa/structs.go"
RUST_WORKSPACE="$REPO_ROOT/sdk/rust"

echo "Checking Rust SDK chain synchronization..."

# Extract active chain IDs from Go source
# Matches: ChainIDSolana ChainID = 1
# Skips: // OBSOLETE: ChainIDOasis ChainID = 7
# Skips: ChainIDUnset (we use Chain::Any for 0)
go_chains=$(grep -E '^\s+ChainID[A-Za-z0-9]+\s+ChainID\s+=\s+[0-9]+' "$GO_SOURCE" | \
    grep -v 'ChainIDUnset' | \
    awk '{print $4}' | \
    sort -n)

echo "Found $(echo "$go_chains" | wc -l) active chains in Go SDK"

# Build the Rust crate to trigger code generation
echo "Building Rust SDK to generate code..."
cd "$RUST_WORKSPACE"
cargo build --package wormhole-supported-chains --quiet 2>/dev/null || {
    echo "❌ Failed to build Rust SDK"
    exit 1
}

# Find the generated file (most recent one)
generated_file=$(find "$RUST_WORKSPACE/target" -path "*/wormhole-supported-chains-*/out/chains_generated.rs" 2>/dev/null | sort | tail -1)

if [ ! -f "$generated_file" ]; then
    echo "❌ Could not find generated chains_generated.rs"
    echo "   Expected in: target/debug/build/wormhole-supported-chains-*/out/"
    exit 1
fi

echo "Found generated file: $(basename $(dirname $generated_file))/out/chains_generated.rs"

# Extract chain IDs from generated Rust code
# Matches: 1 => Chain::Solana,
# Skip 0 => Chain::Any (handled specially, not from Go source)
rust_chains=$(grep -E '^\s+[0-9]+\s+=>\s+Chain::' "$generated_file" | \
    awk '{print $1}' | \
    grep -v '^0$' | \
    sort -n)

echo "Found $(echo "$rust_chains" | wc -l) active chains in Rust generated code"

# Compare the lists
missing_in_rust=""
extra_in_rust=""

for id in $go_chains; do
    if ! echo "$rust_chains" | grep -q "^${id}$"; then
        missing_in_rust="$missing_in_rust $id"
    fi
done

for id in $rust_chains; do
    if ! echo "$go_chains" | grep -q "^${id}$"; then
        extra_in_rust="$extra_in_rust $id"
    fi
done

# Report results
if [ -n "$missing_in_rust" ]; then
    echo "❌ MISSING in Rust SDK:"
    for id in $missing_in_rust; do
        chain_name=$(grep -E "ChainID\w+\s+ChainID\s+=\s+${id}" "$GO_SOURCE" | awk '{print $1}')
        echo "   Chain ID $id ($chain_name)"
    done
    exit 1
fi

if [ -n "$extra_in_rust" ]; then
    echo "⚠️  WARNING: Extra chains in Rust SDK (not in Go):"
    for id in $extra_in_rust; do
        echo "   Chain ID $id"
    done
    echo "   This might indicate a parsing bug in build.rs"
    exit 1
fi

echo "✅ All Go SDK chains are present in Rust generated code!"
echo ""
echo "Summary:"
echo "  Go SDK:   $(echo "$go_chains" | wc -l) chains"
echo "  Rust SDK: $(echo "$rust_chains" | wc -l) chains"
echo "  Status:   Synchronized ✓"
