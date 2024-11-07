#!/usr/bin/env bash

set -u

# Usage: ./verify-tb-explorer.sh <path to ethereum deployment directory> <env file>
# You also need to export an env var `SCAN_API_TOKENS` with the path to a json file that contains a list of named tuples (etherscan API token, chain id).

cd "$1"
source "$2"

export CHAIN="$INIT_EVM_CHAIN_ID"

export ETHERSCAN_API_KEY=$(jq --raw-output ".[] | select(.chainId == $INIT_CHAIN_ID) | .etherscan" "$SCAN_API_TOKENS")

# TODO: allow other json files to be read too
wormhole_address=$(jq -r '.returns.deployedAddress.value' < ./broadcast/DeployCore.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
returnInfo=$(cat ./broadcast/DeployTokenBridge.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
token_bridge_address=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
token_implementation_address=$(jq -r '.returns.tokenImplementationAddress.value' <<< "$returnInfo")
token_bridge_setup_address=$(jq -r '.returns.bridgeSetupAddress.value' <<< "$returnInfo")
token_bridge_implementation_address=$(jq -r '.returns.bridgeImplementationAddress.value' <<< "$returnInfo")

# verification relying on foundry mapping the explorer url to the evm chain id
forge verify-contract --watch \
  "$token_bridge_setup_address" contracts/bridge/BridgeSetup.sol:BridgeSetup
forge verify-contract --watch \
  "$token_implementation_address" contracts/bridge/token/TokenImplementation.sol:TokenImplementation
forge verify-contract --watch \
  "$token_bridge_implementation_address" contracts/bridge/BridgeImplementation.sol:BridgeImplementation
forge verify-contract --watch \
  "$token_bridge_address" contracts/bridge/TokenBridge.sol:TokenBridge \
  --constructor-args $(cast abi-encode "constructor(address,bytes)" "$token_bridge_setup_address" \
    $(cast calldata "setup(address,uint16,address,uint16,bytes32,address,address,uint8,uint256)" \
      "$token_bridge_implementation_address" "$BRIDGE_INIT_CHAIN_ID" "$wormhole_address" "$BRIDGE_INIT_GOV_CHAIN_ID" "$BRIDGE_INIT_GOV_CONTRACT" \
      "$token_implementation_address" "$BRIDGE_INIT_WETH" "$BRIDGE_INIT_FINALITY" "$INIT_EVM_CHAIN_ID"))

# verification on oklink requires specifying the URL
# forge verify-contract --verifier-url https://www.oklink.com/api/v5/explorer/contract/verify-source-code-plugin/XLAYER "$token_bridge_setup_address" contracts/bridge/BridgeSetup.sol:BridgeSetup --watch
# forge verify-contract --verifier-url https://www.oklink.com/api/v5/explorer/contract/verify-source-code-plugin/XLAYER "$token_implementation_address" contracts/bridge/token/TokenImplementation.sol:TokenImplementation --watch
# forge verify-contract --verifier-url https://www.oklink.com/api/v5/explorer/contract/verify-source-code-plugin/XLAYER "$token_bridge_implementation_address" contracts/bridge/BridgeImplementation.sol:BridgeImplementation --watch
# forge verify-contract --verifier-url https://www.oklink.com/api/v5/explorer/contract/verify-source-code-plugin/XLAYER "$token_bridge_address" contracts/bridge/TokenBridge.sol:TokenBridge --watch \
#   --constructor-args $(cast abi-encode "constructor(address,bytes)" "$token_bridge_setup_address" \
#     $(cast calldata "setup(address,uint16,address,uint16,bytes32,address,address,uint8,uint256)" \
#       "$token_bridge_implementation_address" "$BRIDGE_INIT_CHAIN_ID" "$wormhole_address" "$BRIDGE_INIT_GOV_CHAIN_ID" "$BRIDGE_INIT_GOV_CONTRACT" \
#       "$token_implementation_address" "$BRIDGE_INIT_WETH" "$BRIDGE_INIT_FINALITY" "$INIT_EVM_CHAIN_ID"))


# verification on mantle
# mantle_explorer_url="https://explorer.mantle.xyz/api?module=contract&action=verify"
# forge verify-contract --verifier blockscout --verifier-url "$mantle_explorer_url" --watch \
#   "$token_bridge_setup_address" contracts/bridge/BridgeSetup.sol:BridgeSetup
# forge verify-contract --verifier blockscout --verifier-url "$mantle_explorer_url" --watch \
#   "$token_implementation_address" contracts/bridge/token/TokenImplementation.sol:TokenImplementation
# forge verify-contract --verifier blockscout --verifier-url "$mantle_explorer_url" --watch \
#   "$token_bridge_implementation_address" contracts/bridge/BridgeImplementation.sol:BridgeImplementation
# forge verify-contract --verifier blockscout --verifier-url "$mantle_explorer_url" --watch \
#   "$token_bridge_address" contracts/bridge/TokenBridge.sol:TokenBridge \
#   --constructor-args $(cast abi-encode "constructor(address,bytes)" "$token_bridge_setup_address" \
#     $(cast calldata "setup(address,uint16,address,uint16,bytes32,address,address,uint8,uint256)" \
#       "$token_bridge_implementation_address" "$BRIDGE_INIT_CHAIN_ID" "$wormhole_address" "$BRIDGE_INIT_GOV_CHAIN_ID" "$BRIDGE_INIT_GOV_CONTRACT" \
#       "$token_implementation_address" "$BRIDGE_INIT_WETH" "$BRIDGE_INIT_FINALITY" "$INIT_EVM_CHAIN_ID"))
