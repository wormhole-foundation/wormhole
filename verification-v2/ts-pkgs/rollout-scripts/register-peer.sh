#!/bin/bash
# Sign and upload guardian peer data to the peer discovery server.

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
PROJECT_ROOT="${SCRIPT_DIR}/../.."

log_info() { echo "[INFO] $1"; }
log_error() { echo "[ERROR] $1"; }

if [ $# -lt 5 ]; then
    echo "Usage: $0 <Guardian key option> <CERT_PATH> <TLS_HOSTNAME> <TLS_PORT> <PEER_SERVER_URL>"
    echo ""
    echo "Arguments:"
    echo "  CERT_PATH           - Path to the TLS certificate"
    echo "  TLS_HOSTNAME        - Hostname for this guardian's DKG server"
    echo "  TLS_PORT            - Port for this guardian's DKG server"
    echo "  PEER_SERVER_URL     - URL of the peer discovery server"
    echo "Guardian key option must be exactly one of these:"
    echo "  --key <KEY_PATH>    - Path to the guardian's Wormhole private key"
    echo "  --arn <AWS_KMS_ARN> - ARN of AWS KMS key"
    exit 1
fi

GUARDIAN_KEY_OPTION="$1"
CERT_PATH="$3"
TLS_HOSTNAME="$4"
TLS_PORT="$5"
PEER_SERVER_URL="$6"

if [ ${GUARDIAN_KEY_OPTION} == "--key" ]; then
    GUARDIAN_KEY_PATH="$2"
    if [ ! -f "${GUARDIAN_KEY_PATH}" ]; then
        log_error "Guardian key file not found: ${GUARDIAN_KEY_PATH}"
        exit 1
    fi
elif [ ${GUARDIAN_KEY_OPTION} == "--arn" ]; then
    GUARDIAN_KEY_ARN="$2"
else
    log_error "Either '--key' or '--arn' option needs to be provided"
    exit 1
fi

if [ ! -f "${CERT_PATH}" ]; then
    log_error "Certificate file not found: ${CERT_PATH}"
    exit 1
fi

export DOCKER_BUILDKIT=1

# TSS_E2E_DOCKER_BUILDER should NOT be used in production.
builder_option=""
if [ -n "${TSS_E2E_DOCKER_BUILDER:-}" ]; then
    builder_option+="--builder ${TSS_E2E_DOCKER_BUILDER} --network=host "
fi

if [ -n "${GUARDIAN_KEY_ARN:-}" ]; then
    builder_option+="--build-arg GUARDIAN_PRIVATE_KEY_ARN=${GUARDIAN_KEY_ARN} "
fi

run_option=""
if [ -n "${GUARDIAN_KEY_PATH:-}" ]; then
    run_option+="--mount=type=secret,id=guardian_pk,src=${GUARDIAN_KEY_PATH} "
fi

docker build ${builder_option} \
    --file "${PROJECT_ROOT}/ts-pkgs/peer-client/Dockerfile" \
    --build-arg TLS_HOSTNAME="${TLS_HOSTNAME}" \
    --build-arg TLS_PORT="${TLS_PORT}" \
    --build-arg PEER_SERVER_URL="${PEER_SERVER_URL}" \
    --tag register-peer
    "${PROJECT_ROOT}"

docker run ${run_option} \
    --rm \
    --mount=type=secret,id=cert.pem,src="${CERT_PATH}" \
    register-peer

log_info "Registration complete"

TLS_KEYS_DIR="$(dirname "${CERT_PATH}")"

echo ""
echo "=============================================="
echo "NEXT STEP: Run the DKG ceremony"
echo "=============================================="
echo ""
echo "Run the following command from the rollout-scripts directory:"
echo ""
echo "  ./run-dkg.sh \\"
echo "    ${TLS_KEYS_DIR} \\"
echo "    ${TLS_HOSTNAME} \\"
echo "    ${TLS_PORT} \\"
echo "    ${PEER_SERVER_URL} \\"
echo "    <ETHEREUM_RPC_URL>"
echo ""
echo "Where:"
echo "  ETHEREUM_RPC_URL  - Ethereum mainnet RPC URL"
echo ""
