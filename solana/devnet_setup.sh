#!/usr/bin/env bash
# This script configures the devnet for test transfers with hardcoded addresses.
set -x

# Configure CLI (works the same as upstream Solana CLI)
mkdir -p ~/.config/solana/cli
cat <<EOF > ~/.config/solana/cli/config.yml
json_rpc_url: "http://127.0.0.1:8899"
websocket_url: ""
keypair_path: /usr/src/solana/keys/solana-devnet.json
EOF

# Static key for the mint so it always has the same address
cat <<EOF > token.json
[179,228,102,38,68,102,75,133,127,56,63,167,143,42,59,29,220,215,100,149,220,241,176,204,154,241,168,147,195,139,55,100,22,88,9,115,146,64,160,172,3,185,132,64,254,137,133,84,142,58,166,131,205,13,77,157,245,181,101,150,105,250,163,1]
EOF

# Constants
cli_address=6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J
bridge_address=Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o
token_bridge_address=B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE
initial_guardian=befa429d57cd18b7f8a4d91a2da9ab4af05d0fbe
recipient_address=90F8bf6A479f320ead074411a4B0e7944Ea8c9C1
chain_id_ethereum=2

retry () {
  while ! $@; do
    sleep 1
  done
}

# Fund our account (as seen in solana-devnet.json).
retry solana airdrop 1000 --faucet-port 9900 --faucet-host 127.0.0.1

# Create a new SPL token
token=$(spl-token create-token -- token.json | grep 'Creating token' | awk '{ print $3 }')
echo "Created token $token"

# Create token account
account=$(spl-token create-account "$token" | grep 'Creating account' | awk '{ print $3 }')
echo "Created token account $account"

# Mint new tokens owned by our CLI account
spl-token mint "$token" 10000000000 "$account"

# Create the bridge contract at a known address
# OK to fail on subsequent attempts (already created).
retry client create-bridge "$bridge_address" "$initial_guardian" 86400 100

# Initialize the token bridge
retry token-bridge-client create-bridge "$token_bridge_address" "$bridge_address"

# Register the Solana Endpoint on ETH
node /usr/src/clients/token_bridge/main.js solana execute_governance_vaa 01000000000100a1f5ebf9d460dbbbaf0b72df36b37139baac61bb3128eec2d77f2e53a57c15592f4c97521f8eb3a7279299219be0342877960a8bfa7da9782313bb0ff5855aba01000000010000000100010000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000546f6b656e4272696467650100000002000000000000000000000000e982e462b094850f12af94d21d470e21be9d0e9c

# Let k8s startup probe succeed
nc -k -l -p 2000
