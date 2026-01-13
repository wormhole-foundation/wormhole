#!/bin/bash
# Start the peer discovery server for DKG coordination.
# Usage: ./run-peer-server.sh <SERVER_PORT> <ETHEREUM_RPC_URL> [WORMHOLE_ADDRESS]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

log_error() { echo "[ERROR] $1"; }

if [ $# -lt 2 ]; then
    echo "Usage: $0 <SERVER_PORT> <ETHEREUM_RPC_URL> [WORMHOLE_ADDRESS]"
    echo ""
    echo "Arguments:"
    echo "  SERVER_PORT       - Port for the peer server to listen on"
    echo "  ETHEREUM_RPC_URL  - Ethereum mainnet RPC URL"
    echo "  WORMHOLE_ADDRESS  - (Optional) Wormhole contract address"
    exit 1
fi

SERVER_PORT="$1"
ETHEREUM_RPC_URL="$2"
WORMHOLE_ADDRESS="${3:-0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B}"

if ! [[ "$SERVER_PORT" =~ ^[0-9]+$ ]]; then
    log_error "SERVER_PORT must be a number"
    exit 1
fi

docker build \
    --tag peer-server \
    --file "${REPO_ROOT}/ts-pkgs/peer-server/Dockerfile" \
    --build-arg SERVER_PORT="${SERVER_PORT}" \
    --build-arg ETHEREUM_RPC_URL="${ETHEREUM_RPC_URL}" \
    --build-arg WORMHOLE_ADDRESS="${WORMHOLE_ADDRESS}" \
    --progress=plain \
    "${REPO_ROOT}"

docker run \
    --rm \
    --name peer-server \
    --publish "${SERVER_PORT}:${SERVER_PORT}" \
    --mount type=bind,src="$(pwd)",dst=/output \
    peer-server

