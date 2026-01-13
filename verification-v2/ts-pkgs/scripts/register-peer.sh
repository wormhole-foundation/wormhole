#!/bin/bash
# Sign and upload guardian peer data to the peer discovery server.
# Usage: ./register-peer.sh <GUARDIAN_KEY_PATH> <CERT_PATH> <TLS_HOSTNAME> <TLS_PORT> <PEER_SERVER_URL>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

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

if ! [[ "$TLS_PORT" =~ ^[0-9]+$ ]]; then
    log_error "TLS_PORT must be a number"
    exit 1
fi

GUARDIAN_KEY_PATH="$(cd "$(dirname "${GUARDIAN_KEY_PATH}")" && pwd)/$(basename "${GUARDIAN_KEY_PATH}")"
CERT_PATH="$(cd "$(dirname "${CERT_PATH}")" && pwd)/$(basename "${CERT_PATH}")"

export DOCKER_BUILDKIT=1

docker build \
    --file "${REPO_ROOT}/ts-pkgs/peer-client/Dockerfile" \
    --secret id=guardian_pk,src="${GUARDIAN_KEY_PATH}" \
    --secret id=cert.pem,src="${CERT_PATH}" \
    --build-arg TLS_HOSTNAME="${TLS_HOSTNAME}" \
    --build-arg TLS_PORT="${TLS_PORT}" \
    --build-arg PEER_SERVER_URL="${PEER_SERVER_URL}" \
    --progress=plain \
    "${REPO_ROOT}"

log_info "Registration complete"

