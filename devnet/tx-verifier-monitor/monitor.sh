#!/bin/sh

log_file="${ERROR_LOG_PATH:-/logs/error.log}"
error_pattern="${ERROR_PATTERN:-ERROR}"
status_file="/logs/status"

# Wait for log file to exist and be non-empty
while [ ! -s "${log_file}" ]; do
    echo "Waiting for ${log_file} to be created and contain data..."
    sleep 5
done

# Initialize status
echo "RUNNING" > "$status_file"
echo "Monitoring file '${log_file}' for error pattern: '${error_pattern}'"

# Watch for changes in the log file. If we find the error pattern that means we have
# succeeded. (Transfer verifier should correctly detect errors.
inotifywait -m -e modify "${log_file}" | while read -r directory events filename; do
    if grep -q "$error_pattern" "$log_file"; then
        echo "SUCCESS" > "$status_file"
        echo "Found error pattern. Exiting."
        exit 0
    fi
done
