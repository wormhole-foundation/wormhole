#!/bin/bash

#
# This script deploys the DelegatedManagerSet contract.
# Usage: RPC_URL= MNEMONIC= EVM_CHAIN_ID= WORMHOLE_ADDRESS= ./sh/deployDelegatedManagerSet.sh
#  tilt: WORMHOLE_ADDRESS=0xC89Ce4735882C9F0f0FE26686c53074E09B0D550 ./sh/deployDelegatedManagerSet.sh
#  anvil: EVM_CHAIN_ID=31337 MNEMONIC=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 WORMHOLE_ADDRESS=0x... ./sh/deployDelegatedManagerSet.sh

# Example verification for Sepolia
# forge verify-contract 0x086a699900262D829512299ABe07648870000Dd1 ./contracts/delegated_manager_set/DelegatedManagerSet.sol:DelegatedManagerSet --watch --chain-id 11155111 --etherscan-api-key <YOUR_ETHERSCAN_KEY> --constructor-args $(cast abi-encode "constructor(address)" "0x4a8bc80Ed5a4067f1CCf107057b8270E0cC11A78")

set -euo pipefail

if [ "${WORMHOLE_ADDRESS:-X}" == "X" ]; then
  echo "Error: WORMHOLE_ADDRESS environment variable is required"
  echo "Usage: WORMHOLE_ADDRESS=0x... ./sh/deployDelegatedManagerSet.sh"
  exit 1
fi

if [ "${RPC_URL:-X}" == "X" ]; then
  RPC_URL=http://localhost:8545
fi

if [ "${MNEMONIC:-X}" == "X" ]; then
  MNEMONIC=0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d
fi

if [ "${EVM_CHAIN_ID:-X}" == "X" ]; then
  EVM_CHAIN_ID=1337
fi

echo "Deploying DelegatedManagerSet..."
echo "  RPC_URL: $RPC_URL"
echo "  EVM_CHAIN_ID: $EVM_CHAIN_ID"
echo "  WORMHOLE_ADDRESS: $WORMHOLE_ADDRESS"

forge script ./forge-scripts/DeployDelegatedManagerSet.s.sol:DeployDelegatedManagerSet \
	--sig "run(address)" "$WORMHOLE_ADDRESS" \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast ${FORGE_ARGS:-}

returnInfo=$(cat ./broadcast/DeployDelegatedManagerSet.s.sol/$EVM_CHAIN_ID/run-latest.json)

DEPLOYED_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
echo "Deployed DelegatedManagerSet to address: $DEPLOYED_ADDRESS"
