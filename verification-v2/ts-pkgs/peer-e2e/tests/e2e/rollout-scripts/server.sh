#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

SERVER_PORT="3000"
ETHEREUM_RPC_URL="http://anvil-with-verifier:8545"
WORMHOLE_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"

export DOCKER_NETWORK="dkg-test"
../../../../rollout-scripts/run-peer-server.sh "${SERVER_PORT}" "${ETHEREUM_RPC_URL}" "${WORMHOLE_ADDRESS}"

