#!/usr/bin/env bash
set -euo pipefail

# TODO(leo): remove after a while
rm -rf bridge

rm -rf node/pkg/proto
DOCKER_BUILDKIT=1 tilt docker build -- --target go-export -f Dockerfile.proto -o type=local,dest=node .
