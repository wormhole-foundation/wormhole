#!/bin/bash

### maybe a validator is already running
pgrep -f solana-test-validator
if [ $? -eq 0 ]; then
    echo "solana-test-validator already running"
    exit 1;
fi

ROOT=$(dirname $0)
SOLANA=$ROOT/../../solana

### prepare local validator
ARTIFACTS=$SOLANA/artifacts-devnet
ACCOUNTS=$ROOT/sdk-tests/accounts
TEST=$ROOT/.test

solana-test-validator --reset \
  --bpf-program Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o $ARTIFACTS/bridge.so \
  --account FKoMTctsC7vJbEqyRiiPskPnuQx2tX1kurmvWByq5uZP $ACCOUNTS/bridge_config.json \
  --account 6MxkvoEwgB9EqQRLNhvYaPGhfcLtBtpBqdQugr3AZUgD $ACCOUNTS/guardian_set.json \
  --account GXBsgBD3LDn3vkRZF6TfY5RqgajVZ4W5bMAdiAaaUARs $ACCOUNTS/fee_collector.json \
  --bpf-program B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE $ARTIFACTS/token_bridge.so \
  --account 3GwVs8GSLdo4RUsoXTkGQhojauQ1sXcDNjm7LSDicw19 $ACCOUNTS/token_config.json \
  --account 7UqWgfVW1TrjrqauMfDoNMcw8kEStSsQXWNoT2BbhDS5 $ACCOUNTS/ethereum_token_bridge.json \
  --bpf-program metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s $SOLANA/modules/token_bridge/token-metadata/spl_token_metadata.so \
  --ledger $TEST > validator.log 2>&1 &
sleep 2

### write program logs
PROGRAM_LOGS=$TEST/program-logs
mkdir -p $PROGRAM_LOGS

RPC=http://localhost:8899
solana logs Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o --url $RPC > $PROGRAM_LOGS/Bridge1p5gheXUvJ6jGWGeCsgPKgnE3YgdGKRVCMY9o &
solana logs B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE --url $RPC > $PROGRAM_LOGS/B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE &
solana logs KeccakSecp256k11111111111111111111111111111 --url $RPC > $PROGRAM_LOGS/KeccakSecp256k11111111111111111111111111111 &


### run tests
yarn run ts-mocha -p ./tsconfig.json -t 1000000 sdk-tests/*.ts

### nuke
pkill -f "solana logs"
pkill -f solana-test-validator
