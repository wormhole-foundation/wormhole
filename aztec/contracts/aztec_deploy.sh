#!/usr/bin/env bash
# aztec_deploy.sh — One-shot setup for Aztec testnet wallets & contracts
#
# This script automates:
#  - Setting environment variables from a local .env file
#  - Creating owner & receiver wallets
#  - Registering wallets with the Sponsored FPC
#  - Deploying both accounts
#  - Deploying the Token contract and saving its address
#  - Minting private and public tokens
#  - Deploying the Wormhole contract
#
# Logs are automatically saved under ./logs/ with timestamped filenames.
#
# Usage:
#   1. Create a .env file with required values (NODE_URL, SPONSORED_FPC_ADDRESS, OWNER_SK).
#   2. Run:  bash aztec_deploy.sh
#   3. Review logs under ./logs/

set -euo pipefail

# ---------- Load .env if present ----------
if [[ -f .env ]]; then
  set -a; source .env; set +a
fi

# ---------- Config ----------
NODE_URL=${NODE_URL:-"https://aztec-alpha-testnet-fullnode.zkv.xyz"}
SPONSORED_FPC_ADDRESS=${SPONSORED_FPC_ADDRESS:-""}
OWNER_SK=${OWNER_SK:-""}

OWNER_ALIAS=${OWNER_ALIAS:-"owner-wallet"}
RECEIVER_ALIAS=${RECEIVER_ALIAS:-"receiver-wallet"}
SPONSORED_FPC_ALIAS=${SPONSORED_FPC_ALIAS:-"sponsoredfpc"}

TOKEN_ALIAS=${TOKEN_ALIAS:-"token"}
TOKEN_NAME=${TOKEN_NAME:-"WormToken"}
TOKEN_SYMBOL=${TOKEN_SYMBOL:-"WORM"}
TOKEN_DECIMALS=${TOKEN_DECIMALS:-18}
TOKEN_ADDRESS=${TOKEN_ADDRESS:-""}
TOKEN_ADDRESS_FILE=${TOKEN_ADDRESS_FILE:-"./token_address.txt"}

WORMHOLE_ALIAS=${WORMHOLE_ALIAS:-"wormhole"}
WORMHOLE_ARTIFACT=${WORMHOLE_ARTIFACT:-"wormhole_contracts-Wormhole.json"}
WORMHOLE_CHAIN_ID_1=${WORMHOLE_CHAIN_ID_1:-56}
WORMHOLE_CHAIN_ID_2=${WORMHOLE_CHAIN_ID_2:-56}
WORMHOLE_GOV_KEY=${WORMHOLE_GOV_KEY:-"0x1f41267c06dae96c9c3906c5f77cbc28602cc70d6d7e9d2c3072cb0a5b13edd2"}

MINT_PRIVATE_AMOUNT=${MINT_PRIVATE_AMOUNT:-10000}
MINT_PUBLIC_AMOUNT=${MINT_PUBLIC_AMOUNT:-10000}

# ---------- Logging ----------
LOG_DIR=${LOG_DIR:-"./logs"}
mkdir -p "$LOG_DIR"
LOG_FILE="$LOG_DIR/deploy_$(date +%Y%m%d_%H%M%S).log"
exec > >(tee -a "$LOG_FILE") 2>&1

# ---------- Helpers ----------
mask() { local s="$1"; [[ -z "$s" ]] && { echo ""; return; }; echo "${s:0:8}...${s: -4}"; }
need() { local n="$1" v="$2"; [[ -z "$v" ]] && { echo "Missing required env: $n" >&2; exit 1; }; }
check_bin() { command -v aztec-wallet >/dev/null 2>&1 || { echo "aztec-wallet CLI not found" >&2; exit 1; }; }
extract_address() { awk '{print}' | tee /tmp/azc_last_cmd.out | grep -Eo "0x[0-9a-fA-F]{40,}" | head -n1; }
section() { echo -e "\n=== $1 ===\n"; }

# ---------- Pre-flight ----------
section "Pre-flight checks"
check_bin

echo "Logs: $LOG_FILE"

echo "NODE_URL              = $NODE_URL"
echo "SPONSORED_FPC_ADDRESS = $SPONSORED_FPC_ADDRESS"
echo "OWNER_SK              = $(mask "$OWNER_SK")"
echo "OWNER_ALIAS           = $OWNER_ALIAS"
echo "RECEIVER_ALIAS        = $RECEIVER_ALIAS"

echo "TOKEN_ALIAS           = $TOKEN_ALIAS"
echo "TOKEN_NAME            = $TOKEN_NAME"
echo "TOKEN_SYMBOL          = $TOKEN_SYMBOL"
echo "TOKEN_DECIMALS        = $TOKEN_DECIMALS"

