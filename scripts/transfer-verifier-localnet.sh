#!/usr/bin/env bash
# Before running this script, ensure that anvil is running, e.g.:
#
# anvil --host 0.0.0.0 --base-fee 0 --fork-url $(worm info rpc mainnet ethereum) --mnemonic "myth like bonus scare over problem client lizard pioneer submit female collect" --fork-block-number 20641947 --fork-chain-id 1 --chain-id 1 --steps-tracing --auto-impersonate

set -xeuo pipefail

# mainnet 
# CORE_CONTRACT="0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
# TOKEN_BRIDGE_CONTRACT="0x3ee18B2214AFF97000D974cf647E7C347E8fa585"
# WRAPPED_NATIVE_CONTRACT="0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
# devnet 
CORE_CONTRACT="0xC89Ce4735882C9F0f0FE26686c53074E09B0D550"
TOKEN_BRIDGE_CONTRACT="0x0290FB167208Af455bB137780163b7B7a9a10C16"
WRAPPED_NATIVE_CONTRACT="0xDDb64fE46a91D46ee29420539FC25FD07c5FEa3E"

# Needs to be websockets so that the eth connector can get notifications
ETH_RPC_DEVNET="ws://localhost:8545" # from Tilt, via Anvil

# RPC="${ALCHEMY_RPC}"
RPC="${ETH_RPC_DEVNET}"

LOG_LEVEL="debug"

# Do `make node` first to compile transfer-verifier into guardiand. Note that the telemetry parameters are omitted here.
./build/bin/guardiand transfer-verifier evm \
   --rpcUrl "${RPC}" \
   --coreContract "${CORE_CONTRACT}" \
   --tokenContract "${TOKEN_BRIDGE_CONTRACT}" \
   --wrappedNativeContract "${WRAPPED_NATIVE_CONTRACT}" \
   --logLevel "${LOG_LEVEL}"
