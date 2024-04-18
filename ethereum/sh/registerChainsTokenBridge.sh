#!/bin/bash

# MNEMONIC=<redacted> ./sh/registerChainsTokenBridge.sh

. .env

[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }
[[ -z $RPC_URL ]] && { echo "Missing RPC_URL"; exit 1; }
[[ -z $TOKEN_BRIDGE_ADDRESS ]] && { echo "Missing TOKEN_BRIDGE_ADDRESS"; exit 1; }
[[ -z $TOKEN_BRIDGE_REGISTRATION_VAAS ]] && { echo "Missing TOKEN_BRIDGE_REGISTRATION_VAAS"; exit 1; }

forge script ./forge-scripts/RegisterChainsTokenBridge.s.sol:RegisterChainsTokenBridge \
	--sig "run(address,bytes[])" $TOKEN_BRIDGE_ADDRESS $TOKEN_BRIDGE_REGISTRATION_VAAS \
	--rpc-url $RPC_URL \
	--private-key $MNEMONIC \
	--broadcast
