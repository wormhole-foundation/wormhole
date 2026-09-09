#!/usr/bin/env bash
# Deploys and initializes the Wormhole core contract on the Tilt Stellar devnet.
#
# Runs as the `stellar-deploy` sidecar of the quickstart pod (devnet/stellar-devnet.yaml).
# The deployer key and salt are fixed (scripts/devnet-consts.json, chain 61), so the
# contract id is deterministic and the guardian is started with it before this script
# runs. A Soroban contract id depends only on (network passphrase, deployer address, salt),
# not on the wasm or the constructor arguments. To recompute it with the CLI:
#   stellar contract id wasm --salt <coreDeploySalt> --source-account <deployer secret> \
#     --network-passphrase 'Standalone Network ; February 2017' \
#     --rpc-url http://localhost:8000/soroban/rpc
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONSTS="${DEVNET_CONSTS:-$APP_DIR/devnet-consts.json}"
ENV_FILE="${DEVNET_ENV_FILE:-$APP_DIR/.env}"
WASM="${CORE_WASM:-$APP_DIR/wormhole_contract.wasm}"
RPC_URL="${STELLAR_RPC_URL:-http://localhost:8000/soroban/rpc}"
FRIENDBOT_URL="${STELLAR_FRIENDBOT_URL:-http://localhost:8000/friendbot}"

step() { echo "==> $*" >&2; }
die() { echo "devnet_deploy.sh: $*" >&2; exit 1; }

for tool in stellar jq curl; do
  command -v "$tool" >/dev/null 2>&1 || die "missing '$tool'"
done
[[ -f "$CONSTS" ]] || die "missing $CONSTS"
[[ -f "$ENV_FILE" ]] || die "missing $ENV_FILE"
[[ -f "$WASM" ]] || die "missing $WASM"

PASSPHRASE="$(jq -r '.chains."61".networkPassphrase' "$CONSTS")"
DEPLOYER_SECRET="$(jq -r '.chains."61".devnetDeployer.secret' "$CONSTS")"
DEPLOYER_PUBLIC="$(jq -r '.chains."61".devnetDeployer.public' "$CONSTS")"
SALT="$(jq -r '.chains."61".coreDeploySalt' "$CONSTS")"
EXPECTED_ID="$(jq -r '.chains."61".contracts.coreNativeAddress' "$CONSTS")"
GOVERNANCE_EMITTER="$(jq -r '.global.governanceEmitterAddress' "$CONSTS")"
for v in PASSPHRASE DEPLOYER_SECRET DEPLOYER_PUBLIC SALT EXPECTED_ID GOVERNANCE_EMITTER; do
  [[ -n "${!v}" && "${!v}" != "null" ]] || die "$CONSTS is missing the value for $v"
done

# INIT_SIGNERS_CSV is written by scripts/guardian-set-init.sh (via the const-gen image) for
# the configured number of guardians: comma-separated 20-byte hex addresses without 0x.
INIT_SIGNERS_CSV="$(sed -n 's/^INIT_SIGNERS_CSV=//p' "$ENV_FILE")"
[[ -n "$INIT_SIGNERS_CSV" ]] || die "INIT_SIGNERS_CSV not found in $ENV_FILE"
GUARDIANS_JSON="$(jq -Rc 'split(",")' <<<"$INIT_SIGNERS_CSV")"
GUARDIAN_COUNT="$(jq -r 'length' <<<"$GUARDIANS_JSON")"

# Common CLI options. They are options of the `invoke`/`deploy` subcommands, so they go after the
# subcommand name and before `--`: everything after `--` is parsed as contract-function arguments.
CLI_OPTS=(--source-account "$DEPLOYER_SECRET" --rpc-url "$RPC_URL" --network-passphrase "$PASSPHRASE")

rpc() {
  curl -sf -X POST "$RPC_URL" -H 'Content-Type: application/json' -d "$1"
}

step "wait for soroban rpc at $RPC_URL"
until rpc '{"jsonrpc":"2.0","id":1,"method":"getHealth"}' | jq -e '.result.status == "healthy"' >/dev/null 2>&1; do
  sleep 2
done

# Friendbot comes up shortly after the rpc is healthy. Retry while it is unreachable (curl
# failure, or a 5xx from the proxy); accept an already-funded account (the sidecar may be
# restarting); treat any other 4xx as final, since repeating the same request cannot fix it.
# Wordings seen from friendbot: "account already funded to starting balance",
# "createAccountAlreadyExist", "op_already_exists".
step "fund deployer $DEPLOYER_PUBLIC"
funded=false
for _ in $(seq 1 90); do
  response="$(curl -s --max-time 30 -w '\n%{http_code}' "$FRIENDBOT_URL?addr=$DEPLOYER_PUBLIC" || true)"
  http_code="${response##*$'\n'}"
  body="${response%$'\n'*}"
  case "$http_code" in
    200) funded=true; break ;;
    4??)
      if grep -qiE 'already[ _]?(funded|exist)' <<<"$body"; then
        step "deployer already funded"
        funded=true
        break
      fi
      die "friendbot rejected the funding request (HTTP $http_code): $body"
      ;;
    *) sleep 2 ;;
  esac
done
[[ "$funded" == "true" ]] || die "friendbot at $FRIENDBOT_URL did not become available (last response: $response)"

# A restarted sidecar must not fail on an already-deployed contract.
if stellar contract invoke "${CLI_OPTS[@]}" --id "$EXPECTED_ID" -- get_current_guardian_set_index >/dev/null 2>&1; then
  step "core contract $EXPECTED_ID already deployed"
else
  # The constructor runs atomically at deploy time (initial guardian set + governance emitter).
  step "deploy core contract ($GUARDIAN_COUNT guardians)"
  if ! DEPLOY_OUTPUT="$(stellar contract deploy "${CLI_OPTS[@]}" --wasm "$WASM" --salt "$SALT" -- \
      --initial_guardians "$GUARDIANS_JSON" --governance_emitter "\"$GOVERNANCE_EMITTER\"" 2>&1)"; then
    echo "$DEPLOY_OUTPUT" >&2
    die "deploy failed"
  fi
  CONTRACT_ID="$(grep -Eo 'C[A-Z0-9]{55}' <<<"$DEPLOY_OUTPUT" | head -1 || true)"
  [[ -n "$CONTRACT_ID" ]] || { echo "$DEPLOY_OUTPUT" >&2; die "failed to parse contract id"; }
  [[ "$CONTRACT_ID" == "$EXPECTED_ID" ]] || die "deployed contract id $CONTRACT_ID does not match the expected $EXPECTED_ID; update .chains.\"61\" in scripts/devnet-consts.json (the guardian is configured with the expected id)"
fi

step "verify init"
if ! INDEX="$(stellar contract invoke "${CLI_OPTS[@]}" --id "$EXPECTED_ID" -- get_current_guardian_set_index 2>&1)"; then
  echo "$INDEX" >&2
  die "get_current_guardian_set_index failed"
fi
INDEX="$(tail -n 1 <<<"$INDEX" | tr -d '"[:space:]')"
[[ "$INDEX" == "0" ]] || die "unexpected guardian set index '$INDEX'"

echo "stellar devnet core contract: $EXPECTED_ID ($GUARDIAN_COUNT guardians)"
