#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

SERVER_PORT="3000"
ETHEREUM_RPC_URL="http://anvil-with-verifier:8545"
WORMHOLE_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"

docker build --tag peer-server \
  --file ../../../peer-server/Dockerfile \
  --build-arg SERVER_PORT=${SERVER_PORT} \
  --build-arg ETHEREUM_RPC_URL=${ETHEREUM_RPC_URL} \
  --build-arg WORMHOLE_ADDRESS=${WORMHOLE_ADDRESS} \
  --progress=plain ../../../..

# Wait until anvil starts listening
docker run --rm --network=dkg-test --env "ETHEREUM_RPC_URL=$ETHEREUM_RPC_URL" --env "WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS" ghcr.io/foundry-rs/foundry:v1.5.1@sha256:3a70bfa9bd2c732a767bb60d12c8770b40e8f9b6cca28efc4b12b1be81c7f28e '
  deadline=$((SECONDS+60))
  until cast call --rpc-url "$ETHEREUM_RPC_URL" "$WORMHOLE_ADDRESS" "getCurrentGuardianSetIndex()"; do
    if [ "$SECONDS" -ge "$deadline" ]; then
      echo "Timed out waiting for $ETHEREUM_RPC_URL" >&2
      exit 1
    fi
    sleep 0.5
  done
'


docker run --rm --network=dkg-test --name peer-server peer-server
