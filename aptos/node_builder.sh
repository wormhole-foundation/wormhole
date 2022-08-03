#!/bin/bash

set -x

git clone https://github.com/aptos-labs/aptos-core.git
cd aptos-core
./scripts/dev_setup.sh -b

cd aptos-node
CARGO_NET_GIT_FETCH_WITH_CLI=true cargo build
cd ../crates/aptos-faucet
CARGO_NET_GIT_FETCH_WITH_CLI=true cargo build
