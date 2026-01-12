#!/bin/bash

set -meuo pipefail

rm -rf out/

# These scripts are meant to be run with this directory as the working directory

./scripts/clean.sh
./scripts/setup.sh
./scripts/anvil.sh &
./scripts/server.sh &
./scripts/client.sh
./scripts/clean.sh
