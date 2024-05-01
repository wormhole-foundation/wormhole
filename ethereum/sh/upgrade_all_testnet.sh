#!/bin/bash

# Upgrade Core, TokenBridge and NFTBridge contracts on all chains:
#MNEMONIC= GUARDIAN_MNEMONIC= ./sh/upgrade_all_testnet.sh

# Upgrade TokenBridge on a few chains chains:
#MNEMONIC= GUARDIAN_MNEMONIC= CHAINS="avalanche polygon oasis" MODULES=TokenBridge ./sh/upgrade_all_testnet.sh

# Upgrade Core and TokenBridge contracts on all chains:
#MNEMONIC= GUARDIAN_MNEMONIC= MODULES="Core TokenBridge" ./sh/upgrade_all_testnet.sh

if [ "${CHAINS}X" == "X" ]; then
  CHAINS=$(worm evm chains)
fi

if [ "${MODULES}X" == "X" ]; then
  MODULES=(Core TokenBridge NFTBridge)
fi

set -uo pipefail
network=testnet

for module in ${MODULES[@]}; do
  for chain in ${chains[@]}; do
    echo "Upgrading ${chain} ${module} ********************************************************************"
    ./sh/upgrade.sh "$network" "$module" "$chain"
  done
done
