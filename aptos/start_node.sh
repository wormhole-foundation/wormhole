#!/bin/bash

set -x

cd aptos-core/aptos-node
CARGO_NET_GIT_FETCH_WITH_CLI=true cargo run  -- --test
