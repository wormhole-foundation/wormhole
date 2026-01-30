#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

docker build --platform linux/amd64 --tag anvil-with-verifier --file ./anvil/Dockerfile --progress=plain ../../../..

docker run --rm --network=dkg-test --name anvil-with-verifier anvil-with-verifier

