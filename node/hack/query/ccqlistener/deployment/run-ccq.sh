#!/bin/bash

# Simple helper script for running CCQ listener

mkdir -p config

case "$1" in
  "genkey")
    echo "Generating signing key..."
    docker-compose run --rm generate-key
    echo "Key generated - check logs for public key"
    echo "Remember to add peer ID and public key to guardian configuration"
    ;;
    
  "listen")
    echo "Starting CCQ listener in listen-only mode..."
    
    if [ -n "$2" ]; then
      echo "Targeting specific peer ID: $2"
      docker-compose run --rm ccqlistener-listen /ccqlistener --env ${3:-mainnet} --listenOnly --targetPeerId $2 --configDir /app/cfg
    else
      docker-compose up ccqlistener-listen
    fi
    ;;
    
  "query")
    echo "Checking for signing key..."
    if [ ! -f "./config/ccqlistener.signerKey" ]; then
      echo "Signing key not found - generating one first..."
      docker-compose run --rm generate-key
    fi
    
    echo "Starting CCQ listener in query mode..."
    
    # Create temporary script to fetch block number 
    if [ -n "$2" ]; then
      if [[ "$2" == http* ]]; then
        ETHEREUM_RPC_URL="$2"
      else
        ETHEREUM_RPC_URL="https://rpc.ankr.com/eth/$2"
      fi
      
      echo "Using custom RPC URL: $ETHEREUM_RPC_URL"
      
      # Create a temporary script to fetch the block number and then run ccqlistener
      cat > ./config/run_query.sh <<EOL
#!/bin/bash
# Fetch latest block using curl
echo "Fetching latest block number from $ETHEREUM_RPC_URL"
LATEST_BLOCK=\$(curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' $ETHEREUM_RPC_URL | grep -o '"result":"0x[^"]*' | cut -d'"' -f4)

if [ -z "\$LATEST_BLOCK" ]; then
  echo "Failed to fetch latest block number. Check your API key or URL."
  exit 1
fi

echo "Latest block: \$LATEST_BLOCK"

# Run ccqlistener with the manually fetched block number
/ccqlistener --env ${3:-mainnet} --configDir /app/cfg
EOL
      
      # Make the script executable
      chmod +x ./config/run_query.sh
      
      # Run the container with our custom script
      docker-compose run --rm ccqlistener-query /bin/bash /app/cfg/run_query.sh
    else
      echo "Warning: No API key or URL provided. Try using a public endpoint:"
      echo "./run-ccq.sh query https://ethereum.publicnode.com"
      
      # Create a script with public node
      cat > ./config/run_query.sh <<EOL
#!/bin/bash
# Fetch latest block using curl
ETHEREUM_RPC_URL="https://ethereum.publicnode.com"
echo "Fetching latest block number from \$ETHEREUM_RPC_URL"
LATEST_BLOCK=\$(curl -s -X POST -H "Content-Type: application/json" --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \$ETHEREUM_RPC_URL | grep -o '"result":"0x[^"]*' | cut -d'"' -f4)

if [ -z "\$LATEST_BLOCK" ]; then
  echo "Failed to fetch latest block number."
  exit 1
fi

echo "Latest block: \$LATEST_BLOCK"

# Run ccqlistener with the manually fetched block number
/ccqlistener --env ${3:-mainnet} --configDir /app/cfg
EOL
      
      # Make the script executable
      chmod +x ./config/run_query.sh
      
      # Run the container with our custom script
      docker-compose run --rm ccqlistener-query /bin/bash /app/cfg/run_query.sh
    fi
    ;;
    
  "build")
    echo "Building all services..."
    docker-compose build
    ;;
    
  *)
    echo "CCQ Listener - Docker Compose Helper"
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  genkey   - Generate signing key for query mode"
    echo "  listen   - Start in listen-only mode"
    echo "  listen <PEER_ID> [ENV] - Listen for specific peer ID (ENV=mainnet by default)"
    echo "  query    [API_KEY|URL] [ENV] - Start in query mode (generates key if needed)"
    echo "  build    - Build all Docker images"
    echo ""
    echo "Examples:"
    echo "  $0 genkey               - Generate a new signing key"
    echo "  $0 listen               - Start listener in monitoring mode"
    echo "  $0 listen PEER_ID       - Listen on mainnet for specific guardian"
    echo "  $0 listen PEER_ID testnet - Listen on testnet for specific guardian"
    echo "  $0 query                - Start listener in query mode"
    echo "  $0 query YOUR_API_KEY   - Start listener in query mode with API key"
    echo "  $0 query https://rpc.ankr.com/eth/YOUR_API_KEY - Use full RPC URL"
    ;;
esac 