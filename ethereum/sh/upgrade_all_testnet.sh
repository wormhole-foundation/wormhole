#!/bin/bash

# Upgrade Core and TokenBridge contracts on all chains:
#MNEMONIC= GUARDIAN_MNEMONIC= ./sh/upgrade_all_testnet.sh

# Upgrade TokenBridge on a few chains:
#MNEMONIC= GUARDIAN_MNEMONIC= CHAINS="avalanche polygon oasis" MODULES=TokenBridge ./sh/upgrade_all_testnet.sh

# Chains that require nondefault Foundry options should be grouped by the same
# FORGE_ARGS value using the CHAINS selector above.

if [ "${CHAINS}X" == "X" ]; then
  CHAINS=$(worm evm chains) || exit 1
  # Empty enumeration must not look like a successful bulk run.
  [ -n "$CHAINS" ] || exit 1
fi

if [ "${MODULES}X" == "X" ]; then
  MODULES=(Core TokenBridge)
fi

# Stop on the first failed child and resolve an RPC separately per chain.
set -euo pipefail
unset RPC_URL
network=testnet

for module in ${MODULES[@]}; do
  for chain in ${CHAINS[@]}; do
    echo "Upgrading ${chain} ${module} ********************************************************************"
    ./sh/upgrade.sh "$network" "$module" "$chain"
  done
done
