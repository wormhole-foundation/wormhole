#!/usr/bin/env bash
# =============================================================================
# Wormhole Core Integration Test Suite for Stellar
# =============================================================================
#
# This script deploys the Wormhole Core contract to Stellar testnet and runs
# a comprehensive suite of integration tests to verify contract functionality.
#
# PREREQUISITES:
#   - stellar CLI installed and configured
#   - jq installed for JSON parsing
#   - curl installed for RPC calls
#   - Test data files in ./test_data/ directory:
#       * guardian_keys.json          - Initial guardian keys (3 guardians)
#       * guardian_set_upgrade_vaas.json - VAAs for guardian set transitions
#       * set_message_fee_vaas.json   - VAAs for fee configuration
#       * transfer_fees_testnet_vaas.json - VAAs for fee transfers
#   - Network access to Stellar testnet
#
# USAGE:
#   ./run-integration-tests.sh [network]
#
#   Arguments:
#     network    Target network (default: testnet, only testnet supported)
#
# ENVIRONMENT VARIABLES:
#   DEPLOYER_IDENTITY   Stellar identity name for deployment (default: deployer)
#
# TEST FLOW:
#   1. Deploy fresh Wormhole Core contract via deploy.sh
#   2. Verify initial guardian set index is 0
#   3. Upgrade guardian set: 0 → 1 → 2
#   4. Set message fee to 10 XLM
#   5. Fund contract with XLM and test fee transfers (0.5 XLM)
#   6. Post message with fee (user1)
#   7. Reset fee to zero and post message without fee (user2)
#
# OUTPUTS:
#   - Contract info written to $STELLAR_ROOT/.testnet-contract-info
#   - Prints "ok: <CONTRACT_ID>" on success
#
# EXIT CODES:
#   0 - All tests passed
#   1 - Test failure or missing dependency
#
# =============================================================================
set -euo pipefail

die() { echo "run-integration-tests.sh: $*" >&2; exit 1; }
need() { command -v "$1" >/dev/null 2>&1 || die "missing '$1'"; }
step() { echo "==> $*" >&2; }

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
STELLAR_ROOT="$(cd "$SCRIPT_DIR/../../../.." && pwd)"
TEST_DATA="$SCRIPT_DIR/test_data"
DEPLOY_SH="$STELLAR_ROOT/scripts/deploy.sh"
NETWORK="${1:-testnet}"
DEPLOYER_IDENTITY="${DEPLOYER_IDENTITY:-deployer}"

# -----------------------------------------------------------------------------
# Validation
# -----------------------------------------------------------------------------
[[ "$NETWORK" == "testnet" ]] || die "only testnet is supported (got: $NETWORK)"
[[ -d "$TEST_DATA" ]] || die "missing test data: $TEST_DATA"
[[ -x "$DEPLOY_SH" ]] || die "missing deploy script: $DEPLOY_SH"

need stellar
need jq
need curl

cd "$STELLAR_ROOT"

# -----------------------------------------------------------------------------
# Deploy Contract
# -----------------------------------------------------------------------------
step "deploy Wormhole Core"
DEPLOY_JSON="$("$DEPLOY_SH" "$NETWORK")"
CONTRACT_ID="$(jq -er '.contract_id' <<<"$DEPLOY_JSON")"
XLM_CONTRACT="$(stellar contract id asset --asset native --network "$NETWORK")"

# -----------------------------------------------------------------------------
# Helper Functions
# -----------------------------------------------------------------------------

# Extract VAA hex from test data JSON by test case name
# Usage: vaa_hex <json_file> <test_case_name>
vaa_hex() { jq -er --arg name "$2" '.testCases[] | select(.name==$name) | .vaa.hex' "$1"; }

# Invoke Wormhole Core contract method
# Usage: core <source_identity> <method> [args...]
core() { local src="$1"; shift; stellar contract invoke --id "$CONTRACT_ID" --source-account "$src" --network "$NETWORK" -- "$@"; }

# Invoke native XLM contract method (for transfers/approvals)
# Usage: xlm <source_identity> <method> [args...]
xlm() { local src="$1"; shift; stellar contract invoke --id "$XLM_CONTRACT" --source-account "$src" --network "$NETWORK" -- "$@"; }

# Ensure a Stellar identity exists and is funded
# Creates the identity if it doesn't exist, funds via friendbot
# Usage: ensure_identity <identity_name>
# Returns: The identity's public address
ensure_identity() {
  local name="$1"
  stellar keys address "$name" >/dev/null 2>&1 || stellar keys generate "$name" --network "$NETWORK" >/dev/null 2>&1
  stellar keys fund "$name" --network "$NETWORK" >/dev/null 2>&1 || true
  stellar keys address "$name"
}

# -----------------------------------------------------------------------------
# Setup Test Identities
# -----------------------------------------------------------------------------
USER1_ADDR="$(ensure_identity user1)"
USER2_ADDR="$(ensure_identity user2)"
DEPLOYER_ADDR="$(stellar keys address "$DEPLOYER_IDENTITY")"

