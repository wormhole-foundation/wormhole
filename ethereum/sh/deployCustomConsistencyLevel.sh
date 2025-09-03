#!/bin/bash

#
# This script deploys the CustomConsistencyLevel contract.
# Usage: RPC_URL= MNEMONIC= EVM_CHAIN_ID= ./sh/deployCustomConsistencyLevel.sh
#  tilt: ./sh/deployCustomConsistencyLevel.sh
#  anvil: EVM_CHAIN_ID=31337 MNEMONIC=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80 ./sh/deployCustomConsistencyLevel.sh

if [ "${RPC_URL}X" == "X" ]; then
  RPC_URL=http://localhost:8545
fi

if [ "${MNEMONIC}X" == "X" ]; then
  MNEMONIC=0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d
fi

if [ "${EVM_CHAIN_ID}X" == "X" ]; then
  EVM_CHAIN_ID=1337
fi

forge script ./forge-scripts/DeployCustomConsistencyLevel.s.sol:DeployCustomConsistencyLevel \
	--sig "run()" \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast ${FORGE_ARGS}

returnInfo=$(cat ./broadcast/DeployCustomConsistencyLevel.s.sol/$EVM_CHAIN_ID/run-latest.json)

DEPLOYED_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
echo "Deployed custom consistency level to address: $DEPLOYED_ADDRESS"
