#!/usr/bin/env bash
# Deploy script for Stellar Wormhole contract
# Usage: deploy.sh <testnet|mainnet> [--skip-init] [--yes]

set -euo pipefail

# Helper functions
die() { echo "deploy.sh: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "missing '$1'"; }
step() { echo "==> $*" >&2; }
usage() {
  cat >&2 <<EOF
Usage: $(basename "$0") <testnet|mainnet> [OPTIONS]

Deploy the Stellar Wormhole contract to the specified network.

Arguments:
  testnet|mainnet    Target network for deployment

Options:
  --skip-init        Skip contract initialization (useful when deploying to an
                     already-initialized contract or when config has no guardians)
  --yes              Skip confirmation prompt (required for mainnet deployments)
  -h, --help         Show this help message

Requirements:
  - stellar CLI tool must be installed and in PATH
  - yq must be installed (YAML processor)
  - curl must be installed
  - Configuration file must exist at: $CONFIG_DIR/<network>.yaml

Configuration:
  The script reads configuration from YAML files in $CONFIG_DIR/:
  - testnet.yaml: Configuration for testnet deployment
  - mainnet.yaml: Configuration for mainnet deployment

  Required fields in config:
  - deployer: Stellar keys identity name
  - friendbot_url: URL for funding testnet accounts (can be empty for mainnet)
  - governance_emitter: 32-byte hex address of the governance emitter
                        (standard is 0000...0004)
  - guardians: Array of guardian addresses (can be empty if using --skip-init)

Examples:
  # Deploy to testnet
  $(basename "$0") testnet

  # Deploy to testnet without initialization
  $(basename "$0") testnet --skip-init

  # Deploy to mainnet (requires --yes flag)
  $(basename "$0") mainnet --yes

Output:
  The script outputs deployment information as JSON, including:
  - network: Target network name
  - contract_id: Deployed contract ID
  - wasm_hash: Hash of the deployed WASM
  - deployer: Deployer account address
  - guardian_count: Number of guardians initialized
  - initialized: Whether the contract was initialized
  - timestamp: Deployment timestamp (UTC)
EOF
  exit 2
}

# Set up directory paths
# SCRIPT_DIR: absolute path to the directory containing this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# STELLAR_ROOT: absolute path to the stellar directory (parent of scripts/)
STELLAR_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
CONFIG_DIR="$SCRIPT_DIR/config"

# Parse command-line arguments
# NETWORK: first positional argument (testnet or mainnet)
NETWORK="${1:-}"; shift || true
SKIP_INIT=false
YES=false

# Process remaining command-line flags
while [[ $# -gt 0 ]]; do
  case "$1" in
    --skip-init) SKIP_INIT=true ;;  # Skip contract initialization
    --yes) YES=true ;;              # Skip confirmation prompt (required for mainnet)
    -h|--help) usage ;;
    *) die "unknown option: $1" ;;
  esac
  shift
done

# Validate inputs and check dependencies
[[ -n "$NETWORK" ]] || usage
need stellar  # Stellar CLI tool
need yq       # YAML processor
need curl     # HTTP client

# Load configuration from YAML file
CONFIG_FILE="$CONFIG_DIR/$NETWORK.yaml"
[[ -f "$CONFIG_FILE" ]] || die "unknown network '$NETWORK' (missing $CONFIG_FILE)"

# Extract configuration values from YAML
DEPLOYER="$(yq -r '.deployer' "$CONFIG_FILE")"                    # Stellar keys identity name
FRIENDBOT_URL="$(yq -r '.friendbot_url' "$CONFIG_FILE")"          # Friendbot URL for funding (testnet only)
GUARDIANS_JSON="$(yq -o=json '.guardians' "$CONFIG_FILE")"        # Guardian addresses as JSON array
GUARDIAN_COUNT="$(yq -r '.guardians | length' "$CONFIG_FILE")"    # Number of guardians
GOVERNANCE_EMITTER="$(yq -r '.governance_emitter' "$CONFIG_FILE")" # Governance emitter address (32 bytes hex)

# Validate configuration
[[ -n "$DEPLOYER" && "$DEPLOYER" != "null" ]] || die "config missing .deployer: $CONFIG_FILE"
# Require governance_emitter unless --skip-init is used
[[ "$SKIP_INIT" == "true" || ( -n "$GOVERNANCE_EMITTER" && "$GOVERNANCE_EMITTER" != "null" ) ]] || die "config missing .governance_emitter: $CONFIG_FILE"
# Require guardians unless --skip-init is used
[[ "$SKIP_INIT" == "true" || "$GUARDIAN_COUNT" -gt 0 ]] || die "config has no guardians (use --skip-init)"
# Require --yes flag for mainnet deployments
[[ "$NETWORK" != "mainnet" || "$YES" == "true" ]] || die "mainnet requires --yes"