# =============================================================================
# TEST EXECUTION
# =============================================================================

# Test 1: Verify contract initialization
step "verify init"
[[ "$(core "$DEPLOYER_IDENTITY" get_current_guardian_set_index | grep -Eo '[0-9]+' | head -1)" == "0" ]] \
  || die "unexpected guardian set index"

# Test 2: Guardian set upgrade chain (0 → 1 → 2)
step "guardian set upgrades"
core "$DEPLOYER_IDENTITY" submit_guardian_set_upgrade \
  --vaa-bytes "\"$(vaa_hex "$TEST_DATA/guardian_set_upgrade_vaas.json" guardian_set_upgrade_0_to_1)\"" >/dev/null
core "$DEPLOYER_IDENTITY" submit_guardian_set_upgrade \
  --vaa-bytes "\"$(vaa_hex "$TEST_DATA/guardian_set_upgrade_vaas.json" guardian_set_upgrade_1_to_2)\"" >/dev/null
[[ "$(core "$DEPLOYER_IDENTITY" get_current_guardian_set_index | grep -Eo '[0-9]+' | head -1)" == "2" ]] \
  || die "guardian set upgrade failed"

# Test 3: Set message fee via governance VAA
step "set message fee (10 XLM)"
core "$DEPLOYER_IDENTITY" submit_set_message_fee \
  --vaa-bytes "\"$(vaa_hex "$TEST_DATA/set_message_fee_vaas.json" set_message_fee_10_xlm)\"" >/dev/null
[[ "$(core "$DEPLOYER_IDENTITY" get_message_fee | grep -Eo '[0-9]+' | head -1)" == "100000000" ]] \
  || die "unexpected message fee"

# Test 4: Fund contract and verify fee transfer governance action
step "fund contract and transfer fees"
xlm "$DEPLOYER_IDENTITY" transfer --from "$DEPLOYER_ADDR" --to "$CONTRACT_ID" --amount 200000000 >/dev/null
BALANCE_BEFORE="$(xlm "$DEPLOYER_IDENTITY" balance --id "$CONTRACT_ID" | grep -Eo '[0-9]+' | head -1)"
core "$DEPLOYER_IDENTITY" submit_transfer_fees \
  --vaa-bytes "\"$(vaa_hex "$TEST_DATA/transfer_fees_testnet_vaas.json" transfer_fees_0.5_xlm)\"" >/dev/null
BALANCE_AFTER="$(xlm "$DEPLOYER_IDENTITY" balance --id "$CONTRACT_ID" | grep -Eo '[0-9]+' | head -1)"
[[ "$BALANCE_AFTER" == "$((BALANCE_BEFORE - 5000000))" ]] || die "fee transfer balance mismatch"

# Test 5: Post a message with fee payment (requires XLM approval)
step "post message (with fee)"
# Use stellar rpc to get current ledger, then add offset for expiration
# Max offset allowed is 3110400 (~180 days), we use 1M (~58 days)
LEDGER_JSON=$(curl -s -X POST "https://soroban-testnet.stellar.org" \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"getLatestLedger"}' 2>/dev/null || echo '{}')
CURRENT_LEDGER=$(echo "$LEDGER_JSON" | jq -r '.result.sequence // 4000000')
EXPIRATION_LEDGER=$((CURRENT_LEDGER + 1000000))
xlm user1 approve --from "$USER1_ADDR" --spender "$CONTRACT_ID" --amount 100000000 --expiration-ledger "$EXPIRATION_LEDGER" >/dev/null
core user1 post_message \
  --emitter "$USER1_ADDR" \
  --nonce 42 \
  --payload "48656c6c6f20576f726d686f6c6521" \
  --consistency-level 1 >/dev/null  # 1 = Confirmed, payload = "Hello Wormhole!"

# Test 6: Post a message without fee (after setting fee to zero)
step "post message (no fee)"
core "$DEPLOYER_IDENTITY" submit_set_message_fee \
  --vaa-bytes "\"$(vaa_hex "$TEST_DATA/set_message_fee_vaas.json" set_message_fee_zero_fee)\"" >/dev/null
core user2 post_message \
  --emitter "$USER2_ADDR" \
  --nonce 100 \
  --payload "5465737420776974686f757420666565" \
  --consistency-level 1 >/dev/null  # 1 = Confirmed, payload = "Test without fee"

# -----------------------------------------------------------------------------
# Save Contract Info for Subsequent Scripts
# -----------------------------------------------------------------------------
cat > "$STELLAR_ROOT/.testnet-contract-info" <<EOF
CONTRACT_ID=$CONTRACT_ID
XLM_CONTRACT=$XLM_CONTRACT
NETWORK=$NETWORK
EOF

# All tests passed
echo ""
echo "┌─────────────────────────────────────────────────────────────────┐"
echo "│                    ✅ ALL TESTS PASSED                          │"
echo "└─────────────────────────────────────────────────────────────────┘"
echo "  Contract ID:    $CONTRACT_ID"
echo "  Network:        $NETWORK"
echo ""
