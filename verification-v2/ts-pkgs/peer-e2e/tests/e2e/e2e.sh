#!/bin/bash

set -meuo pipefail

rm -rf out/

# These scripts are meant to be run with this directory as the working directory

if [[ -z "${GITHUB_ACTIONS:-}" ]]; then
    ./scripts/clean.sh
fi
./scripts/setup.sh
./scripts/anvil.sh &
./scripts/server.sh &
./scripts/client.sh
./scripts/clean.sh

# Wait for anvil and server subshells to check that their exit codes are zero
wait