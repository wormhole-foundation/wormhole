#!/bin/bash

# Source the prompt_functions.sh file
source "./forge-scripts/prompt_functions.sh"

# Obtain environment variables
prompt_for_variable "RPC_URL" "Please provide the RPC URL"
prompt_for_variable "PRIVATE_KEY" "Please provide your private key"
prompt_for_variable "NONCE" "Please provide the nonce amount to increment your account by"


# Deploy dummy contract(s) to match devnet addresses in Anvil to what they originally were in Ganache 
# (the addresses depend on the number of contracts that have been previously deployed, and the wallet address, I believe!)

forge script ./forge-scripts/SetNonce.s.sol:SetNonce --sig "incrementNonce(uint64)" $NONCE --rpc-url "$RPC_URL" --private-key "$PRIVATE_KEY" --broadcast
