#!/bin/bash

if [ "$DEV" != "True" ]; then
  if [ $CHAIN_ID -eq 4 ]; then
    RPC_URL='http://localhost:8545'
  else 
    RPC_URL='http://localhost:8545'
  fi
else 
  if [ $CHAIN_ID -eq 4 ]; then
    RPC_URL='http://localhost:8545'
  else 
    RPC_URL='http://localhost:8545'
  fi
fi

# Load the environment variables from .env
if [ -f .env ]; then
  source .env
else
  echo "The .env file does not exist."
  exit 1
fi

MNEMONIC=0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d
NETWORK='devnet'

npm run build:forge

NUM_RUNS=2 source ./sh/deployDummyContract.sh

echo "Deploying WORMHOLE_CORE"
source ./sh/deployCoreBridge.sh
returnInfo=$(cat ./broadcast/DeployCore.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
WORMHOLE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
echo "WORMHOLE_ADDRESS: ${WORMHOLE_ADDRESS}"

NUM_RUNS=1 source ./sh/deployDummyContract.sh

echo "Deploying TOKEN_BRIDGE"
source ./sh/deployTokenBridge.sh
returnInfo=$(cat ./broadcast/DeployTokenBridge.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
TOKEN_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
echo "TOKEN_BRIDGE_ADDRESS: $TOKEN_BRIDGE_ADDRESS"

NUM_RUNS=1 source ./sh/deployDummyContract.sh

echo "Deploying NFT_BRIDGE"
source ./sh/deployNFTBridge.sh
returnInfo=$(cat ./broadcast/DeployNFTBridge.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
NFT_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")
echo "NFT_BRIDGE_ADDRESS: $NFT_BRIDGE_ADDRESS"

NUM_RUNS=17 source ./sh/deployDummyContract.sh

echo "Deploying test tokens"
forge script ./forge-scripts/DeployTestToken.s.sol:DeployTestToken --rpc-url $RPC_URL --private-key $MNEMONIC --broadcast
echo "Done deploying test tokens"

# Registration of chains
token_bridge_registration_vaas_arr=()
nft_bridge_registration_vaas_arr=()
while IFS= read -r line; do
  # Use a regular expression to match the desired pattern
  if [[ "$line" =~ ^REGISTER_.*_TOKEN_BRIDGE_VAA=([^[:space:]]+) ]]; then
    token_bridge_registration_vaas_arr+=("0x${BASH_REMATCH[1]}")
  fi
  if [[ "$line" =~ ^REGISTER_.*_NFT_BRIDGE_VAA=([^[:space:]]+) ]]; then
    nft_bridge_registration_vaas_arr+=("0x${BASH_REMATCH[1]}")
  fi
done < ".env"

TOKEN_BRIDGE_REGISTRATION_VAAS="[$(IFS=','; echo "${token_bridge_registration_vaas_arr[*]}")]";
NFT_BRIDGE_REGISTRATION_VAAS="[$(IFS=','; echo "${nft_bridge_registration_vaas_arr[*]}")]";

echo "Registering chains on token bridge"
echo "TOKEN_BRIDGE_REGISTRATION_VAAS: $TOKEN_BRIDGE_REGISTRATION_VAAS"
source "./sh/registerChainsTokenBridge.sh"
echo "Done registering chains on token bridge"

echo "Registering chains on nft bridge"
echo "NFT_BRIDGE_REGISTRATION_VAAS: $NFT_BRIDGE_REGISTRATION_VAAS"
source "./sh/registerChainsNFTBridge.sh"
echo "Done registering chains on nft bridge"
