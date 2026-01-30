#!/bin/bash
# Run the DKG ceremony. Polls for all peers, then generates key shards.
# Usage: ./run-dkg.sh <TLS_KEYS_DIR> <TLS_HOSTNAME> <TLS_PORT> <PEER_SERVER_URL> <ETHEREUM_RPC_URL> [WORMHOLE_ADDRESS]

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
REPO_ROOT="${SCRIPT_DIR}/../.."

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

# TSS_E2E_DOCKER_NETWORK should NOT be used in production
network_option=""
publish_option="--publish ${TLS_PORT}:${TLS_PORT}"
if [ -n "${TSS_E2E_DOCKER_NETWORK:-}" ]; then
    network_option="--network=${TSS_E2E_DOCKER_NETWORK}"
    publish_option=""
fi

wormhole_option=""
if [ -n "${WORMHOLE_ADDRESS}" ]; then
    wormhole_option="--env WORMHOLE_CONTRACT_ADDRESS=${WORMHOLE_ADDRESS}"
fi

docker build --tag dkg-client --file "${REPO_ROOT}/ts-pkgs/peer-client/dkg.Dockerfile" "${REPO_ROOT}"

docker run \
    --rm \
    --name "${TLS_HOSTNAME}" \
    ${network_option} \
    ${publish_option} \
    --mount type=bind,src="${TLS_KEYS_DIR}",dst=/keys \
    --env TLS_HOSTNAME="${TLS_HOSTNAME}" \
    --env TLS_PORT="${TLS_PORT}" \
    --env PEER_SERVER_URL="${PEER_SERVER_URL}" \
    --env ETHEREUM_RPC_URL="${ETHEREUM_RPC_URL}" \
    ${wormhole_option} \
    dkg-client


echo ""
echo "=============================================="
echo "DKG CEREMONY COMPLETED SUCCESSFULLY"
echo "=============================================="
echo ""
echo "Your DKG key shards have been generated and saved to ${TLS_KEYS_DIR}."
echo ""
echo "Please verify the generated files and securely back them up."
echo ""
