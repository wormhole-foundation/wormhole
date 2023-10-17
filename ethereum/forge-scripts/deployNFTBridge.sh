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
prompt_for_variable "BRIDGE_INIT_FINALITY" 'Please provide the initial finality value' 1

# Step 1: Run 'forge script DeployNFTBridge', get the JSON output from the specified file
FOUNDRY_PROFILE=production forge script ./forge-scripts/DeployNFTBridge.s.sol:DeployNFTBridge --sig "run(uint16,uint16,bytes32,uint8,uint256,address)" $INIT_CHAIN_ID $INIT_GOV_CHAIN_ID $INIT_GOV_CONTRACT $BRIDGE_INIT_FINALITY $INIT_EVM_CHAIN_ID $WORMHOLE_ADDRESS --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast
returnInfo=$(cat ./broadcast/DeployNFTBridge.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
NFT_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
NFT_IMPLEMENTATION_ADDRESS=$(jq -r '.returns.nftImplementationAddress.value' <<< "$returnInfo")
NFT_BRIDGE_SETUP_ADDRESS=$(jq -r '.returns.setupAddress.value' <<< "$returnInfo")
NFT_BRIDGE_IMPLEMENTATION_ADDRESS=$(jq -r '.returns.implementationAddress.value' <<< "$returnInfo")

echo "Deployed NFT bridge address: $NFT_BRIDGE_ADDRESS"

# Get the current time as a Unix timestamp
current_time=$(date +%s)

mkdir -p "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/NFTBridge/"

json="{\"NFT_BRIDGE_ADDRESS\":\"$NFT_BRIDGE_ADDRESS\", \"NFT_BRIDGE_SETUP_ADDRESS\":\"$NFT_BRIDGE_SETUP_ADDRESS\", \"NFT_BRIDGE_IMPLEMENTATION_ADDRESS\":\"$NFT_BRIDGE_IMPLEMENTATION_ADDRESS\", \"NFT_IMPLEMENTATION_ADDRESS\":\"$NFT_IMPLEMENTATION_ADDRESS\"}"

echo "$json" > "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/NFTBridge/latest.json"
echo "$json" > "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/NFTBridge/$current_time.json"