#!/bin/bash

# MNEMONIC=<redacted> ./sh/registerChainsNFTBridge.sh

. .env

[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }
[[ -z $RPC_URL ]] && { echo "Missing RPC_URL"; exit 1; }
[[ -z $NFT_BRIDGE_ADDRESS ]] && { echo "Missing NFT_BRIDGE_ADDRESS"; exit 1; }
[[ -z $NFT_BRIDGE_REGISTRATION_VAAS ]] && { echo "Missing NFT_BRIDGE_REGISTRATION_VAAS"; exit 1; }

forge script ./forge-scripts/RegisterChainsNFTBridge.s.sol:RegisterChainsNFTBridge \
	--sig "run(address,bytes[])" $NFT_BRIDGE_ADDRESS $NFT_BRIDGE_REGISTRATION_VAAS \
	--rpc-url $RPC_URL \
	--private-key $MNEMONIC \
	--broadcast
