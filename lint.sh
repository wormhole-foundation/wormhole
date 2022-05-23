#!/usr/bin/env bash

# fail if any command fails
set -e
set -o pipefail

# we duplicate stderr to stdout and then filter and parse stdout to only include errors that are readable as github annotations
DOCKER_BUILDKIT=1 docker build -f Dockerfile.lint . 2>&1  | while read line; do echo $line; echo $line >&2; done | (grep "::" || true) | cut -f3- -d " "
