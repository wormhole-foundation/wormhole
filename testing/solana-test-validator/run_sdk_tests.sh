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
  --bpf-program B6RHG3mfcckmrYN1UhmJzyS1XX3fZKbkeUcpJe9Sy3FE $ARTIFACTS/token_bridge.so \
  --bpf-program NFTWqJR8YnRVqPDvTJrYuLrQDitTG5AScqbeghi4zSA $ARTIFACTS/nft_bridge.so \
  --bpf-program CP1co2QMMoDPbsmV7PGcUTLFwyhgCgTXt25gLQ5LewE1 $ARTIFACTS/cpi_poster.so \
  --bpf-program Ex9bCdVMSfx7EzB3pgSi2R4UHwJAXvTw18rBQm5YQ8gK $ARTIFACTS/wormhole_migration.so \
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
