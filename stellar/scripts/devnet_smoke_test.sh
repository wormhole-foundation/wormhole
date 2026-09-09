#!/usr/bin/env bash
# Tilt CI smoke test for the Stellar integration (devnet/stellar-ci-tests.yaml).
#
# Deploys the devnet-emitter contract, publishes one message through the core contract and
# waits until the guardians serve a signed VAA for it. Exit code 0 means the whole path
# (contract event -> stellar watcher -> gossip/quorum -> public REST) works.
set -euo pipefail

APP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONSTS="${DEVNET_CONSTS:-$APP_DIR/devnet-consts.json}"
EMITTER_WASM="${EMITTER_WASM:-$APP_DIR/devnet_emitter.wasm}"
RPC_URL="${STELLAR_RPC_URL:-http://stellar:8000/soroban/rpc}"
GUARDIAN_HOST="${GUARDIAN_HOST:-guardian}"   # headless service: pods resolve as guardian-<i>.guardian
NUM_GUARDIANS="${NUM_GUARDIANS:-1}"
VAA_TIMEOUT_SECONDS="${VAA_TIMEOUT_SECONDS:-300}"
CHAIN_ID=61
NONCE=1
PAYLOAD_HEX=deadbeef
CONSISTENCY_LEVEL=1   # ConsistencyLevel::Confirmed

step() { echo "==> $*" >&2; }
die() { echo "devnet_smoke_test.sh: $*" >&2; exit 1; }

for tool in stellar jq curl base32 od; do
  command -v "$tool" >/dev/null 2>&1 || die "missing '$tool'"
done
[[ -f "$CONSTS" ]] || die "missing $CONSTS"
[[ -f "$EMITTER_WASM" ]] || die "missing $EMITTER_WASM"

PASSPHRASE="$(jq -r '.chains."61".networkPassphrase' "$CONSTS")"
DEPLOYER_SECRET="$(jq -r '.chains."61".devnetDeployer.secret' "$CONSTS")"
CORE_ID="$(jq -r '.chains."61".contracts.coreNativeAddress' "$CONSTS")"
for v in PASSPHRASE DEPLOYER_SECRET CORE_ID; do
  [[ -n "${!v}" && "${!v}" != "null" ]] || die "$CONSTS is missing the value for $v"
done

# Common CLI options. They are options of the `invoke`/`deploy` subcommands, so they go after the
# subcommand name and before `--`: everything after `--` is parsed as contract-function arguments.
CLI_OPTS=(--source-account "$DEPLOYER_SECRET" --rpc-url "$RPC_URL" --network-passphrase "$PASSPHRASE")

# Decodes a StrKey contract id (C...) to the 32-byte hex the guardian uses as emitter address:
# 1 version byte + 32 payload bytes + 2 CRC bytes, base32 without padding.
strkey_to_hex() {
  printf '%s' "$1" | base32 -d | od -An -tx1 -v | tr -d ' \n' | cut -c3-66
}

step "wait for $NUM_GUARDIANS guardian(s)"
for ((i = 0; i < NUM_GUARDIANS; i++)); do
  until [[ "$(curl -s -o /dev/null -w '%{http_code}' "http://guardian-$i.$GUARDIAN_HOST:6060/readyz")" == "200" ]]; do
    sleep 5
  done
done

step "wait for core contract $CORE_ID"
until stellar contract invoke "${CLI_OPTS[@]}" --id "$CORE_ID" -- get_current_guardian_set_index >/dev/null 2>&1; do
  sleep 5
done

step "deploy devnet-emitter"
if ! DEPLOY_OUTPUT="$(stellar contract deploy "${CLI_OPTS[@]}" --wasm "$EMITTER_WASM" -- --core "$CORE_ID" 2>&1)"; then
  echo "$DEPLOY_OUTPUT" >&2
  die "emitter deploy failed"
fi
EMITTER_ID="$(grep -Eo 'C[A-Z0-9]{55}' <<<"$DEPLOY_OUTPUT" | grep -v "$CORE_ID" | head -1 || true)"
[[ -n "$EMITTER_ID" ]] || { echo "$DEPLOY_OUTPUT" >&2; die "failed to parse emitter contract id"; }
EMITTER_HEX="$(strkey_to_hex "$EMITTER_ID")"
[[ ${#EMITTER_HEX} -eq 64 ]] || die "bad emitter hex '$EMITTER_HEX'"
step "emitter $EMITTER_ID ($EMITTER_HEX)"

step "publish message"
SEQUENCE="$(stellar contract invoke "${CLI_OPTS[@]}" --id "$EMITTER_ID" -- send \
  --nonce "$NONCE" --payload "\"$PAYLOAD_HEX\"" --consistency_level "$CONSISTENCY_LEVEL" 2>/dev/null | tr -d '"' | tr -d '[:space:]')"
[[ "$SEQUENCE" =~ ^[0-9]+$ ]] || die "unexpected sequence '$SEQUENCE'"
step "published sequence $SEQUENCE"

URL="http://guardian-0.$GUARDIAN_HOST:7071/v1/signed_vaa/$CHAIN_ID/$EMITTER_HEX/$SEQUENCE"
step "wait up to ${VAA_TIMEOUT_SECONDS}s for signed VAA at $URL"
deadline=$((SECONDS + VAA_TIMEOUT_SECONDS))
until BODY="$(curl -sf "$URL")"; do
  (( SECONDS < deadline )) || die "timed out waiting for the signed VAA"
  sleep 5
done
jq -e '.vaaBytes | length > 0' <<<"$BODY" >/dev/null || die "unexpected response: $BODY"

echo "stellar smoke test passed: VAA for $CHAIN_ID/$EMITTER_HEX/$SEQUENCE is available"
