#!/bin/bash

set -meuo pipefail

docker network create dkg-test
docker buildx create --name dkg-builder --driver docker-container --driver-opt network=dkg-test

