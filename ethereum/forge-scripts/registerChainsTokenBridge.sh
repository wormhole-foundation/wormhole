# Source the prompt_functions.sh file
source "./forge-scripts/prompt_functions.sh"

# Obtain environment variables
prompt_for_variable "RPC_URL" "Please provide the RPC URL"
prompt_for_variable "PRIVATE_KEY" "Please provide your private key"
prompt_for_variable "TOKEN_BRIDGE_ADDRESS" "Please provide the Token Bridge address"
prompt_for_variable "TOKEN_BRIDGE_REGISTRATION_VAAS" "Please provide the Token Bridge registration vaas (seperated by ',' - e.g. '0x1234,0x5678')"

forge script ./forge-scripts/RegisterChainsTokenBridge.s.sol:RegisterChainsTokenBridge --sig "run(address,bytes[])" $TOKEN_BRIDGE_ADDRESS $TOKEN_BRIDGE_REGISTRATION_VAAS --rpc-url $RPC_URL --private-key $PRIVATE_KEY --broadcast
