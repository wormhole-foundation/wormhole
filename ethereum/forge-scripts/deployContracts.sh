#!/bin/bash

# Check if the three arguments are provided
if [ $# -ne 3 ]; then
  echo "Usage: $0 RPC_URL NETWORK CHAIN_ID"
  exit 1
fi

# Extract the arguments into separate variables
RPC_URL="$1"
NETWORK="$2"
CHAIN_ID="$3"

echo 'Here is the env file you are working with!'
cat .env

# Step 0: Deploy 4 dummy contracts to match devnet addresses in Anvil to what they originally were in Ganache 
# (the addresses depend on the number of contracts that have been previously deployed, and the wallet address, I believe!)
forge script ./forge-scripts/DeployDummyContract.s.sol:DeployDummyContract --sig "run(uint256)" 2 --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast

# Step 1: Run 'forge script DeployCore', get the JSON output from the specified file
returnInfo=$(forge script ./forge-scripts/DeployCore.s.sol:DeployCore --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast --json | tail -n 1)
echo 'this is the return info from the core bridge deployment'
echo $returnInfo
# Extract the 'WORMHOLE_ADDRESS' value from 'returnInfo'
WORMHOLE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

echo "Wormhole address that will be used to initialize token bridge and nft bridge: $WORMHOLE_ADDRESS"

# Step 2: Replace 'WORMHOLE_ADDRESS' in the .env file with the extracted value
sed -i "s/^WORMHOLE_ADDRESS=.*$/WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS/" .env || echo "WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS" >> .env

# Step 3: Run 'forge script DeployTokenBridge'
returnInfo=$(forge script ./forge-scripts/DeployTokenBridge.s.sol:DeployTokenBridge --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast --json | tail -n 1)
echo 'this is the return info from the token bridge deployment'
echo $returnInfo
TOKEN_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

# Step 4: Run 'forge script DeployNFTBridge'
returnInfo=$(forge script ./forge-scripts/DeployNFTBridge.s.sol:DeployNFTBridge --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast --json | tail -n 1)
echo 'this is the return info from the ngt bridge deployment'
echo $returnInfo
NFT_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

echo "Deployment of Core Bridge, Token Bridge, and NFT Bridge completed successfully."

# TODO: Add other important addresses (setup, implementation, etc)
json="{\"WORMHOLE_ADDRESS\":\"$WORMHOLE_ADDRESS\",\"TOKEN_BRIDGE_ADDRESS\":\"$TOKEN_BRIDGE_ADDRESS\",\"NFT_BRIDGE_ADDRESS\":\"$NFT_BRIDGE_ADDRESS\"}"

# Get the current time as a Unix timestamp
current_time=$(date +%s)

mkdir -p "./deployment-addresses/$NETWORK/$CHAIN_ID/"

echo "$json" > "./deployment-addresses/$NETWORK/$CHAIN_ID/latest.json"
echo "$json" > "./deployment-addresses/$NETWORK/$CHAIN_ID/$current_time.json"