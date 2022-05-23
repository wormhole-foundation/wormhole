#!/usr/bin/env bash
set -e
set -o pipefail
DOCKER_BUILDKIT=1 docker build -f Dockerfile.lint . 2>&1  | while read line; do echo $line; echo $line >&2; done | (grep "::" || true) | cut -f3- -d " "
