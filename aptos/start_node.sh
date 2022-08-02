#!/bin/bash

set -x

cd aptos-core
CARGO_NET_GIT_FETCH_WITH_CLI=true cargo run -p aptos-node -- --test
