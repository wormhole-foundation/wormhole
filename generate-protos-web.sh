#!/usr/bin/env bash

rm -rf explorer/src/proto sdk/js/src/proto

DOCKER_BUILDKIT=1 tilt docker build -- --target node-export -f Dockerfile.proto -o type=local,dest=. .
