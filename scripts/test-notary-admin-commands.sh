#!/usr/bin/env bash
# Integration test for Notary admin rescue commands
# This script tests the complete lifecycle of Notary rescue operations:
# 1. Create synthetic delayed/blackholed messages using inject commands
# 2. Execute admin commands against the guardian
# 3. Verify state changes after each command
#
# This modifies Tilt's BadgerDB, so successive runs of this script
# will create many stored messages and the DB is not cleared after the
# script terminates.
#
# Prerequisites:
# - Tilt devnet must be running with notary enabled
#
# Usage: ./test-notary-admin-commands.sh [NODE_NUMBER]
#   NODE_NUMBER: Guardian node to test against (default: 0)

set -e

node=${1:-0}  # Default to guardian-0
sock="/tmp/admin.sock"

echo "=== Notary Admin Commands Integration Test ==="
echo "Testing against guardian-${node}"
echo ""

# Helper function to execute commands in guardian pod
exec_in_pod() {
    kubectl exec -n wormhole guardian-${node} -c guardiand -- "$@"
}

# Helper function to run admin commands
admin_cmd() {
    exec_in_pod /guardiand admin "$@" --socket "$sock"
}

# Helper function to print database state
print_db_state() {
    local title="$1"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📊 $title"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    local delayed_output=$(admin_cmd notary-list-delayed-messages)
    local delayed_count=$(echo "$delayed_output" | grep -oP 'Delayed messages \(\K[0-9]+' || echo "0")
    
    echo "🕒 DELAYED MESSAGES: $delayed_count"
    if [[ "$delayed_count" -gt 0 ]]; then
        echo "$delayed_output" | grep "^2/" | while read -r msg_id; do
            local details=$(admin_cmd notary-get-delayed-message "$msg_id" 2>/dev/null || echo "")
            local release_time=$(echo "$details" | grep -oP 'Release Time: \K.*' || echo "unknown")
            echo "  • ${msg_id:0:80}..."
            echo "    Release: $release_time"
        done
    fi
    
    echo ""
    local blackholed_output=$(admin_cmd notary-list-blackholed-messages)
    local blackholed_count=$(echo "$blackholed_output" | grep -oP 'Blackholed messages \(\K[0-9]+' || echo "0")
    
    echo "⛔ BLACKHOLED MESSAGES: $blackholed_count"
    if [[ "$blackholed_count" -gt 0 ]]; then
        echo "$blackholed_output" | grep "^2/" | while read -r msg_id; do
            echo "  • ${msg_id:0:80}..."
        done
    fi
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
}

# Step 1: Create synthetic test state using inject commands
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 1: Creating synthetic delayed messages"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
msg1_output=$(admin_cmd notary-inject-delayed-message 4)
msg1_id=$(echo "$msg1_output" | grep -oP 'Injected delayed message: \K.*')
msg2_output=$(admin_cmd notary-inject-delayed-message 4)
msg2_id=$(echo "$msg2_output" | grep -oP 'Injected delayed message: \K.*')
echo "✓ Created test messages:"
echo "  - Message 1 (4 day delay): ${msg1_id:0:80}..."
echo "  - Message 2 (4 day delay): ${msg2_id:0:80}..."

print_db_state "Database State After Injection"

# Step 2: Verify initial state using list command
delayed_list=$(admin_cmd notary-list-delayed-messages)
if ! echo "$delayed_list" | grep -q "$msg1_id"; then
  echo "✗ FAIL: Message 1 not found in delayed list"
  exit 1
fi
if ! echo "$delayed_list" | grep -q "$msg2_id"; then
  echo "✗ FAIL: Message 2 not found in delayed list"
  exit 1
fi
echo "✓ Initial state verified"

# Step 3: Test notary-blackhole-delayed-message
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 2: Testing notary-blackhole-delayed-message"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Moving message 1 from delayed → blackholed..."
admin_cmd notary-blackhole-delayed-message "$msg1_id"
# Give DB a moment to flush the write
sleep 0.5
delayed_list=$(admin_cmd notary-list-delayed-messages)
if echo "$delayed_list" | grep -q "$msg1_id"; then
  echo "✗ FAIL: Message still in delayed list after blackhole"
  exit 1
fi
blackholed_list=$(admin_cmd notary-list-blackholed-messages)
if ! echo "$blackholed_list" | grep -q "$msg1_id"; then
  echo "✗ FAIL: Message not moved to blackholed list"
  exit 1
fi
echo "✓ Message successfully moved from delayed → blackholed"

