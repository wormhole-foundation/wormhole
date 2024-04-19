#!/bin/bash

# MNEMONIC=<redacted> ./sh/deployCoreBridge.sh

. .env

[[ -z $INIT_SIGNERS ]] && { echo "Missing INIT_SIGNERS"; exit 1; }
[[ -z $INIT_CHAIN_ID ]] && { echo "Missing INIT_CHAIN_ID"; exit 1; }
[[ -z $INIT_GOV_CHAIN_ID ]] && { echo "Missing INIT_GOV_CHAIN_ID"; exit 1; }
[[ -z $INIT_GOV_CONTRACT ]] && { echo "Missing INIT_GOV_CONTRACT"; exit 1; }
[[ -z $INIT_EVM_CHAIN_ID ]] && { echo "Missing INIT_EVM_CHAIN_ID"; exit 1; }

[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }
[[ -z $RPC_URL ]] && { echo "Missing RPC_URL"; exit 1; }

forge script ./forge-scripts/DeployCore.s.sol:DeployCore \
	--sig "run(address[],uint16,uint16,bytes32,uint256)" $INIT_SIGNERS $INIT_CHAIN_ID $INIT_GOV_CHAIN_ID $INIT_GOV_CONTRACT $INIT_EVM_CHAIN_ID \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast ${FORGE_ARGS}

returnInfo=$(cat ./broadcast/DeployCore.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
WORMHOLE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
SETUP_ADDRESS=$(jq -r '.returns.setupAddress.value' <<< "$returnInfo")
IMPLEMENTATION_ADDRESS=$(jq -r '.returns.implAddress.value' <<< "$returnInfo")

echo "-- Wormhole Core Addresses --------------------------------------------------"
echo "| Setup address                | $SETUP_ADDRESS |"
echo "| Implementation address       | $IMPLEMENTATION_ADDRESS |"
echo "| Wormhole address             | $WORMHOLE_ADDRESS |"
echo "-----------------------------------------------------------------------------"