#!/bin/bash

# Source the prompt_functions.sh file
source "./forge-scripts/prompt_functions.sh"

# Obtain environment variables
prompt_for_variable "RPC_URL" "Please provide the RPC URL"
prompt_for_variable "PRIVATE_KEY" "Please provide your private key" 
prompt_for_variable "INIT_EVM_CHAIN_ID" "Please provide the EVM Chain ID (e.g. 1 for Ethereum)"
prompt_for_variable "INIT_CHAIN_ID" "Please provide the Wormhole Chain ID (e.g. 2 for Ethereum)"
prompt_for_variable "NETWORK" "Please provide the network (e.g. 'mainnet', 'testnet', 'devnet', 'ci')"

prompt_for_variable "INIT_SIGNERS" 'Please provide the array of initial guardian signers (e.g. ["0x1234", "0x5678"])'
prompt_for_variable "INIT_GOV_CHAIN_ID" 'Please provide the initial governance chain id' 1
prompt_for_variable "INIT_GOV_CONTRACT" 'Please provide the initial governance contract address' 0x0000000000000000000000000000000000000000000000000000000000000004

# Step 1: Run 'forge script DeployCore', get the JSON output from the specified file
forge script ./forge-scripts/DeployCore.s.sol:DeployCore --sig "run(address[],uint16,uint16,bytes32,uint256)" $INIT_SIGNERS $INIT_CHAIN_ID $INIT_GOV_CHAIN_ID $INIT_GOV_CONTRACT $INIT_EVM_CHAIN_ID --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast --via-ir
returnInfo=$(cat ./broadcast/DeployCore.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
WORMHOLE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
SETUP_ADDRESS=$(jq -r '.returns.setupAddress.value' <<< "$returnInfo")
IMPLEMENTATION_ADDRESS=$(jq -r '.returns.implAddress.value' <<< "$returnInfo")

echo "Deployed wormhole address: $WORMHOLE_ADDRESS"

# Get the current time as a Unix timestamp
current_time=$(date +%s)

mkdir -p "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/CoreBridge/"

json="{\"WORMHOLE_ADDRESS\":\"$WORMHOLE_ADDRESS\", \"SETUP_ADDRESS\":\"$SETUP_ADDRESS\", \"IMPLEMENTATION_ADDRESS\":\"$IMPLEMENTATION_ADDRESS\"}"

echo "$json" > "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/CoreBridge/latest.json"
echo "$json" > "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/CoreBridge/$current_time.json"