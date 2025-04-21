#!/bin/bash

# Deploy and Create Staking Pools Script
# This script deploys the QueryTypeStakerFactory and creates both EVM and Solana staking pools
#
# Pools created:
#   1. EVM Pool - eth_call, eth_call_by_timestamp, eth_call_with_finality (types 1, 2, 3)
#   2. Solana Pool - sol_account, sol_pda (types 4, 5)
#
# Uses devnet configuration from ../wormhole/scripts/devnet-consts.json

set -e  # Exit on error

# ============================================================================
# Configuration
# ============================================================================
RPC_URL="${ETH_RPC_URL:-http://localhost:8545}"
PRIVATE_KEY="0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d"
W_TOKEN_ADDRESS="0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A"
RATE_LIMITS_FILE="${RATE_LIMITS_FILE:-./ccq-rate-limits.json}"

# Event topic for CreateQueryTypeStakingPool(bytes32 indexed queryType, address indexed poolAddress)
CREATE_POOL_EVENT_TOPIC="0x34d4b91c04bf254b71c435c46e26f1c0b6ec05b426b3bbebb5a80d3e71c030db"

# Common pool parameters
# Note: Decay rate is now encoded in the lowest byte of queryType (bits 0-7)
# Query type flags are in bits 8+
INITIAL_ENTRY=${INITIAL_ENTRY:-"0x0000000000000000000000000000000000000000000000000000000000000001"}

export PRIVATE_KEY W_TOKEN_ADDRESS

# ============================================================================
# Helper Functions
# ============================================================================

# Extract pool address from the CreateQueryTypeStakingPool event in broadcast file
extract_pool_address() {
    local broadcast_file="$1"
    if [ -f "$broadcast_file" ]; then
        local result
        result=$(jq -r --arg topic "$CREATE_POOL_EVENT_TOPIC" \
            '.receipts[0].logs[] | select(.topics[0] == $topic) | .topics[2]' \
            "$broadcast_file" 2>/dev/null)
        if [ -n "$result" ] && [ "$result" != "null" ]; then
            echo "$result" | sed 's/0x000000000000000000000000/0x/'
        fi
    fi
}

# Create a staking pool with the given parameters
# Usage: create_pool "Pool Name" "query_type_hex" "min_stake_wei_or_empty" "description"
create_pool() {
    local name="$1"
    local query_type="$2"
    local min_stake="$3"
    local description="$4"

    echo ""
    echo "Creating $name staking pool..."

    export QUERY_TYPE="$query_type"
    export INITIAL_ENTRY

    if [ -n "$min_stake" ]; then
        export MINIMUM_STAKE="$min_stake"
    else
        unset MINIMUM_STAKE
    fi

    echo "$name Pool parameters:"
    echo "  QUERY_TYPE: $query_type ($description)"
    echo "  MIN_STAKE: ${min_stake:-default (100 tokens)}"
    echo "  RATE_LIMITS_CID: $RATE_LIMITS_CID"

    forge script script/CreateStakingPool.s.sol:CreateStakingPool \
        --rpc-url "$RPC_URL" \
        --broadcast

    local pool_address
    pool_address=$(extract_pool_address "broadcast/CreateStakingPool.s.sol/1337/run-latest.json")

    if [ -n "$pool_address" ] && [ "$pool_address" != "null" ]; then
        echo "$name staking pool created at: $pool_address"
        echo "$pool_address"
    fi
}

# ============================================================================
# Main Script
# ============================================================================

echo "=== Query Staking Pool Deployment Script ==="
echo "Using RPC URL: $RPC_URL"
echo "Using W_TOKEN_ADDRESS: $W_TOKEN_ADDRESS"
echo "Using devnet test wallet (0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1)"

# Compute RATE_LIMITS_CID from the actual rate limits JSON file
if [ -f "$RATE_LIMITS_FILE" ]; then
    RATE_LIMITS_CID="0x$(shasum -a 256 "$RATE_LIMITS_FILE" | awk '{print $1}')"
    echo "Computed RATE_LIMITS_CID from $RATE_LIMITS_FILE"
else
    echo "Warning: Rate limits file not found at $RATE_LIMITS_FILE"
    echo "Using empty hash - rate limits will not work!"
    RATE_LIMITS_CID="0x0000000000000000000000000000000000000000000000000000000000000000"
fi
export RATE_LIMITS_CID

# Step 1: Deploy the factory
echo ""
echo "Step 1: Deploying QueryTypeStakerFactory..."
forge script script/Deploy.s.sol:Deploy \
    --rpc-url "$RPC_URL" \
    --broadcast

BROADCAST_FILE="broadcast/Deploy.s.sol/1337/run-latest.json"
if [ ! -f "$BROADCAST_FILE" ]; then
    echo "Error: Broadcast file not found at $BROADCAST_FILE"
    exit 1
fi

FACTORY_ADDRESS=$(jq -r '.transactions[0].contractAddress' "$BROADCAST_FILE")
if [ -z "$FACTORY_ADDRESS" ] || [ "$FACTORY_ADDRESS" == "null" ]; then
    echo "Error: Could not extract factory address"
    exit 1
fi

echo "Factory deployed at: $FACTORY_ADDRESS"
export FACTORY_ADDRESS

# Step 2: Create EVM pool
# Query Types 1, 2, 3 (binary 0b111 = 0x07)
# TODO: Investigate factory queryTypePools mapping - may need shifted format (0x0700) for decay rate encoding
echo ""
echo "Step 2: Creating EVM staking pool..."
EVM_POOL_ADDRESS=$(create_pool "EVM" \
    "0x0000000000000000000000000000000000000000000000000000000000000007" \
    "10000000000000000000" \
    "eth_call, eth_call_by_timestamp, eth_call_with_finality")

# Step 3: Create Solana pool
# Query Types 4, 5: bit(4-1)=3 -> 0x08, bit(5-1)=4 -> 0x10, combined = 0x18
echo ""
echo "Step 3: Creating Solana staking pool..."
SOLANA_POOL_ADDRESS=$(create_pool "Solana" \
    "0x0000000000000000000000000000000000000000000000000000000000000018" \
    "125000000000000000000" \
    "sol_account, sol_pda")

# Summary
echo ""
echo "=== Deployment Complete ==="
echo ""
echo "Factory Address: $FACTORY_ADDRESS"
echo ""
echo "EVM Pool: ${EVM_POOL_ADDRESS:-'(check logs above)'}"
echo "  Query types: eth_call (1), eth_call_by_timestamp (2), eth_call_with_finality (3)"
echo "  Min stake: 100 tokens"
echo ""
echo "Solana Pool: ${SOLANA_POOL_ADDRESS:-'(check logs above)'}"
echo "  Query types: sol_account (4), sol_pda (5)"
echo "  Min stake: 12,500 tokens"
