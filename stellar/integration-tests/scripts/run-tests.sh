#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
ENV_FILE="$SCRIPT_DIR/../.env.localnet"

die() {
  echo "$*" >&2
  exit 1
}

cleanup() {
  if [[ -n "${LIB_RS:-}" && -f "${LIB_RS}.bak" ]]; then
    mv "${LIB_RS}.bak" "$LIB_RS"
  fi
}

trap cleanup EXIT

[ -f "$ENV_FILE" ] || die "Environment file $ENV_FILE not found"

# Load the localnet configuration consumed by both the shell wrapper and the
# Rust integration test harness.
# shellcheck disable=SC1091
source "$ENV_FILE"

: "${STELLAR_NETWORK:?STELLAR_NETWORK not set}"
: "${STELLAR_IDENTITY:?STELLAR_IDENTITY not set}"
: "${SOROBAN_RPC_URL:?SOROBAN_RPC_URL not set}"
: "${WORMHOLE_WASM_PATH:?WORMHOLE_WASM_PATH not set}"

if [[ "$WORMHOLE_WASM_PATH" != /* ]]; then
  # Resolve the configured wasm path relative to the workspace root.
  WORMHOLE_WASM_PATH="$PROJECT_ROOT/$WORMHOLE_WASM_PATH"
fi

# The Rust tests read these values via `std::env`, so they must be exported
# before `cargo test` launches the test binaries.
export STELLAR_NETWORK
export STELLAR_IDENTITY
export SOROBAN_RPC_URL
export WORMHOLE_WASM_PATH

echo "Building contracts using stellar contract build..."
cd "$PROJECT_ROOT"
rm -f target/wasm32v1-none/release/wormhole_contract.wasm
stellar contract build
# Keep a copy of the original WASM so we can restore it after preparing the
# upgrade artifact used by the contract-upgrade integration test.
cp target/wasm32v1-none/release/wormhole_contract.wasm target/wasm32v1-none/release/wormhole_contract_original.wasm

# Build a version for upgrade test, with different chain ID
echo "Building upgraded contract version for integration tests..."
LIB_RS="contracts/wormhole-contract/src/lib.rs"
sed -i.bak 's/u32::from(CHAIN_ID_STELLAR)/999u32/' "$LIB_RS"
stellar contract build
cp target/wasm32v1-none/release/wormhole_contract.wasm target/wasm32v1-none/release/wormhole_contract_upgrade.wasm

# Restore original source and WASM file
touch "$LIB_RS"
cp target/wasm32v1-none/release/wormhole_contract_original.wasm target/wasm32v1-none/release/wormhole_contract.wasm

export WORMHOLE_UPGRADE_WASM_PATH="$PROJECT_ROOT/target/wasm32v1-none/release/wormhole_contract_upgrade.wasm"

echo "Running integration tests..."
cargo test -p integration-tests -- --ignored --nocapture
