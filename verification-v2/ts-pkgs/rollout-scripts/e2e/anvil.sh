#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

docker build --tag anvil-with-verifier --file ../../peer-e2e/tests/e2e/anvil/Dockerfile --progress=plain ../../../..

docker run --rm --network=dkg-test --name anvil-with-verifier anvil-with-verifier
