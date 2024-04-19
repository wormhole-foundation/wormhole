#!/bin/bash

# MNEMONIC=<redacted> ./sh/deployCoreShutdown.sh

. .env

[[ -z $INIT_EVM_CHAIN_ID ]] && { echo "Missing INIT_EVM_CHAIN_ID"; exit 1; }

[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }
[[ -z $RPC_URL ]] && { echo "Missing RPC_URL"; exit 1; }

forge script ./forge-scripts/DeployCoreShutdown.s.sol:DeployCoreShutdown \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast ${FORGE_ARGS}

returnInfo=$(cat ./broadcast/DeployCoreShutdown.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
SHUTDOWN_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

echo "Wormhole Core Shutdown address: $SHUTDOWN_ADDRESS"
