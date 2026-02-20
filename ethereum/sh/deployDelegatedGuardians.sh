#!/bin/bash

. .env


[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }
[[ -z $RPC_URL ]] && { echo "Missing RPC_URL"; exit 1; }
[[ -z $WORMHOLE_ADDRESS ]] && { echo "Missing WORMHOLE_ADDRESS"; exit 1; }
[[ -z $CHAIN_ID ]] && { echo "Missing CHAIN_ID"; exit 1; }
[[ -z $INIT_EVM_CHAIN_ID ]] && { echo "Missing INIT_EVM_CHAIN_ID"; exit 1; }

# we only deploy this contract in Ethereum (chain id 2)
if [ $CHAIN_ID -eq 2 ]; then
  forge script ./forge-scripts/DeployWormholeDelegatedGuardians.s.sol:DeployWormholeDelegatedGuardians \
      --sig "run(address)" $WORMHOLE_ADDRESS \
      --rpc-url $RPC_URL \
      --private-key $MNEMONIC \
      --broadcast ${FORGE_ARGS}

  returnInfo=$(cat ./broadcast/DeployWormholeDelegatedGuardians.s.sol/$INIT_EVM_CHAIN_ID/run-latest.json)
  DELEGATED_GUARDIANS_ADDRESS=$(jq -r '.returns.deployedDelegatedGuardians.value' <<< "$returnInfo")

  echo "-- Delegated Guardians Addresses --------------------------------------------------"
  echo "| Delegated Guardians address | $DELEGATED_GUARDIANS_ADDRESS |"
  echo "-------------------------------------------------------------------------------"

else 
  echo "Skipping Delegated Guardians deployment for chain id: $CHAIN_ID"
fi
