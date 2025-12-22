#!/bin/bash

set -meuo pipefail

rm -rf out/

# These scripts are meant to be run with this directory as the working directory

./scripts/clean.sh
./scripts/setup.sh

# TODO: There is technically a race condition between these, use docker inspect?
./scripts/anvil.sh &
sleep 1
./scripts/server.sh &
sleep 1

./scripts/client.sh
./scripts/clean.sh
