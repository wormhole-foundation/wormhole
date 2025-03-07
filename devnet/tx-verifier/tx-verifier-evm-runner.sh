#!/bin/sh

log_file="${ERROR_LOG_PATH:-/logs/error.log}"
error_pattern="${ERROR_PATTERN:-ERROR}"
TARGET=2

# start the guardian node
guardiand \
    transfer-verifier \
    evm \
    --rpcUrl ws://eth-devnet:8545 \
    --coreContract 0xC89Ce4735882C9F0f0FE26686c53074E09B0D550 \
    --tokenContract 0x0290FB167208Af455bB137780163b7B7a9a10C16 \
    --wrappedNativeContract 0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E \
    --logLevel=info \
    2> /tmp/error.log &

# start the test script
/tx-verifier-evm-tests.sh

# run the checks to see if the tests succeeded
current_count=$(grep -c "$error_pattern" "$log_file")    
echo "Found ${current_count} of ${TARGET} instances"

# if we found the requisite number of error messages, we can exit
if [ $current_count -ne $TARGET ]; then 
    echo "Tests failed. Only found ${current_count} of ${TARGET} required log messages"
    exit 1
fi

touch /tmp/success