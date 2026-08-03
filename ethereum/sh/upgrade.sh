#!/bin/bash

# For testnet:
#MNEMONIC= GUARDIAN_MNEMONIC= ./sh/upgrade.sh testnet Core blast

# For mainnet:
#MNEMONIC= ./sh/upgrade.sh mainnet Core blast

set -euo pipefail

network="${1:-}"
module="${2:-}"
chain="${3:-}"

if [ -z "$network" ] || [ -z "$module" ] || [ -z "$chain" ]; then
  echo "Usage: MNEMONIC=... $0 <network> <module> <chain name>" >&2
  exit 1
fi

case "$network" in
  mainnet|testnet) ;;
  *)
    echo "unknown network $network, must be testnet or mainnet" >&2
    exit 1
    ;;
esac

if [ -z "${MNEMONIC:-}" ]; then
  echo "MNEMONIC unset"
  exit 1
fi

secret=$MNEMONIC
guardian_secret=""

if [ "$network" = testnet ]; then
  if [ -z "${GUARDIAN_MNEMONIC:-}" ]; then
    echo "GUARDIAN_MNEMONIC unset"
    exit 1
  fi
  guardian_secret=$GUARDIAN_MNEMONIC
fi

SCRIPT=""
verify_module=""
case "$module" in
    Core)
        SCRIPT="DeployCoreImplementationOnly.s.sol:DeployCoreImplementationOnly"
        SOLFILE="DeployCoreImplementationOnly.s.sol"
        FILE="build-forge/Implementation.sol/Implementation.json"
        verify_module="core"
        ;;
    TokenBridge)
        SCRIPT="DeployTokenBridgeImplementationOnly.s.sol:DeployTokenBridgeImplementationOnly"
        SOLFILE="DeployTokenBridgeImplementationOnly.s.sol"
        FILE="build-forge/BridgeImplementation.sol/BridgeImplementation.json"
        verify_module="token_bridge"
        ;;
    NFTBridge)
        echo "NFT bridge is not currently supported" >&2
        exit 1
        ;;
    *) echo "unknown module $module" >&2
    exit 1
       ;;
esac

if [ -z "${RPC_URL:-}" ]; then
  RPC_URL=$(worm info rpc "$network" "$chain")
fi

# Use the RPC's chain ID for the Foundry broadcast directory.
evm_chain_id=$(cast chain-id --rpc-url "$RPC_URL")
[[ "$evm_chain_id" =~ ^[0-9]+$ ]] || exit 1

if [ -z ${FORGE_ARGS+x} ]; then
  FORGE_ARGS=""
fi

ret=0
implementation=$(worm evm info -c "$chain" -m "$module" -n "$network" -i 2>/dev/null) || ret=$?

if [ $ret != 0 ]; then
  printf "☐ %s %s: skipping (no deployment available)\n" "$chain" "$module"
  exit 1
fi

ret=0
(./verify -n "$network" -c "$chain" "$FILE" "$implementation" > /dev/null) || ret=$?

# Only status 1 means bytecode mismatch; operational errors must not deploy.
if [ $ret = 0 ]; then
  printf "✔ %s %s: skipping (implementation matches same bytecode)\n" "$chain" "$module"
  exit
elif [ $ret != 1 ]; then
  exit $ret
fi

forge script ./forge-scripts/${SCRIPT} \
	--rpc-url "$RPC_URL" \
	--private-key "$secret" \
	--broadcast ${FORGE_ARGS}

returnInfo=$(cat "./broadcast/${SOLFILE}/${evm_chain_id}/run-latest.json")
# Extract the address values from 'returnInfo'
new_implementation=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

ret=0
(./verify -n "$network" -c "$chain" "$FILE" "$new_implementation" > /dev/null) || ret=$?

# Governance is allowed only after the deployed bytecode matches.
if [ $ret = 0 ]; then
  printf "✔ %s %s: deployed (%s)\n" "$chain" "$module" "$new_implementation"
else
  printf "✘ %s %s: deployed (%s) but failed to match bytecode\n"  "$chain" "$module" "$new_implementation"
  exit 1
fi

if [ "$network" = testnet ]; then
  # Keep generated governance data in one submit argument.
  upgrade_vaa=$(worm generate upgrade -c "$chain" -a "$new_implementation" -m "$module" -g "$guardian_secret")
  worm submit "$upgrade_vaa" -n "$network"
else
  echo "../scripts/contract-upgrade-governance.sh -c $chain -m $verify_module -a $new_implementation"
fi
