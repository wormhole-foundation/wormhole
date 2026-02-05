#!/bin/bash

set -meuo pipefail

if [[ -z "${GITHUB_ACTIONS:-}" ]]; then
    # Create a network for the guardians to comunicate
    docker network create dkg-test
fi