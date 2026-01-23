#!/bin/bash
# Run the DKG ceremony. Polls for all peers, then generates key shards.
# Usage: ./run-dkg.sh <TLS_KEYS_DIR> <TLS_HOSTNAME> <TLS_PORT> <PEER_SERVER_URL> <ETHEREUM_RPC_URL> [WORMHOLE_ADDRESS]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

log_info() { echo "[INFO] $1"; }
log_error() { echo "[ERROR] $1"; }

if [ $# -lt 5 ]; then
    echo "Usage: $0 <TLS_KEYS_DIR> <TLS_HOSTNAME> <TLS_PORT> <PEER_SERVER_URL> <ETHEREUM_RPC_URL> [WORMHOLE_ADDRESS]"
    echo ""
    echo "Arguments:"
    echo "  TLS_KEYS_DIR      - Directory containing key.pem and cert.pem (also used for DKG outputs)"
    echo "  TLS_HOSTNAME      - Hostname for this guardian's DKG server"
    echo "  TLS_PORT          - Port for this guardian's DKG server"
    echo "  PEER_SERVER_URL   - URL of the peer discovery server"
    echo "  ETHEREUM_RPC_URL  - Ethereum RPC URL"
    echo "  WORMHOLE_ADDRESS  - (Optional) Wormhole contract address for testnet/devnet"
    exit 1
fi

TLS_KEYS_DIR="$1"
TLS_HOSTNAME="$2"
TLS_PORT="$3"
PEER_SERVER_URL="$4"
ETHEREUM_RPC_URL="$5"
WORMHOLE_ADDRESS="${6:-}"

if [ ! -d "${TLS_KEYS_DIR}" ]; then
    log_error "TLS keys directory not found: ${TLS_KEYS_DIR}"
    exit 1
fi

if [ ! -f "${TLS_KEYS_DIR}/key.pem" ]; then
    log_error "TLS private key not found: ${TLS_KEYS_DIR}/key.pem"
    exit 1
fi

if [ ! -f "${TLS_KEYS_DIR}/cert.pem" ]; then
    log_error "TLS certificate not found: ${TLS_KEYS_DIR}/cert.pem"
    exit 1
fi

if ! [[ "$TLS_PORT" =~ ^[0-9]+$ ]]; then
    log_error "TLS_PORT must be a number"
    exit 1
fi

TLS_KEYS_DIR="$(cd "${TLS_KEYS_DIR}" && pwd)"

# Optional: use DOCKER_NETWORK env var for custom network
NETWORK_FLAG=""
if [ -n "${DOCKER_NETWORK:-}" ]; then
    NETWORK_FLAG="--network=${DOCKER_NETWORK}"
fi

# Build the DKG client image (skip if SKIP_BUILD is set)
if [ -z "${SKIP_BUILD:-}" ]; then
    docker build \
        --tag dkg-client \
        --file "${REPO_ROOT}/ts-pkgs/peer-client/dkg.Dockerfile" \
        "${REPO_ROOT}"
fi

DOCKER_IT_FLAG="-it"
if [ ! -t 0 ] || [ -n "${NON_INTERACTIVE:-}" ]; then
    DOCKER_IT_FLAG=""
fi

PUBLISH_FLAG="--publish ${TLS_PORT}:${TLS_PORT}"
if [ -n "${DOCKER_NETWORK:-}" ]; then
    PUBLISH_FLAG=""
fi

WORMHOLE_ENV=""
if [ -n "${WORMHOLE_ADDRESS}" ]; then
    WORMHOLE_ENV="--env WORMHOLE_CONTRACT_ADDRESS=${WORMHOLE_ADDRESS}"
fi

docker run \
    ${DOCKER_IT_FLAG} \
    --rm \
    --name "${TLS_HOSTNAME}" \
    ${NETWORK_FLAG} \
    ${PUBLISH_FLAG} \
    --mount type=bind,src="${TLS_KEYS_DIR}",dst=/keys \
    --env TLS_HOSTNAME="${TLS_HOSTNAME}" \
    --env TLS_PORT="${TLS_PORT}" \
    --env PEER_SERVER_URL="${PEER_SERVER_URL}" \
    --env ETHEREUM_RPC_URL="${ETHEREUM_RPC_URL}" \
    ${WORMHOLE_ENV} \
    dkg-client

log_info "DKG complete"

echo ""
echo "=============================================="
echo "DKG CEREMONY COMPLETED SUCCESSFULLY"
echo "=============================================="
echo ""
echo "Your DKG key shards have been generated and saved to ${TLS_KEYS_DIR}."
echo ""
echo "Please verify the generated files and securely back them up."
echo ""
