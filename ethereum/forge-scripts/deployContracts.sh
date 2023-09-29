#!/bin/bash

# Check if the RPC_URL argument is provided
if [ $# -ne 1 ]; then
  echo "Usage: $0 RPC_URL"
  exit 1
fi

RPC_URL="$1"

echo 'Here is the env file you are working with!'
env

# Step 0: Deploy 4 dummy contracts to match devnet addresses in Anvil to what they originally were in Ganache 
# (the addresses depend on the number of contracts that have been previously deployed, and the wallet address, I believe!)
forge script ./forge-scripts/DeployDummyContract.s.sol:DeployDummyContract --sig "run(uint256)" 4 --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast

# Step 1: Run 'forge script DeployCore' 
forge script ./forge-scripts/DeployCore.s.sol:DeployCore --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast
# Get the JSON output from the specified file
returnInfo=$(cat ./broadcast/DeployCore.s.sol/1/run-latest.json)

# Extract the 'WORMHOLE_ADDRESS' value from 'returnInfo'
WORMHOLE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

echo "Wormhole address that will be used to initialize token bridge and nft bridge: $WORMHOLE_ADDRESS"

# Step 2: Replace 'WORMHOLE_ADDRESS' in the .env file with the extracted value
sed -i "s/^WORMHOLE_ADDRESS=.*$/WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS/" .env || echo "WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS" >> .env

# Step 3: Run 'forge script DeployTokenBridge'
forge script ./forge-scripts/DeployTokenBridge.s.sol:DeployTokenBridge --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast

# Step 4: Run 'forge script DeployNFTBridge'
forge script ./forge-scripts/DeployNFTBridge.s.sol:DeployNFTBridge --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast

echo "Deployment of Core Bridge, Token Bridge, and NFT Bridge completed successfully."