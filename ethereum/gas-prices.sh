#!/bin/bash

if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <config-dir>"
    echo "where <config-dir> is the name of the directory in ts-scripts/relayer/config"
    echo "                   e.g: mainnet, testnet or devnet"
    exit 1
fi

# Parse the JSON and loop through each entry to get `rpc` and `chainId`
jq -c '.chains[] | {chainId: .chainId, rpc: .rpc}' ts-scripts/relayer/config/$1/chains.json | while read -r entry; do
    chainId=$(echo "$entry" | jq -r '.chainId')
    rpc=$(echo "$entry" | jq -r '.rpc')

    echo "ChainId: $chainId"
    echo "   gas-price:$(cast gas-price -r "$rpc")"
    echo "   base fee: $(cast basefee -r "$rpc")"
done
