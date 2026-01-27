#!/bin/bash
# Start the peer discovery server for DKG coordination.
# Usage: ./run-peer-server.sh <SERVER_PORT> <ETHEREUM_RPC_URL> [WORMHOLE_ADDRESS]

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
REPO_ROOT="${SCRIPT_DIR}/../.."

# TODO: add argument for peer server output directory
if [ $# -lt 2 ]; then
    echo "Usage: $0 <SERVER_PORT> <ETHEREUM_RPC_URL> [WORMHOLE_ADDRESS]"
    echo ""
    echo "Arguments:"
    echo "  SERVER_PORT       - Port for the peer server to listen on"
    echo "  ETHEREUM_RPC_URL  - Ethereum mainnet RPC URL"
    echo "  WORMHOLE_ADDRESS  - (Only set for testnet networks) Wormhole contract address"
    exit 1
fi

SERVER_PORT="$1"
ETHEREUM_RPC_URL="$2"
WORMHOLE_ADDRESS="${3:-0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B}"

# For local testing: use DOCKER_NETWORK env var to join a specific network
# Do NOT use in production
DOCKER_NETWORK="${DOCKER_NETWORK:-}"

if [ -n "${DOCKER_NETWORK}" ]; then
    NETWORK_FLAG="--network=${DOCKER_NETWORK}"
else
    NETWORK_FLAG=""
fi

docker build \
    --tag peer-server \
    --file "${REPO_ROOT}/ts-pkgs/peer-server/Dockerfile" \
    --build-arg SERVER_PORT="${SERVER_PORT}" \
    --build-arg ETHEREUM_RPC_URL="${ETHEREUM_RPC_URL}" \
    --build-arg WORMHOLE_ADDRESS="${WORMHOLE_ADDRESS}" \
    "${REPO_ROOT}"

docker run \
    --interactive \
    --tty \
    --rm \
    --name peer-server \
    --publish "${SERVER_PORT}:${SERVER_PORT}" \
    --mount type=bind,src="${SCRIPT_DIR}",dst=/output \
    ${NETWORK_FLAG} \
    peer-server

