#!/bin/bash
# Start the peer discovery server for DKG coordination.
# Usage: ./run-peer-server.sh <SERVER_PORT> <ETHEREUM_RPC_URL> <OUTPUT_DIRECTORY> [WORMHOLE_ADDRESS]

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
REPO_ROOT="${SCRIPT_DIR}/../.."

# TODO: add argument for peer server output directory
if [ $# -lt 3 ]; then
    echo "Usage: $0 <SERVER_PORT> <ETHEREUM_RPC_URL> <OUTPUT_DIRECTORY> [WORMHOLE_ADDRESS]"
    echo ""
    echo "Arguments:"
    echo "  SERVER_PORT       - Port for the peer server to listen on"
    echo "  ETHEREUM_RPC_URL  - Ethereum mainnet RPC URL"
    echo "  OUTPUT_DIRECTORY  - Output directory where peers will be stored"
    echo "  WORMHOLE_ADDRESS  - (Only set for testnet networks) Wormhole contract address"
    exit 1
fi

SERVER_PORT="$1"
ETHEREUM_RPC_URL="$2"
OUTPUT_DIRECTORY="$3"
WORMHOLE_ADDRESS="${4:-0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B}"

mkdir -p "$OUTPUT_DIRECTORY"

# TSS_E2E_DOCKER_NETWORK should NOT be used in production
if [ -n "${TSS_E2E_DOCKER_NETWORK:-}" ]; then
    network_option="--network=${TSS_E2E_DOCKER_NETWORK}"
    publish_options=""
else
    publish_options="--publish ${SERVER_PORT}:${SERVER_PORT}"
    network_option=""
fi

docker build \
    --tag peer-server \
    --file "${REPO_ROOT}/ts-pkgs/peer-server/Dockerfile" \
    --build-arg SERVER_PORT="${SERVER_PORT}" \
    --build-arg ETHEREUM_RPC_URL="${ETHEREUM_RPC_URL}" \
    --build-arg WORMHOLE_ADDRESS="${WORMHOLE_ADDRESS}" \
    "${REPO_ROOT}"

interactive_options="--interactive --tty"
if [ -n "${NON_INTERACTIVE}" ]; then
    interactive_options=""
fi

docker run \
    ${interactive_options} \
    --rm \
    --name peer-server \
    ${publish_options} \
    --mount type=bind,src="${OUTPUT_DIRECTORY}",dst=/output \
    "${network_option}" \
    peer-server

