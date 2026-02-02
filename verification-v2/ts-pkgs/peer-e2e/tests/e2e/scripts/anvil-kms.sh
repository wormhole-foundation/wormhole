#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

docker build --platform linux/amd64 --tag anvil-for-kms --file ./anvil/kms.Dockerfile --progress=plain ../../../..

docker run --rm --network=dkg-test --name anvil-with-verifier -p 8545:8545 anvil-for-kms

