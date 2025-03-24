#!/bin/bash
# Fetch latest block using curl
ETHEREUM_RPC_URL="https://ethereum.publicnode.com"
echo "Fetching latest block number from $ETHEREUM_RPC_URL"
LATEST_BLOCK=$(curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' $ETHEREUM_RPC_URL | grep -o '"result":"0x[^"]*' | cut -d'"' -f4)

if [ -z "$LATEST_BLOCK" ]; then
  echo "Failed to fetch latest block number."
  exit 1
fi

echo "Latest block: $LATEST_BLOCK"

# Run ccqlistener with the manually fetched block number
/ccqlistener --env mainnet --configDir /app/cfg
