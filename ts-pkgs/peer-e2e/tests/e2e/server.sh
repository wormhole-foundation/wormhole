#!/bin/bash

set -meuo pipefail

SERVER_PORT="3000"
ETHEREUM_RPC_URL="http://127.0.0.1:8545"
WORMHOLE_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"

DOCKER_BUILDKIT=1 docker build -t peer-server -f ../../../peer-server/Dockerfile --build-arg SERVER_PORT=${SERVER_PORT} --build-arg ETHEREUM_RPC_URL=${ETHEREUM_RPC_URL} --build-arg WORMHOLE_ADDRESS=${WORMHOLE_ADDRESS} --progress=plain .
docker run --network="host" --name peer-server peer-server
