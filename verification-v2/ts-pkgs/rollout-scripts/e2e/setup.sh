#!/bin/bash

set -meuo pipefail

# Create a network for the guardians to comunicate
docker network create dkg-test
# Create a builder that connects directly to the dkg-test network so it can be accessed during build.
docker buildx create --name dkg-builder --driver docker-container --driver-opt network=dkg-test
