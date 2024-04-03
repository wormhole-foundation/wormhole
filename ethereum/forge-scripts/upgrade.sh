#!/bin/bash

#MNEMONIC= ./forge-scripts/upgrade.sh testnet Core blast

set -euo pipefail

network=$1
module=$2
chain=$3

SCRIPT=""
verify_module=""
case "$module" in
    Core)
        SCRIPT="DeployCoreImplementationOnly.s.sol:DeployCoreImplementationOnly"
        FILE="build-forge/Implementation.sol/Implementation.json"
        verify_module="core"
        ;;
    TokenBridge)
        SCRIPT="DeployTokenBridgeImplementationOnly.s.sol:DeployTokenBridgeImplementationOnly"
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
fi

if ! [ -f ./$ENV_FILE ]; then
  echo "Environment file \"${ENV_FILE}\" does not exist."
  exit 1
fi

. ./$ENV_FILE

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

echo "Implementation: ${implementation}"
echo "ret: ${ret}"

exit 0


forge script ./forge-scripts/${SCRIPT} \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast --slow