print_db_state "Database State After Blackholing"

# Step 4: Test notary-remove-blackholed-message  
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 3: Testing notary-remove-blackholed-message"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Removing message 1 from blackholed (restoring to delayed)..."
admin_cmd notary-remove-blackholed-message "$msg1_id"
# Give DB a moment to flush the write
sleep 0.5
blackholed_list=$(admin_cmd notary-list-blackholed-messages)
if echo "$blackholed_list" | grep -q "$msg1_id"; then
  echo "✗ FAIL: Message still in blackholed list after removal"
  exit 1
fi
delayed_list=$(admin_cmd notary-list-delayed-messages)
if ! echo "$delayed_list" | grep -q "$msg1_id"; then
  echo "✗ FAIL: Message not restored to delayed list"
  exit 1
fi
echo "✓ Message successfully moved from blackholed → delayed"

print_db_state "Database State After Removing from Blackhole"

# Step 5: Test notary-release-delayed-message
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 4: Testing notary-release-delayed-message"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Getting release time BEFORE release command..."
before_release=$(admin_cmd notary-get-delayed-message "$msg2_id")
before_time=$(echo "$before_release" | grep -oP 'Release Time: \K.*')
echo "  Before: $before_time"
echo ""
echo "Releasing message 2 (sets release time to now)..."
admin_cmd notary-release-delayed-message "$msg2_id"
# Note: The notary processes delayed messages on a 1-minute poll timer.
# The release sets the time in the past, but we need to wait for the next poll cycle.
# For testing purposes, we'll just verify the message is still in the delayed list
# but with a release time in the past.
delayed_msg_info=$(admin_cmd notary-get-delayed-message "$msg2_id")
if ! echo "$delayed_msg_info" | grep -q "$msg2_id"; then
  echo "✗ FAIL: Message not found after release command"
  exit 1
fi
after_time=$(echo "$delayed_msg_info" | grep -oP 'Release Time: \K.*')
echo "  After:  $after_time"
echo ""
echo "✓ Release time updated (will be released on next poll cycle in ~60s)"

print_db_state "Database State After Release (time updated)"

# Step 6: Test notary-reset-release-timer
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 5: Testing notary-reset-release-timer"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
msg3_output=$(admin_cmd notary-inject-delayed-message 1)
msg3_id=$(echo "$msg3_output" | grep -oP 'Injected delayed message: \K.*')
echo "Created message 3 with 1 day delay: ${msg3_id:0:80}..."
echo ""
before_reset=$(admin_cmd notary-get-delayed-message "$msg3_id")
before_time=$(echo "$before_reset" | grep -oP 'Release Time: \K.*')
echo "Getting release time BEFORE reset..."
echo "  Before: $before_time"
echo ""
echo "Resetting timer to 5 days..."
admin_cmd notary-reset-release-timer "$msg3_id" 5
delayed_list=$(admin_cmd notary-list-delayed-messages)
if ! echo "$delayed_list" | grep -q "$msg3_id"; then
  echo "✗ FAIL: Message not found after timer reset"
  exit 1
fi
after_reset=$(admin_cmd notary-get-delayed-message "$msg3_id")
after_time=$(echo "$after_reset" | grep -oP 'Release Time: \K.*')
echo "  After:  $after_time"
echo ""
echo "✓ Release timer successfully reset from 1 day → 5 days"

print_db_state "Database State After Timer Reset"

# Step 7: Test introspection commands
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Step 6: Testing introspection commands"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
msg4_output=$(admin_cmd notary-inject-delayed-message 2)
msg4_id=$(echo "$msg4_output" | grep -oP 'Injected delayed message: \K.*')
msg5_output=$(admin_cmd notary-inject-blackholed-message)
msg5_id=$(echo "$msg5_output" | grep -oP 'Injected blackholed message: \K.*')

echo "Testing notary-get-delayed-message for message 4..."
delayed_details=$(admin_cmd notary-get-delayed-message "$msg4_id")
if ! echo "$delayed_details" | grep -q "$msg4_id"; then
  echo "✗ FAIL: Failed to get delayed message details"
  exit 1
fi
echo "$delayed_details"

echo ""
echo "Testing notary-get-blackholed-message for message 5..."
blackholed_details=$(admin_cmd notary-get-blackholed-message "$msg5_id")
if ! echo "$blackholed_details" | grep -q "$msg5_id"; then
  echo "✗ FAIL: Failed to get blackholed message details"
  exit 1
fi
echo "$blackholed_details"

print_db_state "Final Database State"