# Ensure deployer key exists, generate if needed
stellar keys address "$DEPLOYER" >/dev/null 2>&1 || stellar keys generate "$DEPLOYER" --network "$NETWORK" >/dev/null 2>&1
DEPLOYER_ADDR="$(stellar keys address "$DEPLOYER")"

# Fund deployer account using friendbot (testnet only)
if [[ -n "$FRIENDBOT_URL" && "$FRIENDBOT_URL" != "null" ]]; then
  step "fund deployer (friendbot)"
  curl -sSf --max-time 30 "$FRIENDBOT_URL?addr=$DEPLOYER_ADDR" >/dev/null 2>&1 || true
  sleep 2
fi

# Set up contract paths
CONTRACT_DIR="$STELLAR_ROOT/contracts/wormhole-contract"
CONTRACT_WASM="$STELLAR_ROOT/target/wasm32v1-none/release/wormhole_contract.wasm"

# Build the contract
step "build"
(cd "$CONTRACT_DIR" && stellar contract build --optimize >/dev/null)
[[ -f "$CONTRACT_WASM" ]] || die "missing wasm: $CONTRACT_WASM"

# Deploy the contract
step "deploy ($NETWORK)"
DEPLOY_OUTPUT="$(stellar contract deploy --wasm "$CONTRACT_WASM" --source-account "$DEPLOYER" --network "$NETWORK" 2>&1)"
# Extract contract ID from deployment output (Stellar contract IDs are 56 chars starting with 'C')
CONTRACT_ID="$(grep -Eo 'C[A-Z0-9]{55}' <<<"$DEPLOY_OUTPUT" | head -1 || true)"
[[ -n "$CONTRACT_ID" ]] || { echo "$DEPLOY_OUTPUT" >&2; die "failed to parse contract id"; }
# Extract WASM hash from deployment output (64 hex characters)
WASM_HASH="$(grep -Eo '[a-f0-9]{64}' <<<"$DEPLOY_OUTPUT" | head -1 || true)"

# Initialize the contract with guardians (unless --skip-init is used)
if [[ "$SKIP_INIT" != "true" ]]; then
  step "initialize ($GUARDIAN_COUNT guardians)"
  stellar contract invoke --id "$CONTRACT_ID" --source-account "$DEPLOYER" --network "$NETWORK" -- \
    initialize --initial_guardians "$GUARDIANS_JSON" --governance_emitter "\"$GOVERNANCE_EMITTER\"" >/dev/null
  # Verify initialization succeeded by checking guardian set index is 0
  [[ "$(stellar contract invoke --id "$CONTRACT_ID" --source-account "$DEPLOYER" --network "$NETWORK" -- \
    get_current_guardian_set_index 2>/dev/null | grep -Eo '[0-9]+' | head -1)" == "0" ]] \
    || die "init failed"
fi

# Output deployment summary
TIMESTAMP="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
INITIALIZED="$([[ "$SKIP_INIT" == "true" ]] && echo "no" || echo "yes")"

# Human-readable output to stderr (for display)
{
  echo ""
  echo "┌─────────────────────────────────────────────────────────────────┐"
  echo "│                    ✅ DEPLOYMENT SUCCESSFUL                     │"
  echo "└─────────────────────────────────────────────────────────────────┘"
  echo ""
  echo "  Network:        $NETWORK"
  echo "  Contract ID:    $CONTRACT_ID"
  echo "  WASM Hash:      ${WASM_HASH:-N/A}"
  echo "  Deployer:       $DEPLOYER_ADDR"
  echo "  Guardians:      $GUARDIAN_COUNT"
  echo "  Initialized:    $INITIALIZED"
  echo "  Timestamp:      $TIMESTAMP"
  echo ""
} >&2

# JSON output to stdout (for scripting)
printf '{"network":"%s","contract_id":"%s","wasm_hash":"%s","deployer":"%s","guardian_count":%s,"initialized":%s,"timestamp":"%s"}\n' \
  "$NETWORK" "$CONTRACT_ID" "${WASM_HASH:-}" "$DEPLOYER_ADDR" "$GUARDIAN_COUNT" \
  "$([[ "$SKIP_INIT" == "true" ]] && echo false || echo true)" "$TIMESTAMP"
