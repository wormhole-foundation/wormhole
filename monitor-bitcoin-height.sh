#!/bin/bash

# Monitor Bitcoin block height every second
while true; do
  HEIGHT=$(kubectl exec bitcoin-hacknet-0 -c bitcoin-node -n wormhole -- bitcoin-cli -rpcconnect=127.0.0.1 getblockcount 2>/dev/null || echo "ERROR")
  TIMESTAMP=$(date '+%H:%M:%S')
  echo "[$TIMESTAMP] Block height: $HEIGHT"
  sleep 1
done