#!/bin/bash

# Check if the three arguments are provided
if [ $# -ne 3 ]; then
  echo "Usage: $0 RPC_URL NETWORK PRIVATE_KEY"
  exit 1
fi


echo 'Here is the env file you are working with!'
cat .env

# Load the environment variables from .env
if [ -f .env ]; then
  source .env
else
  echo "The .env file does not exist."
  exit 1
fi

# Extract the arguments into separate variables
RPC_URL="$1"
NETWORK="$2"
PRIVATE_KEY="$3"


# Step 0: Deploy 2 dummy contracts to match devnet addresses in Anvil to what they originally were in Ganache 
# (the addresses depend on the number of contracts that have been previously deployed, and the wallet address, I believe!)
forge script ./forge-scripts/DeployDummyContract.s.sol:DeployDummyContract --sig "run(uint256)" 2 --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast

# Step 1: Run 'forge script DeployCore', get the JSON output from the specified file
forge script ./forge-scripts/DeployCore.s.sol:DeployCore --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast
returnInfo=$(cat ./broadcast/DeployCore.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the 'WORMHOLE_ADDRESS' value from 'returnInfo'
WORMHOLE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

echo "Wormhole address that will be used to initialize token bridge and nft bridge: $WORMHOLE_ADDRESS"

# Step 2: Replace 'WORMHOLE_ADDRESS' in the .env file with the extracted value
if grep -q "^WORMHOLE_ADDRESS=" .env; then 
  if [[ "$OSTYPE" == "darwin"* ]]; then
    # on macOS's sed, the -i flag needs the '' argument to not create
    # backup files
    sed -i '' "s/^WORMHOLE_ADDRESS=.*$/WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS/" .env
  else
    sed -i "s/^WORMHOLE_ADDRESS=.*$/WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS/" .env || echo "WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS" >> .env
  fi
  echo "Replaced Wormhole address in .env to be $WORMHOLE_ADDRESS"
else 
  echo -e "\nWORMHOLE_ADDRESS=$WORMHOLE_ADDRESS" >> .env
  echo "Added WORMHOLE_ADDRESS=$WORMHOLE_ADDRESS to .env"
fi

# Step 0: Deploy 1 dummy contracts to match devnet addresses in Anvil to what they originally were in Ganache 
# (the addresses depend on the number of contracts that have been previously deployed, and the wallet address, I believe!)
forge script ./forge-scripts/DeployDummyContract.s.sol:DeployDummyContract --sig "run(uint256)" 1 --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast


# Step 3: Run 'forge script DeployTokenBridge'
forge script ./forge-scripts/DeployTokenBridge.s.sol:DeployTokenBridge --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast 
returnInfo=$(cat ./broadcast/DeployTokenBridge.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
TOKEN_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

# Step 0: Deploy 1 dummy contracts to match devnet addresses in Anvil to what they originally were in Ganache 
# (the addresses depend on the number of contracts that have been previously deployed, and the wallet address, I believe!)
forge script ./forge-scripts/DeployDummyContract.s.sol:DeployDummyContract --sig "run(uint256)" 1 --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast


# Step 4: Run 'forge script DeployNFTBridge'
forge script ./forge-scripts/DeployNFTBridge.s.sol:DeployNFTBridge --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast
returnInfo=$(cat ./broadcast/DeployNFTBridge.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
NFT_BRIDGE_ADDRESS=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

echo "Deployment of Core Bridge, Token Bridge, and NFT Bridge completed successfully."

# TODO: Add other important addresses (setup, implementation, etc)
json="{\"WORMHOLE_ADDRESS\":\"$WORMHOLE_ADDRESS\",\"TOKEN_BRIDGE_ADDRESS\":\"$TOKEN_BRIDGE_ADDRESS\",\"NFT_BRIDGE_ADDRESS\":\"$NFT_BRIDGE_ADDRESS\"}"

# Get the current time as a Unix timestamp
current_time=$(date +%s)

mkdir -p "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/"

echo "$json" > "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/latest.json"
echo "$json" > "./deployment-addresses/$NETWORK/$INIT_CHAIN_ID/$current_time.json"