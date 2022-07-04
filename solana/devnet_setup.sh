#!/usr/bin/env bash
# This script configures the devnet for test transfers with hardcoded addresses.
set -xu

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

# Static key for the NFT mint so it always has the same address
cat <<EOF > nft.json
[155,117,110,235,96,214,56,128,109,79,49,209,212,13,134,5,43,123,213,68,21,156,128,100,95,8,43,51,188,230,21,197,156,0,108,72,200,203,243,56,73,203,7,163,249,54,21,156,197,35,249,89,28,177,153,154,189,69,137,14,197,254,233,183]
EOF

# Static key for the 2nd NFT mint so it always has the same address
cat <<EOF > nft2.json
[40,74,92,250,81,56,202,67,129,124,193,219,24,161,198,98,191,214,136,7,112,26,72,17,33,249,24,225,183,237,27,216,11,179,26,170,82,220,3,253,152,185,151,186,12,21,138,161,175,46,180,3,167,165,70,51,128,45,237,143,146,49,34,180]
EOF

# Constants
cli_address=6sbzC1eH4FTujJXWj51eQe25cYvr4xfXbJ1vAj7j2k5J
bridge_address=Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o
nft_bridge_address=NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA
token_bridge_address=B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE
recipient_address=90F8bf6A479f320ead074411a4B0e7944Ea8c9C1
chain_id_ethereum=2

# load the .env file with the devent init data
source .env

retry () {
  while ! $@; do
    sleep 1
  done
}

# Fund our account (as defined in solana/keys/solana-devnet.json).
retry solana airdrop 1000

# Create a new SPL token
token=$(spl-token create-token -- token.json | grep 'Creating token' | awk '{ print $3 }')
echo "Created token $token"

# Create token account
account=$(spl-token create-account "$token" | grep 'Creating account' | awk '{ print $3 }')
echo "Created token account $account"

# Mint new tokens owned by our CLI account
spl-token mint "$token" 10000000000 "$account"

# Create meta for token
token-bridge-client create-meta "$token" "Solana Test Token" "SOLT" ""

# Create a new SPL NFT
nft=$(spl-token create-token --decimals 0 -- nft.json | grep 'Creating token' | awk '{ print $3 }')
echo "Created NFT $nft"

# Create NFT account
nft_account=$(spl-token create-account "$nft" | grep 'Creating account' | awk '{ print $3 }')
echo "Created NFT account $nft_account"

# Mint new NFT owned by our CLI account
spl-token mint "$nft" 1 "$nft_account"

# Create meta for token
token-bridge-client create-meta "$nft" "Not a PUNK🎸" "PUNK🎸" "https://wrappedpunks.com:3000/api/punks/metadata/39"

nft=$(spl-token create-token --decimals 0 -- nft2.json | grep 'Creating token' | awk '{ print $3 }')
echo "Created NFT $nft"

nft_account=$(spl-token create-account "$nft" | grep 'Creating account' | awk '{ print $3 }')
echo "Created NFT account $nft_account"

spl-token mint "$nft" 1 "$nft_account"

token-bridge-client create-meta "$nft" "Not a PUNK 2🎸" "PUNK2🎸" "https://wrappedpunks.com:3000/api/punks/metadata/51"

# Create the bridge contract at a known address
# OK to fail on subsequent attempts (already created).
retry client create-bridge "$bridge_address" "$INIT_SIGNERS_CSV" 86400 100

# Initialize the token bridge
retry token-bridge-client create-bridge "$token_bridge_address" "$bridge_address"
# Initialize the NFT bridge
retry token-bridge-client create-bridge "$nft_bridge_address" "$bridge_address"

# pass the chain registration VAAs sourced from .env to the client's execute-governance command:
pushd /usr/src/clients/js
make build
# Register the Token Bridge Endpoint on ETH
node build/main.js submit -c solana -n devnet "$REGISTER_ETH_TOKEN_BRIDGE_VAA"
node build/main.js submit -c solana -n devnet "$REGISTER_TERRA_TOKEN_BRIDGE_VAA"
node build/main.js submit -c solana -n devnet "$REGISTER_BSC_TOKEN_BRIDGE_VAA"
node build/main.js submit -c solana -n devnet "$REGISTER_ALGO_TOKEN_BRIDGE_VAA"
node build/main.js submit -c solana -n devnet "$REGISTER_TERRA2_TOKEN_BRIDGE_VAA"
# Register the NFT Bridge Endpoint on ETH
node build/main.js submit -c solana -n devnet "$REGISTER_ETH_NFT_BRIDGE_VAA"
node build/main.js submit -c solana -n devnet "$REGISTER_TERRA_NFT_BRIDGE_VAA"
popd

# Let k8s startup probe succeed
nc -k -l -p 2000
