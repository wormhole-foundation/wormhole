#!/bin/bash

# Check if the RPC_URL argument is provided
if [ $# -ne 1 ]; then
  echo "Usage: $0 RPC_URL"
  exit 1
fi

RPC_URL="$1"

# Step 1: Run 'forge script DeployCore' and store the JSON output in 'returnInfo'
returnInfo=$(forge script DeployCore --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY")

# Check if 'returnInfo' contains a valid JSON
if ! jq -e . >/dev/null 2>&1 <<< "$returnInfo"; then
  echo "Error: 'forge script DeployCore' did not return valid JSON."
  exit 1
fi

# Extract the 'WORMHOLE_ADDRESS' value from 'returnInfo'
WORMHOLE_ADDRESS=$(jq -r '.results[0].value' <<< "$returnInfo")

# Step 2: Replace 'WORMHOLE_ADDRESS' in the .env file with the extracted value
sed -i "s/^WORMHOLE_ADDRESS=.*$/WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS/" .env

# Step 3: Run 'forge script DeployTokenBridge'
forge script DeployTokenBridge --rpc-url "$RPC_URL"

# Step 4: Run 'forge script DeployNFTBridge'
forge script DeployNFTBridge --rpc-url "$RPC_URL"

echo "Deployment of Core Bridge, Token Bridge, and NFT Bridge completed successfully."