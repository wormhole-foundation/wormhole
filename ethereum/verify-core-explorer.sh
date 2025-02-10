#!/usr/bin/env bash

set -u

# Usage: ./verify-core-explorer.sh <path to ethereum deployment directory> <env file>
# You also need to export an env var `SCAN_API_TOKENS` with the path to a json file that contains a list of named tuples (etherscan API token, chain id).

cd "$1"
source "$2"

export CHAIN="$INIT_EVM_CHAIN_ID"
export ETHERSCAN_API_KEY=$(jq --raw-output ".[] | select(.chainId == $INIT_CHAIN_ID) | .etherscan" "$SCAN_API_TOKENS")
if [ -z $ETHERSCAN_API_KEY ]; then
  echo "No Etherscan API key found for chain $INIT_CHAIN_ID in $SCAN_API_TOKENS"
  exit 1
fi

# TODO: allow other json files to be read too
returnInfo=$(cat "./broadcast/DeployCore.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json")
# Extract the address values from 'returnInfo'
setup_address=$(jq -r '.returns.setupAddress.value' <<< "$returnInfo")
implementation_address=$(jq -r '.returns.implAddress.value' <<< "$returnInfo")
wormhole_address=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

# verification relying on foundry mapping the explorer url to the evm chain id.
# btw, you may need to add special cases below.

if [ $INIT_CHAIN_ID -eq 45 ]; then   # Worldscan
  worldscan_explorer_url="https://api.worldscan.org/api/?apikey=$ETHERSCAN_API_KEY"
  forge verify-contract --verifier blockscout --verifier-url "$worldscan_explorer_url" --watch \
    "$setup_address" contracts/Setup.sol:Setup 
  forge verify-contract --verifier blockscout --verifier-url "$worldscan_explorer_url" --watch \
    "$implementation_address" contracts/Implementation.sol:Implementation
  forge verify-contract --verifier blockscout --verifier-url "$worldscan_explorer_url" --watch \
     "$wormhole_address" contracts/Wormhole.sol:Wormhole \
     --constructor-args $(cast abi-encode "constructor(address,bytes)" "$setup_address" \
      $(cast calldata "setup(address,address[],uint16,uint16,bytes32,uint256)" "$implementation_address" "$INIT_SIGNERS" "$INIT_CHAIN_ID" "$INIT_GOV_CHAIN_ID" "$INIT_GOV_CONTRACT" "$INIT_EVM_CHAIN_ID"))

  exit 0
fi

if [ $INIT_CHAIN_ID -eq 39 ]; then   # Berachain
  berascan_explorer_url="https://api.berascan.com/api?apikey=$ETHERSCAN_API_KEY"
  forge verify-contract --verifier blockscout --verifier-url "$berascan_explorer_url" --watch \
    "$setup_address" contracts/Setup.sol:Setup 
  forge verify-contract --verifier blockscout --verifier-url "$berascan_explorer_url" --watch \
    "$implementation_address" contracts/Implementation.sol:Implementation
  forge verify-contract --verifier blockscout --verifier-url "$berascan_explorer_url" --watch \
     "$wormhole_address" contracts/Wormhole.sol:Wormhole \
     --constructor-args $(cast abi-encode "constructor(address,bytes)" "$setup_address" \
      $(cast calldata "setup(address,address[],uint16,uint16,bytes32,uint256)" "$implementation_address" "$INIT_SIGNERS" "$INIT_CHAIN_ID" "$INIT_GOV_CHAIN_ID" "$INIT_GOV_CONTRACT" "$INIT_EVM_CHAIN_ID"))

  exit 0
fi

# Default handling for other chains

forge verify-contract --watch \
  "$setup_address" contracts/Setup.sol:Setup
forge verify-contract --watch \
  "$implementation_address" contracts/Implementation.sol:Implementation
forge verify-contract --watch \
  "$wormhole_address" contracts/Wormhole.sol:Wormhole \
  --constructor-args $(cast abi-encode "constructor(address,bytes)" "$setup_address" \
    $(cast calldata "setup(address,address[],uint16,uint16,bytes32,uint256)" "$implementation_address" "$INIT_SIGNERS" "$INIT_CHAIN_ID" "$INIT_GOV_CHAIN_ID" "$INIT_GOV_CONTRACT" "$INIT_EVM_CHAIN_ID"))

# verification on oklink for X Layer requires specifying the URL
# forge verify-contract --verifier-url https://www.oklink.com/api/v5/explorer/contract/verify-source-code-plugin/XLAYER "$setup_address" contracts/Setup.sol:Setup --watch
# forge verify-contract --verifier-url https://www.oklink.com/api/v5/explorer/contract/verify-source-code-plugin/XLAYER "$implementation_address" contracts/Implementation.sol:Implementation --watch
# forge verify-contract --verifier-url https://www.oklink.com/api/v5/explorer/contract/verify-source-code-plugin/XLAYER "$wormhole_address" contracts/Wormhole.sol:Wormhole --watch \
#   --constructor-args $(cast abi-encode "constructor(address,bytes)" "$setup_address" \
#     $(cast calldata "setup(address,address[],uint16,uint16,bytes32,uint256)" "$implementation_address" "$INIT_SIGNERS" "$INIT_CHAIN_ID" "$INIT_GOV_CHAIN_ID" "$INIT_GOV_CONTRACT" "$INIT_EVM_CHAIN_ID"))

# verification on mantle
# mantle_explorer_url="https://explorer.mantle.xyz/api?module=contract&action=verify"
# forge verify-contract --verifier blockscout --verifier-url "$mantle_explorer_url" --watch \
#   "$setup_address" contracts/Setup.sol:Setup
# forge verify-contract --verifier blockscout --verifier-url "$mantle_explorer_url" --watch \
#   "$implementation_address" contracts/Implementation.sol:Implementation
# forge verify-contract --verifier blockscout --verifier-url "$mantle_explorer_url" --watch \
#   "$wormhole_address" contracts/Wormhole.sol:Wormhole \
#   --constructor-args $(cast abi-encode "constructor(address,bytes)" "$setup_address" \
#     $(cast calldata "setup(address,address[],uint16,uint16,bytes32,uint256)" "$implementation_address" "$INIT_SIGNERS" "$INIT_CHAIN_ID" "$INIT_GOV_CHAIN_ID" "$INIT_GOV_CONTRACT" "$INIT_EVM_CHAIN_ID"))
