#!/usr/bin/env bash

# -e is not set here so that we can see the results of many separate
# clippy runs, even if some of them have errors.
set -uo pipefail

# Required to correctly build SVM crates. We can use dummy values for the purposes of linting
export BRIDGE_ADDRESS=""
export EMITTER_ADDRESS=""
export CHAIN_ID="1"
export VERIFY_VAA_SHIM_PROGRAM_ID="abc"
export POST_MESSAGE_SHIM_PROGRAM_ID="def"
export TOKEN_BRIDGE_ADDRESS="ghi"

# Go to repo root (in case script is called from another dir)
# NOTE: Assumes this script is under `scripts/`
cd "$(dirname "$0")/.." || exit

# This should be the latest available Rust version for the system in order to match
# our CI configuration.
echo "Running Rust linting with active toolchain: $(rustup show active-toolchain)"

# Skip:
# - 3rd party crates
# - Compiled code
# - Version control information
# - Deprecated chains (e.g. Terra)
#
# TODO: Near, CosmWasm/Wormchain, NFT Bridge, and memmap2-rs should probably be linted as well
# but requires a little more work.
find . -type f -name Cargo.toml \
   -not -path "./target/*" \
   -not -path "./node_modules/*" \
   -not -path "./.cargo/*" \
   -not -path "$HOME/.cargo/*" \
   -not -path "./.git/*" \
   -not -path "./terra/*" \
   -not -path "./clients/*" \
   -not -path "./cosmwasm/*" \
   -not -path "*/nft_bridge/*" \
   -not -path "*/memmap2-rs/*" \
   -not -path "./near/*" \
   -not -path "./wormchain/interchaintest/contracts/ibc_hooks/*" \
| while IFS= read -r manifest; do
    crate_dir=$(dirname "$manifest")
    echo "→ Linting crate in: $crate_dir"
    (
      cd "$crate_dir" || exit
      # Extract rust version from rust-toolchain files, if present.
      # This is just for logging purposes: rustup will automatically pick the correct toolchain.
      rust_version=""
      if [ -f "rust-toolchain" ]; then
        rust_version=$(grep -E 'channel\s*=' rust-toolchain | sed -E 's/.*=\s*"([^"]+)".*/\1/')
      elif [ -f "rust-toolchain.toml" ]; then
        # Parse rust version (supports both [toolchain] and TOML key forms)
        rust_version=$(grep -E 'channel\s*=' rust-toolchain.toml | sed -E 's/.*=\s*"([^"]+)".*/\1/')
      fi

      if [ ! -z "$rust_version" ]; then
        echo "   Using crate’s defined toolchain: $rust_version"
      fi
      
      if [[ "$(pwd)" == *"svm/wormhole-core-shims"* ]]; then
        # The Shims code has mutually-exclusive features that prevent compilation.
        # Instead of all-features, choose solana specifically.
        cargo clippy --quiet --all-targets --workspace --locked -- \
          -D warnings
      else 
        cargo clippy --quiet --all-targets --all-features --workspace --locked -- \
          -D warnings
      fi
      
    )
done

echo "✅ Rust linting completed."
