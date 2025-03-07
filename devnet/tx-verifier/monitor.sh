#!/bin/sh
log_file="${ERROR_LOG_PATH:-/logs/error.log}"
error_pattern="${ERROR_PATTERN:-ERROR}"
poll_interval=5
TARGET=2

# Wait for log file to exist and be non-empty
while [ ! -s "${log_file}" ]; do
    echo "Waiting for ${log_file} to be created and contain data..."
    sleep 5
done

echo "Monitoring file '${log_file}' for ${TARGET} total instance(s) of error pattern: '${error_pattern}'"

# Poll until we find the target number of instances
while true; do
    current_count=$(grep -c "$error_pattern" "$log_file")
    
    echo "Found ${current_count} of ${TARGET} instances so far."
    
    if [ $current_count -eq $TARGET ]; then 
        echo "SUCCESS: Found ${TARGET} instances of error pattern. Exiting."
        exit 0
    fi
    
    if [ $current_count -gt $TARGET ]; then 
        echo "Wanted ${TARGET} instances of error pattern but got ${current_count}. This is probably a bug."
        exit 1
    fi
    
    sleep $poll_interval
done
