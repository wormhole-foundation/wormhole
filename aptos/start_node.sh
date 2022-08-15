#!/bin/bash

set -x

cd aptos-core/
target/debug/aptos-node --test --test-dir "/tmp/foo"&
sleep 5
target/debug/aptos-faucet --chain-id TESTING  --mint-key-file-path "/tmp/foo/mint.key" --address 0.0.0.0 --port 8000 --server-url http://127.0.0.1:8080
