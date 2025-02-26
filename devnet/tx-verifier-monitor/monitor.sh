#!/bin/sh
log_file="${ERROR_LOG_PATH:-/logs/error.log}"
error_pattern="${ERROR_PATTERN:-ERROR}"
status_file="${STATUS_FILE:-/logs/status}"
timeout="${TIMEOUT_SECONDS:-500}"
poll_interval=2  # Check for changes every 2 seconds

# Compare with the integration test script: devnet/tx-verifier-monitor/transfer-verifier-test.sh
TARGET=2

echo "Beginning test with configured timeout of ${timeout} seconds"

# Wait for log file to exist and be non-empty
while [ ! -s "${log_file}" ]; do
    echo "Waiting for ${log_file} to be created and contain data..."
    sleep 5
done

# Calculate end time
end_time=$(($(date +%s) + timeout))

echo "Monitoring file '${log_file}' for ${TARGET} total instance(s) of error pattern: '${error_pattern}'"

# Initialize counters
count=0
prev_count=0

# Poll until timeout or success
while [ $(date +%s) -lt $end_time ]; do
    # Count current instances of the pattern
    current_count=$(grep -c "$error_pattern" "$log_file")
    
    # Check if we found more instances
    if [ $current_count -gt $prev_count ]; then
        count=$((count + (current_count - prev_count)))
        echo "Found error pattern: '${error_pattern}'"
        echo "${count} instance(s) found so far."
        prev_count=$current_count
    fi

    # Check for success condition
    if [ $current_count -eq $TARGET ]; then 
        echo "SUCCESS: Found ${current_count} instances of error pattern. Exiting."
        echo "SUCCESS" > "$status_file"
        exit 0
    fi

    if [ $current_count -gt $TARGET ]; then 
        echo "Wanted ${TARGET} instances of error pattern but got ${current_count}. This is probably a bug."
    fi
    
    # Sleep briefly before polling again
    sleep $poll_interval
done

# If we reach here, the test timed out
echo "Test failed: Timed out after ${timeout} seconds."
exit 1
