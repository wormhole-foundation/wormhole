#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

SERVER_PORT="3000"
ETHEREUM_RPC_URL="http://anvil-with-verifier:8545"
WORMHOLE_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"

docker build --tag peer-server --file ../../../peer-server/Dockerfile --build-arg SERVER_PORT=${SERVER_PORT} --build-arg ETHEREUM_RPC_URL=${ETHEREUM_RPC_URL} --build-arg WORMHOLE_ADDRESS=${WORMHOLE_ADDRESS} --progress=plain .

docker run --rm --network=dkg-test --name peer-server peer-server
