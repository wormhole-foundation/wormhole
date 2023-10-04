#!/bin/bash

if [ $CI ]; then
  if [ $CHAIN_ID -eq 4 ]; then
    RPC_URL='http://eth-devnet2:8545'
  else 
    RPC_URL='http://eth-devnet:8545'
  fi
else 
  if [ $CHAIN_ID -eq 4 ]; then
    RPC_URL='http://localhost:8546'
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

PRIVATE_KEY=0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d
NETWORK='devnet'

npm run build:forge

NUM_RUNS=2 source ./forge-scripts/deployDummyContract.sh

source ./forge-scripts/deployCoreBridge.sh

returnInfo=$(cat ./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/CoreBridge/latest.json)
WORMHOLE_ADDRESS=$(jq -r '.WORMHOLE_ADDRESS' <<< "$returnInfo")

NUM_RUNS=1 source ./forge-scripts/deployDummyContract.sh

source ./forge-scripts/deployTokenBridge.sh

NUM_RUNS=1 source ./forge-scripts/deployDummyContract.sh

source ./forge-scripts/deployNFTBridge.sh

NUM_RUNS=17 source ./forge-scripts/deployDummyContract.sh

forge script ./forge-scripts/DeployTestToken.s.sol:DeployTestToken --rpc-url $RPC_URL --private-key $PRIVATE_KEY --broadcast

# Get Token Bridge and NFT Bridge addresses

returnInfo=$(cat ./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/TokenBridge/latest.json)
TOKEN_BRIDGE_ADDRESS=$(jq -r '.TOKEN_BRIDGE_ADDRESS' <<< "$returnInfo")
echo "Token Bridge address: $TOKEN_BRIDGE_ADDRESS"

returnInfo=$(cat ./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/NFTBridge/latest.json)
NFT_BRIDGE_ADDRESS=$(jq -r '.NFT_BRIDGE_ADDRESS' <<< "$returnInfo")
echo "NFT Bridge address: $NFT_BRIDGE_ADDRESS"

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

echo 'token bridge address';
echo $TOKEN_BRIDGE_ADDRESS;
echo 'token bridge registrations';
echo $TOKEN_BRIDGE_REGISTRATION_VAAS;

source "./forge-scripts/registerChainsTokenBridge.sh"
echo 'Registration of token bridges done';
source "./forge-scripts/registerChainsNFTBridge.sh"
echo 'Registration of NFT bridges done';

if [ $CHAIN_ID -eq 4 ]; then
  npm run deploy-relayers-evm2
else 
  npm run deploy-relayers-evm1
fi

nc -lkn 2000