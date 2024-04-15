#!/bin/bash

forge install foundry-rs/forge-std@v1.5.5 --no-git --no-commit
forge install openzeppelin/openzeppelin-contracts@0457042d93d9dfd760dbaa06a4d2f1216fdbe297 --no-git --no-commit
rm contracts/DoubleTransfer.sol

# MNEMONIC= RPC_URL= EVM_CHAIN_ID= WORMHOLE_ADDRESS= ./forge-scripts/deployPublishMsg.sh

[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }
[[ -z $RPC_URL ]] && { echo "Missing RPC_URL"; exit 1; }
[[ -z $EVM_CHAIN_ID ]] && { echo "Missing EVM_CHAIN_ID"; exit 1; }
[[ -z $WORMHOLE_ADDRESS ]] && { echo "Missing WORMHOLE_ADDRESS"; exit 1; }

forge script ./forge-scripts/DeployPublishMsg.s.sol:DeployPublishMsg \
	--sig "run(address)" $WORMHOLE_ADDRESS \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--verifier-url 'https://api.routescan.io/v2/network/testnet/evm/${EVM_CHAIN_ID}/etherscan' \
	--etherscan-api-key "verifyContract" \
	--broadcast

returnInfo=$(cat ./broadcast/DeployPublishMsg.s.sol/$EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
PUBLISH_MSG_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

echo "Deployed PublishMsg to ${PUBLISH_MSG_ADDRESS} using core at ${WORMHOLE_ADDRESS}."
