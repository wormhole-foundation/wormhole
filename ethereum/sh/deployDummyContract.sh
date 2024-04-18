#!/bin/bash

[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }
[[ -z $RPC_URL ]] && { echo "Missing RPC_URL"; exit 1; }
[[ -z $NUM_RUNS ]] && { echo "Missing NUM_RUNS"; exit 1; }

# Deploy dummy contract(s) to match devnet addresses in Anvil to what they originally were in Ganache 
# (the addresses depend on the number of contracts that have been previously deployed, and the wallet address, I believe!)

forge script ./forge-scripts/DeployDummyContract.s.sol:DeployDummyContract \
	--sig "run(uint256)" $NUM_RUNS \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast
