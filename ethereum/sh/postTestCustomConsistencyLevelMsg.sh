#!/bin/bash

#
# This script posts an observation via the TestCustomConsistencyLevel contract.
# Usage: RPC_URL= MNEMONIC= EVM_CHAIN_ID= ADDR= PAYLOAD= ./sh/postTestCustomConsistencyLevelMsg.sh
#  tilt: ./sh/postTestCustomConsistencyLevelMsg.sh

if [ "${RPC_URL}X" == "X" ]; then
  RPC_URL=http://localhost:8545
fi

if [ "${MNEMONIC}X" == "X" ]; then
  MNEMONIC=0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d
fi

if [ "${EVM_CHAIN_ID}X" == "X" ]; then
  EVM_CHAIN_ID=1337
fi

if [ "${ADDR}X" == "X" ]; then
  ADDR=0x40d5B8a71a5e8202a26a53406eFAb755C0db6e93 
fi

if [ "${PAYLOAD}X" == "X" ]; then
  PAYLOAD="Hello, World!"
fi

forge script ./forge-scripts/PostTestCustomConsistencyLevelMsg.s.sol:PostTestCustomConsistencyLevelMsg \
	--sig "run(address,string calldata)" $ADDR "$PAYLOAD" \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast ${FORGE_ARGS}

DEPLOYED_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
echo "Posted a message to address: $ADDR: $returnInfo"