echo "WORMHOLE_ALIAS        = $WORMHOLE_ALIAS"
echo "MINT_PRIVATE_AMOUNT   = $MINT_PRIVATE_AMOUNT"
echo "MINT_PUBLIC_AMOUNT    = $MINT_PUBLIC_AMOUNT"

need SPONSORED_FPC_ADDRESS "$SPONSORED_FPC_ADDRESS"
need OWNER_SK "$OWNER_SK"

section "aztec-wallet version"
aztec-wallet --version || true

# ---------- 2. Create wallets ----------
section "Create OWNER wallet"
aztec-wallet create-account -sk "$OWNER_SK" --register-only --node-url "$NODE_URL" --alias "$OWNER_ALIAS"

section "Create RECEIVER wallet"
aztec-wallet create-account --register-only --node-url "$NODE_URL" --alias "$RECEIVER_ALIAS"

# ---------- 3. Register wallets with FPC ----------
section "Register OWNER with FPC"
aztec-wallet register-contract --node-url "$NODE_URL" --from "$OWNER_ALIAS" --alias "$SPONSORED_FPC_ALIAS" "$SPONSORED_FPC_ADDRESS" SponsoredFPC --salt 0

section "Register RECEIVER with FPC"
aztec-wallet register-contract --node-url "$NODE_URL" --from "$RECEIVER_ALIAS" --alias "$SPONSORED_FPC_ALIAS" "$SPONSORED_FPC_ADDRESS" SponsoredFPC --salt 0

# ---------- 4. Deploy accounts ----------
section "Deploy OWNER account"
aztec-wallet deploy-account --node-url "$NODE_URL" --from "$OWNER_ALIAS" --payment method=fpc-sponsored,fpc=contracts:"$SPONSORED_FPC_ALIAS"

section "Deploy RECEIVER account"
aztec-wallet deploy-account --node-url "$NODE_URL" --from "$RECEIVER_ALIAS" --payment method=fpc-sponsored,fpc=contracts:"$SPONSORED_FPC_ALIAS"

# ---------- 5. Deploy Token contract ----------
section "Deploy Token contract"
if [[ -z "$TOKEN_ADDRESS" ]]; then
  DEPLOY_OUT=$(mktemp)
  set +e
  aztec-wallet deploy --node-url "$NODE_URL" --from accounts:"$OWNER_ALIAS" --payment method=fpc-sponsored,fpc=contracts:"$SPONSORED_FPC_ALIAS" --alias "$TOKEN_ALIAS" TokenContract --args accounts:"$OWNER_ALIAS" "$TOKEN_NAME" "$TOKEN_SYMBOL" "$TOKEN_DECIMALS" --no-wait | tee "$DEPLOY_OUT"
  set -e
  TOKEN_ADDRESS=$(grep -Eo "0x[0-9a-fA-F]{40,}" "$DEPLOY_OUT" | head -n1 || true)
  if [[ -n "$TOKEN_ADDRESS" ]]; then
    echo "Captured TOKEN_ADDRESS=$TOKEN_ADDRESS"
    echo "$TOKEN_ADDRESS" > "$TOKEN_ADDRESS_FILE"
  else
    echo "Could not parse token address."
  fi
else
  echo "Using provided TOKEN_ADDRESS=$TOKEN_ADDRESS"
fi

# ---------- 6. Mint tokens ----------
if [[ -n "$TOKEN_ADDRESS" ]]; then
  section "Mint to PRIVATE"
  aztec-wallet send mint_to_private --node-url "$NODE_URL" --from accounts:"$OWNER_ALIAS" --payment method=fpc-sponsored,fpc=contracts:"$SPONSORED_FPC_ALIAS" --contract-address "$TOKEN_ADDRESS" --args accounts:"$OWNER_ALIAS" accounts:"$OWNER_ALIAS" "$MINT_PRIVATE_AMOUNT"

  section "Mint to PUBLIC"
  aztec-wallet send mint_to_public --node-url "$NODE_URL" --from accounts:"$OWNER_ALIAS" --payment method=fpc-sponsored,fpc=contracts:"$SPONSORED_FPC_ALIAS" --contract-address "$TOKEN_ADDRESS" --args accounts:"$OWNER_ALIAS" "$MINT_PUBLIC_AMOUNT"
fi

# ---------- 7. Deploy Wormhole contract ----------
section "Deploy Wormhole contract"
aztec-wallet deploy --node-url "$NODE_URL" --from accounts:"$OWNER_ALIAS" --payment method=fpc-sponsored,fpc=contracts:"$SPONSORED_FPC_ALIAS" --alias "$WORMHOLE_ALIAS" "$WORMHOLE_ARTIFACT" --args "$WORMHOLE_CHAIN_ID_1" "$WORMHOLE_CHAIN_ID_2" "$WORMHOLE_GOV_KEY" "${TOKEN_ADDRESS:-0x}" --no-wait --init init* || echo "Wormhole deploy returned non-zero status"

section "Done"
echo "Finished. Review $LOG_FILE for details."
