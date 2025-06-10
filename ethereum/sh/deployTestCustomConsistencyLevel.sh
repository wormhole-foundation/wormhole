#!/bin/bash

#
# This script deploys the CustomConsistencyLevel contract.
# Usage: RPC_URL= MNEMONIC= EVM_CHAIN_ID= WORMHOLE_ADDRESS= CUSTOM_CONSISTENCY_LEVEL= ./sh/deployCustomConsistencyLevel.sh
#  tilt: ./sh/deployCustomConsistencyLevel.sh
#  anvil: EVM_CHAIN_ID=31337 MNEMONIC=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 WORMHOLE_ADDRESS= CUSTOM_CONSISTENCY_LEVEL= ./sh/deployCustomConsistencyLevel.sh

if [ "${RPC_URL}X" == "X" ]; then
  RPC_URL=http://localhost:8545
fi

if [ "${MNEMONIC}X" == "X" ]; then
  MNEMONIC=0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d
fi

if [ "${EVM_CHAIN_ID}X" == "X" ]; then
  EVM_CHAIN_ID=1337
fi

[[ -z $WORMHOLE_ADDRESS ]] && { echo "Missing WORMHOLE_ADDRESS"; exit 1; }
[[ -z $CUSTOM_CONSISTENCY_LEVEL ]] && { echo "Missing CUSTOM_CONSISTENCY_LEVEL"; exit 1; }

forge script ./forge-scripts/DeployTestCustomConsistencyLevel.s.sol:DeployTestCustomConsistencyLevel \
	--sig "run(address,address)" $WORMHOLE_ADDRESS $CUSTOM_CONSISTENCY_LEVEL \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast ${FORGE_ARGS}

returnInfo=$(cat ./broadcast/DeployTestCustomConsistencyLevel.s.sol/$EVM_CHAIN_ID/run-latest.json)

DEPLOYED_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
echo "Deployed test custom consistency level to address: $DEPLOYED_ADDRESS using core address $WORMHOLE_ADDRESS and custom consistency level $CUSTOM_CONSISTENCY_LEVEL"

echo "Configuring test custom consistency level at address $DEPLOYED_ADDRESS"
forge script ./forge-scripts/ConfigureTestCustomConsistencyLevel.s.sol:ConfigureTestCustomConsistencyLevel \
	--sig "run(address)" $DEPLOYED_ADDRESS \
	--rpc-url $RPC_URL \
	--private-key $MNEMONIC \
	--broadcast

	echo "Configured test custom consistency level at address $DEPLOYED_ADDRESS"
