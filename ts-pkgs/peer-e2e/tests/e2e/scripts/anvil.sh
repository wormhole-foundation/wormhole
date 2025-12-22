#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

docker build -t anvil-with-verifier -f ./anvil/Dockerfile --progress=plain ./anvil

docker run --rm --network=dkg-test --name anvil-with-verifier anvil-with-verifier
