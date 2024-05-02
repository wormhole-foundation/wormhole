#!/bin/bash

# For testnet: 
#MNEMONIC= GUARDIAN_MNEMONIC= ./sh/upgrade.sh testnet Core blast

# For mainnet: 
#MNEMONIC= ./sh/upgrade.sh mainnet Core blast

set -euo pipefail

network=$1
module=$2
chain=$3

secret=$MNEMONIC
guardian_secret=""

if [ "$network" = testnet ]; then
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

ENV_FILE="env/.env.${chain}"
if [ "$network" = testnet ]; then
  ENV_FILE="${ENV_FILE}.testnet"
else
  ENV_FILE="${ENV_FILE}.mainnet"
fi

if ! [ -f ./$ENV_FILE ]; then
  echo "Environment file \"${ENV_FILE}\" does not exist."
  exit 1
fi

. ./$ENV_FILE

exit 1

[[ -z $INIT_EVM_CHAIN_ID ]] && { echo "Missing INIT_EVM_CHAIN_ID"; exit 1; }
[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }

if [ -z ${RPC_URL+x} ]; then
  ret=0
  RPC_URL=$(worm info rpc "$network" "$chain" 2>/dev/null) || ret=$?
  if [ $ret != 0 ]; then
    echo "Missing RPC_URL and \"worm info rpc\" does not support this chain."
    exit 1
  fi
fi

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
(./verify -n "$network" -c "$chain" $FILE "$implementation" > /dev/null) || ret=$?

if [ $ret = 0 ]; then
  printf "✔ %s %s: skipping (implementation matches same bytecode)\n" "$chain" "$module"
  exit
fi

forge script ./forge-scripts/${SCRIPT} \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast ${FORGE_ARGS}

returnInfo=$(cat ./broadcast/${SOLFILE}/$INIT_EVM_CHAIN_ID/run-latest.json)
# Extract the address values from 'returnInfo'
new_implementation=$(jq -r '.returns.deployedAddress.value' <<< "$returnInfo")

ret=0
(./verify -n "$network" -c "$chain" $FILE "$new_implementation" > /dev/null) || ret=$?

if [ $ret = 0 ]; then
  printf "✔ %s %s: deployed (%s)\n" "$chain" "$module" "$new_implementation"
else
  printf "✘ %s %s: deployed (%s) but failed to match bytecode\n"  "$chain" "$module" "$new_implementation"
  exit 1
fi

if [ "$network" = testnet ]; then
  worm submit $(worm generate upgrade -c "$chain" -a "$new_implementation" -m "$module" -g "$guardian_secret") -n "$network"
else
  echo "../scripts/contract-upgrade-governance.sh -c $chain -m $verify_module -a $new_implementation"
fi