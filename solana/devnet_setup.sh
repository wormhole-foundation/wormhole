#!/bin/bash
# This script configures the devnet for test transfers with hardcoded addresses.
set -x

# Configure CLI (works the same as upstream Solana CLI)
mkdir -p ~/.config/solana/cli
cat <<EOF > ~/.config/solana/cli/config.yml
json_rpc_url: "http://solana-devnet:8899"
websocket_url: ""
keypair_path: /usr/src/solana/id.json
EOF

# Constants
cli_address=6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J
bridge_address=Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o
initial_guardian=befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe
recipient_address=90F8bf6A479f320ead074411a4B0e7944Ea8c9C1
chain_id_ethereum=2

retry () {
  while ! $@; do
    sleep 1
  done
}

# Fund our account (as seen in id.json).
retry cli airdrop solana-devnet:9900

# Create the bridge contract at a known address
# OK to fail on subsequent attempts (already created).
retry cli create-bridge "$bridge_address" "$initial_guardian"

# Create a new SPL token (at a random address)
token=$(cli create-token | grep 'Creating token' | awk '{ print $3 }')
echo "Created token $token"

# Create token account
account=$(cli create-account "$token" | grep 'Creating account' | awk '{ print $3 }')
echo "Created token account $account"

# Mint new tokens owned by our CLI account
cli mint "$token" 10000000000 "$account"

# Do lock transactions <3
#while : ; do
#  cli lock "$bridge_address" "$account" "$token" 10 "$chain_id_ethereum" "$RANDOM" "$recipient_address"
  sleep 5000
#done
