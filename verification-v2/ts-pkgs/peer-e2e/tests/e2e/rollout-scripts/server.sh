#!/bin/bash

set -euo pipefail
export DOCKER_BUILDKIT=1
export NON_INTERACTIVE=1

SERVER_PORT="3000"
ETHEREUM_RPC_URL="http://anvil-with-verifier:8545"
OUTPUT_DIRECTORY=./out/server
WORMHOLE_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"

# Wait until anvil starts listening
docker run --rm --network=dkg-test --env "ETHEREUM_RPC_URL=$ETHEREUM_RPC_URL" --env "WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS" ghcr.io/foundry-rs/foundry:v1.5.1@sha256:3a70bfa9bd2c732a767bb60d12c8770b40e8f9b6cca28efc4b12b1be81c7f28e '
  start=$(date +%s)
  deadline=$((start+60))
  until cast call --rpc-url "$ETHEREUM_RPC_URL" "$WORMHOLE_ADDRESS" "getCurrentGuardianSetIndex()" >/dev/null 2>&1; do
    now=$(date +%s)
    if [ "$now" -ge "$deadline" ]; then
      echo "Timed out waiting for $ETHEREUM_RPC_URL" >&2
      exit 1
    fi
    sleep 0.5
  done
'

export TSS_E2E_DOCKER_NETWORK="dkg-test"
../../../rollout-scripts/run-peer-server.sh "${SERVER_PORT}" "${ETHEREUM_RPC_URL}" "${OUTPUT_DIRECTORY}" "${WORMHOLE_ADDRESS}"

