#!/bin/bash

set -meuo pipefail

rm -rf out/

# These scripts are meant to be run with this directory as the working directory

if [[ -z "${GITHUB_ACTIONS:-}" ]]; then
    ./rollout-scripts/clean.sh
fi
./rollout-scripts/setup.sh
./rollout-scripts/anvil.sh &
./rollout-scripts/server.sh &
./rollout-scripts/client.sh
./rollout-scripts/clean.sh

# Wait for anvil and server subshells to check that their exit codes are zero
wait