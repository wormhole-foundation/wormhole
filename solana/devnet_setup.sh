#!/usr/bin/env bash
# This script configures the devnet for test transfers with hardcoded addresses.
set -x

# Configure CLI (works the same as upstream Solana CLI)
mkdir -p ~/.config/solana/cli
cat <<EOF > ~/.config/solana/cli/config.yml
json_rpc_url: "http://127.0.0.1:8899"
websocket_url: ""
keypair_path: /usr/src/solana/id.json
EOF

# Static key for the mint so it always has the same address
cat <<EOF > token.json
[179,228,102,38,68,102,75,133,127,56,63,167,143,42,59,29,220,215,100,149,220,241,176,204,154,241,168,147,195,139,55,100,22,88,9,115,146,64,160,172,3,185,132,64,254,137,133,84,142,58,166,131,205,13,77,157,245,181,101,150,105,250,163,1]
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
retry cli create-bridge "$bridge_address" "$initial_guardian"

# Create wrapped asset for the token we mint in send-lockups.js (2 = Ethereum, 9 decimals)
wrapped_token=$(cli create-wrapped "$bridge_address" 2 9 000000000000000000000000CfEB869F69431e42cdB54A4F4f105C19C080A601 | grep 'Wrapped Mint address' | awk '{ print $4 }')
echo "Created wrapped token $wrapped_token"

# Create token account to receive wrapped assets from send-lockups.js
wrapped_account=$(spl-token create-account "$wrapped_token" | grep 'Creating account' | awk '{ print $3 }')
echo "Created wrapped token account $wrapped_account"

# Create wrapped asset and token account for Terra tokens (3 for Terra, 8 for precision)
wrapped_terra_token=$(cli create-wrapped "$bridge_address" 3 8 0000000000000000000000003b1a7485c6162c5883ee45fb2d7477a87d8a4ce5 | grep 'Wrapped Mint address' | awk '{ print $4 }')
echo "Created wrapped token for Terra $wrapped_terra_token"
wrapped_terra_account=$(cli create-account "$wrapped_terra_token" | grep 'Creating account' | awk '{ print $3 }')
echo "Created wrapped token account for Terra $wrapped_terra_account"

# Let k8s startup probe succeed
nc -l -p 2000

# Keep the container alive for interactive CLI usage
sleep infinity
