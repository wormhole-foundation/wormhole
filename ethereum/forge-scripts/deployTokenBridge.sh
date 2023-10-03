#!/bin/bash

# Source the prompt_functions.sh file
source "./forge-scripts/prompt_functions.sh"

# Obtain environment variables
prompt_for_variable "RPC_URL" "Please provide the RPC URL"
prompt_for_variable "PRIVATE_KEY" "Please provide your private key"
prompt_for_variable "INIT_EVM_CHAIN_ID" "Please provide the EVM Chain ID (e.g. 1 for Ethereum)"
prompt_for_variable "INIT_CHAIN_ID" "Please provide the Wormhole Chain ID (e.g. 2 for Ethereum)"
prompt_for_variable "NETWORK" "Please provide the network (e.g. 'mainnet', 'testnet', 'devnet', 'ci')"
prompt_for_variable "WORMHOLE_ADDRESS" "Please provide the address of the core bridge on this chain"

# Step 1: Run 'forge script DeployTokenBridge', get the JSON output from the specified file
forge script ./forge-scripts/DeployTokenBridge.s.sol:DeployTokenBridge --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast 
returnInfo=$(cat ./broadcast/DeployTokenBridge.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
TOKEN_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
TOKEN_IMPLEMENTATION_ADDRESS=$(jq -r '.returns.tokenImplementationAddress.value' <<< "$returnInfo")
TOKEN_BRIDGE_SETUP_ADDRESS=$(jq -r '.returns.bridgeSetupAddress.value' <<< "$returnInfo")
TOKEN_BRIDGE_IMPLEMENTATION_ADDRESS=$(jq -r '.returns.bridgeImplementationAddress.value' <<< "$returnInfo")

echo "Deployed token bridge address: $TOKEN_BRIDGE_ADDRESS"

# Get the current time as a Unix timestamp
current_time=$(date +%s)

mkdir -p "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/TokenBridge/"

json="{\"TOKEN_BRIDGE_ADDRESS\":\"$TOKEN_BRIDGE_ADDRESS\", \"TOKEN_BRIDGE_SETUP_ADDRESS\":\"$TOKEN_BRIDGE_SETUP_ADDRESS\", \"TOKEN_BRIDGE_IMPLEMENTATION_ADDRESS\":\"$TOKEN_BRIDGE_IMPLEMENTATION_ADDRESS\", \"TOKEN_IMPLEMENTATION_ADDRESS\":\"$TOKEN_IMPLEMENTATION_ADDRESS\"}"

echo "$json" > "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/TokenBridge/latest.json"
echo "$json" > "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/TokenBridge/$current_time.json"