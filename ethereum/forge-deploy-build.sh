#!/bin/bash

# The relayer won't build because it requires a newer compiler.
rm -rf contracts/interfaces/relayer
rm -rf contracts/mock/relayer
rm -rf contracts/relayer
rm -rf forge-test/relayer

FOUNDRY_PROFILE=deploy forge build

# To deploy the contracts, set up the .env file and do:
# MNEMONIC=<redacted> ./forge-scripts/deployCoreBridge.sh
# MNEMONIC=<redacted> WORMHOLE_ADDRESS=<from_the_previous_command> ./forge-scripts/deployTokenBridge.sh