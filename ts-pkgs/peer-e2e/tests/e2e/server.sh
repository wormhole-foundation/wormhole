#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

SERVER_PORT="3000"
ETHEREUM_RPC_URL="http://anvil-with-verifier:8545"
WORMHOLE_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"

docker stop anvil-with-verifier peer-server || true
docker rm anvil-with-verifier peer-server || true
docker network create dkg-test || true

docker build -t anvil-with-verifier -f ./Dockerfile --progress=plain .

docker run --rm --network=dkg-test --name anvil-with-verifier anvil-with-verifier &

docker build -t peer-server -f ../../../peer-server/Dockerfile --build-arg SERVER_PORT=${SERVER_PORT} --build-arg ETHEREUM_RPC_URL=${ETHEREUM_RPC_URL} --build-arg WORMHOLE_ADDRESS=${WORMHOLE_ADDRESS} --progress=plain .

until docker run --rm --network=dkg-test --name peer-server peer-server
do
    echo "Server probably failed to connect to anvil, trying again..."
    sleep 1
done
