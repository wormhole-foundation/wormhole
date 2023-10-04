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

prompt_for_variable "INIT_GOV_CHAIN_ID" 'Please provide the initial governance chain id' 1
prompt_for_variable "INIT_GOV_CONTRACT" 'Please provide the initial governance contract address' 0x0000000000000000000000000000000000000000000000000000000000000004
prompt_for_variable "BRIDGE_INIT_WETH" 'Please provide the WETH address'
prompt_for_variable "BRIDGE_INIT_FINALITY" 'Please provide the initial finality value' 1

# Step 1: Run 'forge script DeployTokenBridge', get the JSON output from the specified file
forge script ./forge-scripts/DeployTokenBridge.s.sol:DeployTokenBridge --sig "run(uint16,uint16,bytes32,address,uint8,uint256,address)" $INIT_CHAIN_ID $INIT_GOV_CHAIN_ID $INIT_GOV_CONTRACT $BRIDGE_INIT_WETH $BRIDGE_INIT_FINALITY $INIT_EVM_CHAIN_ID $WORMHOLE_ADDRESS --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast --via-ir
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