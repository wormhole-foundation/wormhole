#!/bin/bash

# MNEMONIC=<redacted> WORMHOLE_ADDRESS=<from_the_previous_command> ./sh/deployNFTBridge.sh

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

forge script ./forge-scripts/DeployNFTBridge.s.sol:DeployNFTBridge \
	--sig "run(uint16,uint16,bytes32,uint8,uint256,address)" $INIT_CHAIN_ID $INIT_GOV_CHAIN_ID $INIT_GOV_CONTRACT $BRIDGE_INIT_FINALITY $INIT_EVM_CHAIN_ID $WORMHOLE_ADDRESS \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast

returnInfo=$(cat ./broadcast/DeployNFTBridge.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
NFT_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
NFT_IMPLEMENTATION_ADDRESS=$(jq -r '.returns.nftImplementationAddress.value' <<< "$returnInfo")
NFT_BRIDGE_SETUP_ADDRESS=$(jq -r '.returns.setupAddress.value' <<< "$returnInfo")
NFT_BRIDGE_IMPLEMENTATION_ADDRESS=$(jq -r '.returns.implementationAddress.value' <<< "$returnInfo")

echo "-- NFTBridge Addresses ----------------------------------------------------~~"
echo "| NFT Implementation address      | $NFT_IMPLEMENTATION_ADDRESS |"
echo "| NFTBridgeSetup address          | $NFT_BRIDGE_SETUP_ADDRESS |"
echo "| NFTBridgeImplementation address | $NFT_BRIDGE_IMPLEMENTATION_ADDRESS |"
echo "| NFTBridge address               | $NFT_BRIDGE_ADDRESS |"
echo "-----------------------------------------------------------------------------"