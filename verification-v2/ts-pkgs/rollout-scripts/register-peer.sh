#!/bin/bash
# Sign and upload guardian peer data to the peer discovery server.

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
PROJECT_ROOT="${SCRIPT_DIR}/../.."

log_info()  { echo "[INFO] $1" >&2;  }
log_error() { echo "[ERROR] $1" >&2; }

usage() {
  cat >&2 <<EOF
Usage:
  $0 \
    <Guardian key option> \
    --tls-hostname=HOST \
    --tls-port=PORT \
    --tls-certificate=PATH \
    --peer-server-url=URL

Required:
  --tls-hostname=HOST      Hostname for this guardian's DKG server.
  --tls-port=PORT          Port for this guardian's DKG server.
  --tls-certificate=PATH   Path to the TLS certificate.
  --peer-server-url=URL    URL of the peer discovery server.
Guardian key option must be exactly one of these:
  --key=KEY_PATH           Path to the guardian's Wormhole private key.
  --arn=AWS_KMS_ARN        ARN of AWS KMS key.
EOF
}

check_guardian_option_is_undefined() {
  if [ -n "${GUARDIAN_KEY_OPTION:-}" ]; then
    log_error "Guardian key option must be provided exactly once."
    echo >&2
    usage
    exit 1
  fi
}

# Defaults
TLS_HOSTNAME=""
TLS_PORT=""
TLS_CERTIFICATE=""
PEER_SERVER_URL=""
GUARDIAN_KEY_OPTION=""

for arg in "$@"; do
  case "$arg" in
    --tls-hostname=*)
      TLS_HOSTNAME="${arg#*=}" ;;
    --tls-port=*)
      TLS_PORT="${arg#*=}" ;;
    --tls-certificate=*)
      TLS_CERTIFICATE="${arg#*=}" ;;
    --peer-server-url=*)
      PEER_SERVER_URL="${arg#*=}" ;;
    --key=*)
      check_guardian_option_is_undefined
      GUARDIAN_KEY_OPTION=--key
      GUARDIAN_KEY_PATH="${arg#*=}" ;;
    --arn=*)
      check_guardian_option_is_undefined
      GUARDIAN_KEY_OPTION=--arn
      GUARDIAN_KEY_ARN="${arg#*=}" ;;
    --help|-h)
      usage; exit 0 ;;
    *)
      log_error "Unknown option: $arg"
      echo >&2
      usage
      exit 1 ;;
  esac
done

missing=()
[ -n "$TLS_HOSTNAME"    ] || missing+=("--tls-hostname"   )
[ -n "$TLS_PORT"        ] || missing+=("--tls-port"       )
[ -n "$TLS_CERTIFICATE" ] || missing+=("--tls-certificate")
[ -n "$PEER_SERVER_URL" ] || missing+=("--peer-server-url")

if (( ${#missing[@]} )); then
  log_error "Missing required option(s): ${missing[*]}"
  echo >&2
  usage
  exit 1
fi

if [ -z ${GUARDIAN_KEY_OPTION} ]; then
  log_error "At least one Guardian key option must be provided: --key or --arn"
  echo >&2
  usage
  exit 1
fi

if [ ! -f "${TLS_CERTIFICATE}" ]; then
    log_error "Certificate file not found: ${TLS_CERTIFICATE}"
    exit 1
fi

export DOCKER_BUILDKIT=1

builder_option=""
if [ -n "${GUARDIAN_KEY_ARN:-}" ]; then
    builder_option+="--build-arg GUARDIAN_KMS_ARN=${GUARDIAN_KEY_ARN} "
fi

run_option=""
if [ -n "${GUARDIAN_KEY_PATH:-}" ]; then
    run_option+="--volume ${GUARDIAN_KEY_PATH}:/run/secrets/guardian_pk:ro "
fi

# TSS_E2E_DOCKER_NETWORK should NOT be used in production.
if [ -n "${TSS_E2E_DOCKER_NETWORK:-}" ]; then
    run_option+="--network ${TSS_E2E_DOCKER_NETWORK} "
fi


docker build ${builder_option} \
    --file "${PROJECT_ROOT}/ts-pkgs/peer-client/Dockerfile" \
    --build-arg TLS_HOSTNAME="${TLS_HOSTNAME}" \
    --build-arg TLS_PORT="${TLS_PORT}" \
    --build-arg PEER_SERVER_URL="${PEER_SERVER_URL}" \
    --tag "register-peer${TSS_E2E_GUARDIAN_ID:-}" \
    "${PROJECT_ROOT}"

docker run ${run_option} \
    --rm \
    --volume "${TLS_CERTIFICATE}:/run/secrets/cert.pem:ro" \
    "register-peer${TSS_E2E_GUARDIAN_ID:-}"

log_info "Registration complete"

TLS_KEYS_DIR="$(dirname "${TLS_CERTIFICATE}")"

echo ""
echo "=============================================="
echo "NEXT STEP: Run the DKG ceremony"
echo "=============================================="
echo ""
echo "Run the following command from the rollout-scripts directory:"
echo ""
echo "  ./run-dkg.sh \\"
echo "    --tls-keys-dir=${TLS_KEYS_DIR} \\"
echo "    --tls-hostname=${TLS_HOSTNAME} \\"
echo "    --tls-port=${TLS_PORT} \\"
echo "    --peer-server-url=${PEER_SERVER_URL} \\"
echo "    --ethereum-rpc-url=ETHEREUM_RPC_URL"
echo ""
echo "Where:"
echo "  ETHEREUM_RPC_URL  - Ethereum mainnet RPC URL"
echo ""
