#!/bin/bash

# Deploy and Create Staking Pools Script
# This script deploys the QueryTypeStakerFactory and creates both EVM and Solana staking pools
#
# Pools created:
#   1. EVM Pool - eth_call, eth_call_by_timestamp, eth_call_with_finality (types 1, 2, 3)
#   2. Solana Pool - sol_account, sol_pda (types 4, 5)
#
# Environment variables:
#   ETH_RPC_URL - RPC endpoint (default: http://localhost:8545)
#   RATE_LIMITS_FILE - Path to rate limits JSON file (default: ./ccq-rate-limits.json)
#   EVM_DECAY_RATE - Decay rate for EVM pool, 0-100 (default: 5)
#   SOLANA_DECAY_RATE - Decay rate for Solana pool, 0-100 (default: 5)
#
# Uses devnet configuration from ../wormhole/scripts/devnet-consts.json

set -e  # Exit on error

# ============================================================================
# Configuration
# ============================================================================
RPC_URL="${ETH_RPC_URL:-http://localhost:8545}"
# Use account 10 (0x610Bb1573d1046FCb8A70Bbbd395754cD57C2b60) to avoid nonce conflicts
# with account 0 (deployer) and account 9 (accountant tests)
PRIVATE_KEY="0x77c5495fbb039eed474fc940f29955ed0531693cc9212911efd35dff0373153f"
W_TOKEN_ADDRESS="0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A"
RATE_LIMITS_FILE="${RATE_LIMITS_FILE:-./ccq-rate-limits.json}"

# Event topic for CreateQueryTypeStakingPool(bytes32 indexed queryType, address indexed poolAddress)
CREATE_POOL_EVENT_TOPIC="0x34d4b91c04bf254b71c435c46e26f1c0b6ec05b426b3bbebb5a80d3e71c030db"

# Common pool parameters
# Query Type Encoding:
#   Bits 0-7 (lowest byte): Decay rate (0-100)
#   Bits 8+: Query type flags (bit field of supported query types)
INITIAL_ENTRY=${INITIAL_ENTRY:-"0x0000000000000000000000000000000000000000000000000000000000000001"}

export PRIVATE_KEY W_TOKEN_ADDRESS

# ============================================================================
# Helper Functions
# ============================================================================

# Encode query type with decay rate
# Usage: encode_query_type "query_flags_hex" "decay_rate"
# Example: encode_query_type "0x07" "50" -> 0x0732
encode_query_type() {
    local query_flags="$1"
    local decay_rate="$2"

    # Validate decay rate
    if [ "$decay_rate" -lt 0 ] || [ "$decay_rate" -gt 100 ]; then
        echo "Error: Decay rate must be between 0 and 100" >&2
        exit 1
    fi

    # Convert hex query flags to decimal
    local flags_dec=$((query_flags))

    # Shift flags left by 8 bits and OR with decay rate
    # Formula: (flags << 8) | decay_rate
    local encoded=$(( (flags_dec << 8) | decay_rate ))

    # Convert back to 32-byte hex
    printf "0x%064x" "$encoded"
}

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
# Usage: create_pool "Pool Name" "query_flags_hex" "decay_rate" "min_stake_wei_or_empty" "description"
create_pool() {
    local name="$1"
    local query_flags="$2"
    local decay_rate="$3"
    local min_stake="$4"
    local description="$5"

    # Encode the query type with decay rate
    local query_type
    query_type=$(encode_query_type "$query_flags" "$decay_rate")

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
    echo "  QUERY_FLAGS: $query_flags ($description)"
    echo "  DECAY_RATE: $decay_rate%"
    echo "  ENCODED_QUERY_TYPE: $query_type"
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
echo "Using devnet test wallet account 10 (0x610Bb1573d1046FCb8A70Bbbd395754cD57C2b60)"

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
# Using decay rate from environment or default to 5%
EVM_DECAY_RATE=${EVM_DECAY_RATE:-5}
echo ""
echo "Step 2: Creating EVM staking pool..."
EVM_POOL_ADDRESS=$(create_pool "EVM" \
    "0x07" \
    "$EVM_DECAY_RATE" \
    "10000000000000000000" \
    "eth_call, eth_call_by_timestamp, eth_call_with_finality")

# Step 3: Create Solana pool
# Query Types 4, 5: bit(4-1)=3 -> 0x08, bit(5-1)=4 -> 0x10, combined = 0x18
# Using decay rate from environment or default to 5%
SOLANA_DECAY_RATE=${SOLANA_DECAY_RATE:-5}
echo ""
echo "Step 3: Creating Solana staking pool..."
SOLANA_POOL_ADDRESS=$(create_pool "Solana" \
    "0x18" \
    "$SOLANA_DECAY_RATE" \
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
echo "  Decay rate: ${EVM_DECAY_RATE}%"
echo "  Min stake: 10 tokens"
echo ""
echo "Solana Pool: ${SOLANA_POOL_ADDRESS:-'(check logs above)'}"
echo "  Query types: sol_account (4), sol_pda (5)"
echo "  Decay rate: ${SOLANA_DECAY_RATE}%"
echo "  Min stake: 125 tokens"
echo ""
echo "=== Verifying Deployment ==="
echo "Allowing anvil state to stabilize..."
sleep 2
