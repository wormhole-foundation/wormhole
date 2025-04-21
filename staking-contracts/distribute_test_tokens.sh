#!/bin/bash

# Distribute Test Tokens Script
# Transfers test tokens to common testing accounts for easier testing

set -e  # Exit on error

# ============================================================================
# Configuration
# ============================================================================
RPC_URL="${RPC_URL:-http://localhost:8545}"
PRIVATE_KEY="${PRIVATE_KEY:-0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d}"
W_TOKEN_ADDRESS="${W_TOKEN_ADDRESS:-0x2D8BE6BF0baA74e0A907016679CaE9190e80dD0A}"

# Test accounts that need tokens (from devnet-consts.json ganache defaults)
DEPLOYER_ADDRESS="0x90F8bf6A479f320ead074411a4B0e7944Ea8c9C1"  # Already has tokens (deployer)

TEST_ACCOUNTS=(
    "0xFFcf8FDEE72ac11b5c542428B35EEF5769C409f0"  # Account 1 - Delegator 1 (delegation tests, 100k tokens)
    "0x22d491Bde2303f2f43325b2108D26f1eAbA1e32b"  # Account 2 - Delegator 2 (delegation tests, 101k tokens)
    "0x855FA758c77D68a04990E992aA4dcdeF899F654A"  # Account 11 - Delegator 3 (delegation tests, 155k tokens)
    "0xd03ea8624C8C5987235048901fB614fDcA89b117"  # Account 4 - Staker (ratelimit tests, needs tokens for staking)
)

# Transfer 50,000 tokens to each account
TRANSFER_AMOUNT_TOKENS="${TRANSFER_AMOUNT_TOKENS:-50000}"
TRANSFER_AMOUNT_WEI="${TRANSFER_AMOUNT_TOKENS}000000000000000000"  # Convert to wei (tokens * 10^18)

# ============================================================================
# Main Script
# ============================================================================

echo "=== Test Token Distribution Script ==="
echo "Using RPC URL: $RPC_URL"
echo "Using W_TOKEN_ADDRESS: $W_TOKEN_ADDRESS"
echo "Transfer amount: $TRANSFER_AMOUNT_TOKENS tokens per account"
echo ""

# Check deployer has enough tokens
echo "Checking deployer token balance..."
DEPLOYER_BALANCE=$(cast call "$W_TOKEN_ADDRESS" "balanceOf(address)(uint256)" "$DEPLOYER_ADDRESS" --rpc-url "$RPC_URL")
# Extract just the number, removing any scientific notation annotation
DEPLOYER_BALANCE=$(echo "$DEPLOYER_BALANCE" | awk '{print $1}')
DEPLOYER_BALANCE_TOKENS=$(cast --to-unit "$DEPLOYER_BALANCE" ether)
echo "Deployer ($DEPLOYER_ADDRESS) balance: $DEPLOYER_BALANCE_TOKENS tokens"

TOTAL_NEEDED=$((TRANSFER_AMOUNT_TOKENS * ${#TEST_ACCOUNTS[@]}))
echo "Total tokens needed: $TOTAL_NEEDED tokens"
echo ""

if (( $(echo "$DEPLOYER_BALANCE_TOKENS < $TOTAL_NEEDED" | bc -l) )); then
    echo "Warning: Deployer may not have enough tokens!"
    echo "Proceeding anyway - transfers will fail if insufficient balance"
    echo ""
fi

# Transfer to each test account
for account in "${TEST_ACCOUNTS[@]}"; do
    echo "Processing $account..."

    # Check current balance
    CURRENT_BALANCE=$(cast call "$W_TOKEN_ADDRESS" "balanceOf(address)(uint256)" "$account" --rpc-url "$RPC_URL" 2>/dev/null || echo "0")
    CURRENT_BALANCE=$(echo "$CURRENT_BALANCE" | awk '{print $1}')
    CURRENT_BALANCE_TOKENS=$(cast --to-unit "$CURRENT_BALANCE" ether 2>/dev/null || echo "0")
    echo "  Current balance: $CURRENT_BALANCE_TOKENS tokens"

    # Transfer tokens
    echo "  Transferring $TRANSFER_AMOUNT_TOKENS tokens..."
    TX_HASH=$(cast send "$W_TOKEN_ADDRESS" \
        "transfer(address,uint256)(bool)" \
        "$account" \
        "$TRANSFER_AMOUNT_WEI" \
        --private-key "$PRIVATE_KEY" \
        --rpc-url "$RPC_URL" \
        --json 2>/dev/null | jq -r '.transactionHash')

    if [ -n "$TX_HASH" ] && [ "$TX_HASH" != "null" ]; then
        # Verify new balance
        NEW_BALANCE=$(cast call "$W_TOKEN_ADDRESS" "balanceOf(address)(uint256)" "$account" --rpc-url "$RPC_URL")
        NEW_BALANCE=$(echo "$NEW_BALANCE" | awk '{print $1}')
        NEW_BALANCE_TOKENS=$(cast --to-unit "$NEW_BALANCE" ether)
        echo "  ✓ Transfer complete (tx: ${TX_HASH:0:10}...)"
        echo "  ✓ New balance: $NEW_BALANCE_TOKENS tokens"
    else
        echo "  ✗ Transfer failed!"
    fi

    echo ""
done

echo "=== Token Distribution Complete ==="
echo ""
echo "Summary:"
echo "  Token Address: $W_TOKEN_ADDRESS"
echo "  Amount per account: $TRANSFER_AMOUNT_TOKENS tokens"
echo "  Accounts funded: ${#TEST_ACCOUNTS[@]}"
echo ""
echo "Test accounts with balances:"
for account in "${TEST_ACCOUNTS[@]}"; do
    BALANCE=$(cast call "$W_TOKEN_ADDRESS" "balanceOf(address)(uint256)" "$account" --rpc-url "$RPC_URL" 2>/dev/null || echo "0")
    BALANCE=$(echo "$BALANCE" | awk '{print $1}')
    BALANCE_TOKENS=$(cast --to-unit "$BALANCE" ether 2>/dev/null || echo "0")
    echo "  $account: $BALANCE_TOKENS tokens"
done
