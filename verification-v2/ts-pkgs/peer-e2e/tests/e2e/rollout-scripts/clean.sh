#!/bin/bash

set -meuo pipefail

echo "Cleaning up Docker containers, images, builders and networks..."

echo "Stopping containers..."
for i in $(seq 0 18)
do
  docker stop "Guardian$i" 2>/dev/null &
done

docker stop anvil-with-verifier peer-server 2>/dev/null &

wait

echo "Removing builder..."
docker buildx rm dkg-builder 2>/dev/null || true

echo "Removing network..."
docker network rm dkg-test 2>/dev/null || true

echo "Done!"
