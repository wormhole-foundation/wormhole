#!/bin/bash
# Sign and upload guardian peer data to the peer discovery server.
# Usage: ./register-peer.sh <GUARDIAN_KEY_PATH> <CERT_PATH> <TLS_HOSTNAME> <TLS_PORT> <PEER_SERVER_URL>

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
PROJECT_ROOT="${SCRIPT_DIR}/../.."

log_info() { echo "[INFO] $1"; }
log_error() { echo "[ERROR] $1"; }

if [ $# -lt 5 ]; then
    echo "Usage: $0 <GUARDIAN_KEY_PATH> <CERT_PATH> <TLS_HOSTNAME> <TLS_PORT> <PEER_SERVER_URL>"
    echo ""
    echo "Arguments:"
    echo "  GUARDIAN_KEY_PATH - Path to the guardian's Wormhole private key"
    echo "  CERT_PATH         - Path to the TLS certificate"
    echo "  TLS_HOSTNAME      - Hostname for this guardian's DKG server"
    echo "  TLS_PORT          - Port for this guardian's DKG server"
    echo "  PEER_SERVER_URL   - URL of the peer discovery server"
    exit 1
fi

GUARDIAN_KEY_PATH="$1"
CERT_PATH="$2"
TLS_HOSTNAME="$3"
TLS_PORT="$4"
PEER_SERVER_URL="$5"

if [ ! -f "${GUARDIAN_KEY_PATH}" ]; then
    log_error "Guardian key file not found: ${GUARDIAN_KEY_PATH}"
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
    builder_option="--builder ${TSS_E2E_DOCKER_BUILDER} --network=host"
fi

docker build ${builder_option} \
    --file "${PROJECT_ROOT}/ts-pkgs/peer-client/Dockerfile" \
    --secret id=guardian_pk,src="${GUARDIAN_KEY_PATH}" \
    --secret id=cert.pem,src="${CERT_PATH}" \
    --build-arg TLS_HOSTNAME="${TLS_HOSTNAME}" \
    --build-arg TLS_PORT="${TLS_PORT}" \
    --build-arg PEER_SERVER_URL="${PEER_SERVER_URL}" \
    "${PROJECT_ROOT}"

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
