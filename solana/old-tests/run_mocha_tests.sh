#!/bin/bash

### maybe a validator is already running
pgrep -f solana-test-validator
if [ $? -eq 0 ]; then
    echo "solana-test-validator already running"
    exit 1;
fi

TEST_ROOT=$(dirname $0)
ROOT=$TEST_ROOT/..

### prepare local validator
ARTIFACTS=$ROOT/target/deploy
ACCOUNTS=$TEST_ROOT/test-accounts

MPL_TOKEN_METADATA_PUBKEY=metaqbxxUerdq28cj1RbAWkYQm3ybzjb6a8bt518x1s
MPL_TOKEN_METADATA_BPF=$TEST_ROOT/mpl_token_metadata.so
if [ ! -f $MPL_TOKEN_METADATA_BPF ]; then
  echo "> Fetching MPL Token Metadata program from mainnet-beta"
  solana program dump -u m $MPL_TOKEN_METADATA_PUBKEY $MPL_TOKEN_METADATA_BPF
fi

### Fetch Wormhole programs from main branch
EXISTING_CORE_BRIDGE_BPF=$TEST_ROOT/existing_core_bridge.so
EXISTING_TOKEN_BRIDGE_BPF=$TEST_ROOT/existing_token_bridge.so
EXISTING_NFT_BRIDGE_BPF=$TEST_ROOT/existing_nft_bridge.so

if [ ! -f $EXISTING_CORE_BRIDGE_BPF ]; then
git clone \
	  --depth 1 \
	  --branch main \
	  --filter=blob:none \
	  https://github.com/wormhole-foundation/wormhole \
	  wormhole-main > /dev/null 2>&1
	cd wormhole-main/solana
	echo "> Building Wormhole Solana bridges"
  DOCKER_BUILDKIT=1 docker build \
			-f Dockerfile \
			--build-arg BRIDGE_ADDRESS=agnnozV7x6ffAhi8xVhBd5dShfLnuUKKPEMX1tJ1nDC \
			-o artifacts-localnet .
  cd ../..
  cp wormhole-main/solana/artifacts-localnet/bridge.so $EXISTING_CORE_BRIDGE_BPF
  cp wormhole-main/solana/artifacts-localnet/token_bridge.so $EXISTING_TOKEN_BRIDGE_BPF
  cp wormhole-main/solana/artifacts-localnet/nft_bridge.so $EXISTING_NFT_BRIDGE_BPF
	rm -rf wormhole-main
fi

TEST=$TEST_ROOT/.test

VALIDATOR=$(which solana-test-validator)
echo $VALIDATOR
echo $($VALIDATOR --version)

$VALIDATOR --reset \
  --bpf-program $MPL_TOKEN_METADATA_PUBKEY $MPL_TOKEN_METADATA_BPF \
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
npx ts-mocha -p ./tsconfig.json -t 1000000 tests/[0-9]*/[0-9]*.ts

### nuke
pkill -f "solana logs"
pkill -f solana-test-validator
