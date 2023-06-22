#!/bin/bash

### maybe a validator is already running
pgrep -f solana-test-validator
if [ $? -eq 0 ]; then
    echo "solana-test-validator already running"
    exit 1;
fi

ROOT=$(dirname $0)

### prepare local validator
ARTIFACTS=$ROOT/artifacts
ACCOUNTS=$ROOT/sdk-tests/accounts
TEST=$ROOT/.test

solana-test-validator --reset \
  --bpf-program metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s $ARTIFACTS/mpl_token_metadata.so \
  --account-dir $ACCOUNTS \
  --ledger $TEST > validator.log 2>&1 &
sleep 5

### write program logs
PROGRAM_LOGS=$TEST/program-logs
mkdir -p $PROGRAM_LOGS

RPC=http://localhost:8899
solana logs agnnozV7x6ffAhi8xVhBd5dShfLnuUKKPEMX1tJ1nDC --url $RPC > $PROGRAM_LOGS/agnnozV7x6ffAhi8xVhBd5dShfLnuUKKPEMX1tJ1nDC &
solana logs bPPNmBhmHfkEFJmNKKCvwc1tPqBjzPDRwCw3yQYYXQa --url $RPC > $PROGRAM_LOGS/bPPNmBhmHfkEFJmNKKCvwc1tPqBjzPDRwCw3yQYYXQa &
solana logs caosnXgev6ceZQAUFk3hCjYtUwJLoKWwaoqZx9V9s9Q --url $RPC > $PROGRAM_LOGS/caosnXgev6ceZQAUFk3hCjYtUwJLoKWwaoqZx9V9s9Q &
solana logs KeccakSecp256k11111111111111111111111111111 --url $RPC > $PROGRAM_LOGS/KeccakSecp256k11111111111111111111111111111 &
solana logs metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s --url $RPC > $PROGRAM_LOGS/metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s &

### run tests
yarn run ts-mocha -p ./tsconfig.json -t 1000000 sdk-tests/*.ts

### nuke
pkill -f "solana logs"
pkill -f solana-test-validator
