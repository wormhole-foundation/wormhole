#!/bin/bash
# Generate TLS key pair and certificate for mTLS during DKG.
# Usage: ./generate-tls.sh <TLS_HOSTNAME> <TLS_PUBLIC_IP> <OUTPUT_DIR>

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
REPO_ROOT="${SCRIPT_DIR}/../.."

log_info() { echo "[INFO] $1"; }
log_error() { echo "[ERROR] $1"; }

if [ $# -lt 3 ]; then
    echo "Usage: $0 <TLS_HOSTNAME> <TLS_PUBLIC_IP> <OUTPUT_DIR>"
    echo ""
    echo "Arguments:"
    echo "  TLS_HOSTNAME   - Fully qualified hostname for this guardian"
    echo "  TLS_PUBLIC_IP  - Public IP address for this guardian"
    echo "  OUTPUT_DIR     - Directory to store generated keys"
    exit 1
fi

TLS_HOSTNAME="$1"
TLS_PUBLIC_IP="$2"
OUTPUT_DIR="$3"

mkdir -p "${OUTPUT_DIR}"

if [ -f "${OUTPUT_DIR}/key.pem" ] || [ -f "${OUTPUT_DIR}/cert.pem" ]; then
    if [ -n "${FORCE_OVERWRITE:-}" ]; then
        log_info "Overwriting existing TLS credentials (FORCE_OVERWRITE=1)"
    elif [ -n "${NON_INTERACTIVE:-}" ]; then
        log_error "TLS credentials already exist. Set FORCE_OVERWRITE=1 to overwrite."
        exit 1
    else
        read -p "TLS credentials already exist. Overwrite? (y/N) " -n 1 -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && exit 0
    fi
fi

docker build \
    --tag tls-gen \
    --file "${REPO_ROOT}/ts-pkgs/peer-client/tls.Dockerfile" \
    "${REPO_ROOT}"

docker run \
    --rm \
    --mount type=bind,src="${OUTPUT_DIR}",dst=/keys \
    --env TLS_HOSTNAME="${TLS_HOSTNAME}" \
    --env TLS_PUBLIC_IP="${TLS_PUBLIC_IP}" \
    tls-gen

if [ -f "${OUTPUT_DIR}/key.pem" ] && [ -f "${OUTPUT_DIR}/cert.pem" ]; then
    log_info "TLS credentials saved to ${OUTPUT_DIR}"
else
    log_error "Failed to generate TLS credentials"
    exit 1
fi

if [ -z "${SKIP_NEXT_STEP_HINT:-}" ]; then
    echo ""
    echo "=============================================="
    echo "NEXT STEP: Register your peer with the discovery server"
    echo "=============================================="
    echo ""
    echo "Run the following command from the rollout-scripts directory:"
    echo ""
    echo "  ./register-peer.sh \\"
    echo "    <GUARDIAN_KEY_PATH> \\"
    echo "    ${OUTPUT_DIR}/cert.pem \\"
    echo "    ${TLS_HOSTNAME} \\"
    echo "    <TLS_PORT> \\"
    echo "    <PEER_SERVER_URL>"
    echo ""
    echo "Where:"
    echo "  GUARDIAN_KEY_PATH - Path to your guardian's Wormhole private key"
    echo "  TLS_PORT          - Port your DKG server will listen on (e.g., 8443)"
    echo "  PEER_SERVER_URL   - URL of the peer discovery server"
    echo ""
fi
