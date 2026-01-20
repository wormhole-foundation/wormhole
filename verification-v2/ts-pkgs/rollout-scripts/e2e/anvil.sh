#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

docker build --platform linux/amd64 --tag anvil-with-verifier --file ../../peer-e2e/tests/e2e/anvil/Dockerfile --progress=plain ../../..

docker run -it --rm --network=dkg-test --name anvil-with-verifier anvil-with-verifier
