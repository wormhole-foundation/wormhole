#!/usr/bin/env bash
set -euo pipefail

RPC="${RPC_URL:-ws://eth-devnet:8545}"

# mainnet values
# export CORE_CONTRACT="0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
# export TOKEN_BRIDGE_CONTRACT="0x3ee18B2214AFF97000D974cf647E7C347E8fa585"

# TODO these could be CLI params from the sh/devnet script
CORE_BRIDGE_CONTRACT=0xC89Ce4735882C9F0f0FE26686c53074E09B0D550
TOKEN_BRIDGE_CONTRACT=0x0290FB167208Af455bB137780163b7B7a9a10C16

MNEMONIC=0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d

ERC20_ADDR="0x47bdB2D7d6528C760b6f228b3B8F9F650169a10f" # Test token A

VALUE="1000" # Wei value sent as msg.value
TRANSFER_AMOUNT="10"

# Account reported by anvil when run using $MNEMONIC.
# The account at index 0 is used by other tests in the test suite, so
# account[1] is used here to help encapsulate the tests
ANVIL_USER="0xFFcf8FDEE72ac11b5c542428B35EEF5769C409f0" 
ETH_WHALE="${ANVIL_USER}"
FROM="${ETH_WHALE}"
# Anvil user1 normalized to Wormhole size. (The value itself it unchecked but must have this format.)
RECIPIENT="0x000000000000000000000000FFcf8FDEE72ac11b5c542428B35EEF5769C409f0" 
NONCE="234" # arbitrary

# Build the payload for token transfers. Declared on multiple lines to
# be more legible. Data pulled from an arbitrary LogMessagePublished event
# on etherscan. Metadata and fees commented out, leaving only the payload
PAYLOAD="0x"
declare -a SLOTS=(
   # "0000000000000000000000000000000000000000000000000000000000055baf"
   # "0000000000000000000000000000000000000000000000000000000000000000"
   # "0000000000000000000000000000000000000000000000000000000000000080"
   # "0000000000000000000000000000000000000000000000000000000000000001"
   # "00000000000000000000000000000000000000000000000000000000000000ae"
   "030000000000000000000000000000000000000000000000000000000005f5e1"
   "000000000000000000000000002260fac5e5542a773aa44fbcfedf7c193bc2c5"
   "9900020000000000000000000000000000000000000000000000000000000000"
   "000816001000000000000000000000000044eca3f6295d6d559ca1d99a5ef5a8"
   "f72b4160f10001010200c91f01004554480044eca3f6295d6d559ca1d99a5ef5"
   "a8f72b4160f10000000000000000000000000000000000000000000000000000"
)
for i in "${SLOTS[@]}"
do
   PAYLOAD="$PAYLOAD$i"
done

echo "DEBUG:"
echo "- RPC=${RPC}"
echo "- CORE_BRIDGE_CONTRACT=${CORE_BRIDGE_CONTRACT}"
echo "- TOKEN_BRIDGE_CONTRACT=${TOKEN_BRIDGE_CONTRACT}"
echo "- MNEMONIC=${MNEMONIC}"
echo "- FROM=${FROM}"
echo "- VALUE=${VALUE}" 
echo "- RECIPIENT=${RECIPIENT}" 
echo 

# Fund the token bridge from User0
echo "Start impersonating User0"
cast rpc \
   anvil_impersonateAccount "${ANVIL_USER}" \
   --rpc-url "${RPC}"
echo "Funding token bridge using user0's balance"
cast send --unlocked \
   --rpc-url "${RPC}" \
   --from $ANVIL_USER \
   --value 100000000000000 \
   ${TOKEN_BRIDGE_CONTRACT}
echo ""
echo "End impersonating User0"
cast rpc \
   anvil_stopImpersonatingAccount "${ANVIL_USER}" \
   --rpc-url "${RPC}"

BALANCE_CORE=$(cast balance --rpc-url "${RPC}" $CORE_BRIDGE_CONTRACT)
BALANCE_TOKEN=$(cast balance --rpc-url "${RPC}" $TOKEN_BRIDGE_CONTRACT)
BALANCE_USER=$(cast balance --rpc-url "${RPC}" $ANVIL_USER)
echo "BALANCES:"
echo "- CORE_BRIDGE_CONTRACT=${BALANCE_CORE}"
echo "- TOKEN_BRIDGE_CONTRACT=${BALANCE_TOKEN}"
echo "- ANVIL_USER=${BALANCE_USER}"
echo 

# === Malicious call to transferTokensWithPayload()
# This is the exploit scenario: the token bridge has called publishMessage() without a ERC20 Transfer or Deposit
# being present in the same receipt.
# This is done by impersonating the token bridge contract and sending a message directly to the core bridge.
# Ensure that anvil is using `--auto-impersonate` or else that account impersonation is enabled in your local environment.
# --private-key "$MNEMONIC" \
# --max-fee 500000 \
echo "Start impersonate token bridge" 
cast rpc \
   --rpc-url "${RPC}" \
   anvil_impersonateAccount "${TOKEN_BRIDGE_CONTRACT}"
echo "Calling publishMessage as ${TOKEN_BRIDGE_CONTRACT}" 
cast send --unlocked \
   --rpc-url "${RPC}" \
   --json \
   --gas-limit 10000000 \
   --priority-gas-price 1 \
   --from "${TOKEN_BRIDGE_CONTRACT}" \
   --value "0" \
   "${CORE_BRIDGE_CONTRACT}" \
   "publishMessage(uint32,bytes,uint8)" \
   0 "${PAYLOAD}" 1
echo ""
cast rpc \
   --rpc-url "${RPC}" \
   anvil_stopImpersonatingAccount "${TOKEN_BRIDGE_CONTRACT}"
echo "End impersonate token bridge" 

# TODO add the 'multicall' scenario encoded in the forge script

echo "Done Transfer Verifier integration test."
echo "Exiting."
