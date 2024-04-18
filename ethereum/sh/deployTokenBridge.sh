#!/bin/bash

# MNEMONIC=<redacted> WORMHOLE_ADDRESS=<from_the_previous_command> ./sh/deployTokenBridge.sh

. .env

[[ -z $INIT_EVM_CHAIN_ID ]] && { echo "Missing INIT_EVM_CHAIN_ID"; exit 1; }

[[ -z $BRIDGE_INIT_CHAIN_ID ]] && { echo "Missing BRIDGE_INIT_CHAIN_ID"; exit 1; }
[[ -z $BRIDGE_INIT_GOV_CHAIN_ID ]] && { echo "Missing BRIDGE_INIT_GOV_CHAIN_ID"; exit 1; }
[[ -z $BRIDGE_INIT_GOV_CONTRACT ]] && { echo "Missing BRIDGE_INIT_GOV_CONTRACT"; exit 1; }
[[ -z $BRIDGE_INIT_WETH ]] && { echo "Missing BRIDGE_INIT_WETH"; exit 1; }
[[ -z $BRIDGE_INIT_FINALITY ]] && { echo "Missing BRIDGE_INIT_FINALITY"; exit 1; }

[[ -z $WORMHOLE_ADDRESS ]] && { echo "Missing WORMHOLE_ADDRESS"; exit 1; }

[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }
[[ -z $RPC_URL ]] && { echo "Missing RPC_URL"; exit 1; }

forge script ./forge-scripts/DeployTokenBridge.s.sol:DeployTokenBridge \
	--sig "run(uint16,uint16,bytes32,address,uint8,uint256,address)" $BRIDGE_INIT_CHAIN_ID $BRIDGE_INIT_GOV_CHAIN_ID $BRIDGE_INIT_GOV_CONTRACT $BRIDGE_INIT_WETH $BRIDGE_INIT_FINALITY $INIT_EVM_CHAIN_ID $WORMHOLE_ADDRESS \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast ${FORGE_ARGS}

returnInfo=$(cat ./broadcast/DeployTokenBridge.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
TOKEN_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
TOKEN_IMPLEMENTATION_ADDRESS=$(jq -r '.returns.tokenImplementationAddress.value' <<< "$returnInfo")
TOKEN_BRIDGE_SETUP_ADDRESS=$(jq -r '.returns.bridgeSetupAddress.value' <<< "$returnInfo")
TOKEN_BRIDGE_IMPLEMENTATION_ADDRESS=$(jq -r '.returns.bridgeImplementationAddress.value' <<< "$returnInfo")

echo "-- TokenBridge Addresses ----------------------------------------------------"
echo "| Token Implementation address | $TOKEN_IMPLEMENTATION_ADDRESS |"
echo "| BridgeSetup address          | $TOKEN_BRIDGE_SETUP_ADDRESS |"
echo "| BridgeImplementation address | $TOKEN_BRIDGE_IMPLEMENTATION_ADDRESS |"
echo "| TokenBridge address          | $TOKEN_BRIDGE_ADDRESS |"
echo "-----------------------------------------------------------------------------"
