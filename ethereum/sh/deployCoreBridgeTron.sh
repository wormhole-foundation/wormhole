#!/bin/bash
# Deploy Wormhole Core to Tron via tronweb-style HTTP API.
# Usage: from ethereum/, copy env/.env.tron.testnet to .env, fill in TRON_PRIVATE_KEY,
# then run ./sh/deployCoreBridgeTron.sh

set -euo pipefail

set -a
. .env
set +a

[[ -z "${TRON_PRIVATE_KEY:-${MNEMONIC:-}}" ]] && { echo "Missing TRON_PRIVATE_KEY (or MNEMONIC)"; exit 1; }
[[ -z "${INIT_SIGNERS:-}" ]] && { echo "Missing INIT_SIGNERS"; exit 1; }
[[ -z "${INIT_CHAIN_ID:-}" ]] && { echo "Missing INIT_CHAIN_ID"; exit 1; }
[[ -z "${INIT_GOV_CHAIN_ID:-}" ]] && { echo "Missing INIT_GOV_CHAIN_ID"; exit 1; }
[[ -z "${INIT_GOV_CONTRACT:-}" ]] && { echo "Missing INIT_GOV_CONTRACT"; exit 1; }
[[ -z "${INIT_EVM_CHAIN_ID:-}" ]] && { echo "Missing INIT_EVM_CHAIN_ID"; exit 1; }

if [[ ! -d build-forge ]]; then
  echo "build-forge/ not found; running 'make build'..."
  make build
fi

node sh/deployCoreBridgeTron.js
