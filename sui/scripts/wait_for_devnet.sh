#!/bin/bash

set -e

# Wait for sui to start
while [[ "$(curl -X POST -H "Content-Type: application/json" -d '{ "jsonrpc":"2.0", "method":"rpc.discover","id":1 }' -s -o /dev/null -w '%{http_code}' 0.0.0.0:9000/)" != "200" ]]; do sleep 1; done

# Wait for sui-faucet to start
while [[ "$(curl -s -o /dev/null -w '%{http_code}' 0.0.0.0:5003/)" != "200" ]]; do sleep 1; done
