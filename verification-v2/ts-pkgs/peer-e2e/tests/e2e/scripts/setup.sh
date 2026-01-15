#!/bin/bash

set -meuo pipefail

if [[ -z "${GITHUB_ACTIONS:-}" ]]; then
    # Create a network for the guardians to comunicate
    docker network create dkg-test
    # Create a builder that connects directly to the dkg-test network so it can be accessed during build.
    docker buildx create --name dkg-builder --driver docker-container --driver-opt network=dkg-test
fi