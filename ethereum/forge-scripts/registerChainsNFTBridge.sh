# Source the prompt_functions.sh file
source "./forge-scripts/prompt_functions.sh"

# Obtain environment variables
prompt_for_variable "RPC_URL" "Please provide the RPC URL"
prompt_for_variable "PRIVATE_KEY" "Please provide your private key"
prompt_for_variable "NFT_BRIDGE_ADDRESS" "Please provide the NFT Bridge address"
prompt_for_variable "NFT_BRIDGE_REGISTRATION_VAAS" "Please provide the NFT Bridge registration vaas (seperated by ',' - e.g. '0x1234,0x5678')"

forge script ./forge-scripts/RegisterChainsNFTBridge.s.sol:RegisterChainsNFTBridge --sig "run(address,bytes[])" $NFT_BRIDGE_ADDRESS $NFT_BRIDGE_REGISTRATION_VAAS --rpc-url $RPC_URL --private-key $PRIVATE_KEY --broadcast
