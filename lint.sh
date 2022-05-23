#!/usr/bin/env bash

ls -al $(pwd)/node

docker run --mount=type=bind,target=/app,source=$(pwd)/node --workdir /app $(DOCKER_BUILDKIT=1 docker build -q -f Dockerfile.lint .) sh -c 'GOGC=off /home/lint/golangci-lint run --skip-dirs pkg/supervisor --timeout=10m  --out-format=github-actions ./...'
