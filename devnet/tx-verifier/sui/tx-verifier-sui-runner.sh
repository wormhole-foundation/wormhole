#!/bin/sh

# Add the unsafe functions to the transfer_tokens module. Using sed, the `transfer_tokens_unsafe.move` content
# can be added right after the module definition. Module imports seem to be hoisted, so there's no need to add
# the unsafe code after module imports.
sed -i '/module token_bridge::transfer_tokens/r /tmp/transfer_tokens_unsafe.move' /tmp/token_bridge/sources/transfer_tokens.move

# Configurations, such as the RPC endpoint and output files for various pieces of information collected for the
# tests to be performed.
RPC=http://sui:9000
SETUP_DEVNET_OUTPUT_PATH=/tmp/setup-devnet-output.txt
EXAMPLE_COIN_TX_OUTPUT_PATH=/tmp/setup-example-coin-tx-output.txt

# Deploy Sui packages to devnet node. Even though the Sui node already has a wormhole deployment, in order to
# test Sui transfer verification it is necessary to add code that will allow forcefully triggering the invariant
# being monitored.
echo "[*] setting up a devnet deployment"
cd /tmp && worm sui setup-devnet --rpc=$RPC > $SETUP_DEVNET_OUTPUT_PATH && cd ..

# Get the package IDs that are produced by the `worm sui setup-devnet` command, and convert the IDs to an array.
package_object_ids=`grep -A5 "Summary:" /tmp/setup-devnet-output.txt | grep -oE "0x[a-f0-9]{64}"`

package_ids=()
for i in $package_object_ids; do
    package_ids+=($i)
done

# These are the package IDs and state object IDs that are relevant for interacting with the token bridge.
core_bridge_package_id=${package_ids[0]}
core_bridge_state=${package_ids[1]}
token_bridge_package_id=${package_ids[2]}
token_bridge_state=${package_ids[3]}
token_bridge_emitter_cap=${package_ids[4]}

echo " - core_bridge_package_id    = $core_bridge_package_id"
echo " - core_bridge_state         = $core_bridge_state"
echo " - token_bridge_package_id   = $token_bridge_package_id"
echo " - token_bridge_state        = $token_bridge_state"
echo " - token_bridge_emitter_cap  = $token_bridge_emitter_cap"

# This is a helper function to parse out different pieces of information from the transaction block during which
# the example coins were deployed.
get_coin_information() {
    local filename=$1
    local coin_name=$2
    local coin_info_type=$3
    local treasury_cap_object_id=`
        jq -r --arg COIN $coin_name --arg INFO_TYPE $coin_info_type '.objectChanges[] |
            select(
                .objectType != null and 
                (.objectType | contains($INFO_TYPE)) and 
                (.objectType | contains($COIN))
            ) | 
            .objectId' $filename`

    echo $treasury_cap_object_id
}

# Get the transaction digest of the example coin deployment.
coin_deployment_digest=`cat $SETUP_DEVNET_OUTPUT_PATH | grep "Deploying example coins" -A1 | grep "digest" | cut -d' ' -f3`
# Retrieve the transaction block.
sui client tx-block $coin_deployment_digest --json > $EXAMPLE_COIN_TX_OUTPUT_PATH
# Read the package ID of the coin package that holds the coins.
coin_package_id=`jq -r '.objectChanges[] | select(.type != null and .type == "published") | .packageId' $EXAMPLE_COIN_TX_OUTPUT_PATH`
# Get the TreasuryCap and CoinMetadata object IDs
treasury_cap_10=`get_coin_information $EXAMPLE_COIN_TX_OUTPUT_PATH COIN_10 TreasuryCap`
coin_metadata_10=`get_coin_information $EXAMPLE_COIN_TX_OUTPUT_PATH COIN_10 CoinMetadata`

# A helper function to attest a native token on the token bridge. This effectively registers a coin on Sui as a
# native asset on the token bridge.
attest_token() {
    local coin_type=$1
    local coin_metadata=$2
    local sequence=$3
    sui client ptb \
        --move-call $token_bridge_package_id::attest_token::attest_token "<$coin_type>" @$token_bridge_state @$coin_metadata $sequence \
        --assign message_ticket \
        --split-coins gas [0] \
        --assign empty_coin \
        --move-call $core_bridge_package_id::publish_message::publish_message @$core_bridge_state empty_coin message_ticket @0x06 \
        --gas-budget 10000000
}

echo "[*] adding the COIN_10 coin to the token bridge."
res=`attest_token $coin_package_id::coin_10::COIN_10 $coin_metadata_10 1u32`

# mint_and_transfer_token is a helper function that mints an `amount` of tokens to the caller, and then transfers
# those tokens out via the token bridge. 
# This is used for legitimate token bridge transfers.
mint_and_transfer_token() {
    local treasury_cap=$1
    local coin_type=$2
    local amount=$3
    sui client ptb \
        --move-call sui::tx_context::sender \
        --assign sender \
        --move-call sui::coin::mint "<$coin_type>" @$treasury_cap $amount \
        --assign minted_coin \
        --move-call $token_bridge_package_id::state::verified_asset "<$coin_type>" @$token_bridge_state \
        --assign verified_asset \
        --make-move-vec "<u8>" [] \
        --assign recipient \
        --split-coins gas [0] \
        --assign empty_gas_coin \
        --move-call $token_bridge_package_id::transfer_tokens::prepare_transfer "<$coin_type>" verified_asset minted_coin 1u16 recipient 0u64 1u32 \
        --assign prepare_transfer_result \
        --move-call $token_bridge_package_id::transfer_tokens::transfer_tokens "<$coin_type>" @$token_bridge_state prepare_transfer_result.0 \
        --assign message_ticket \
        --move-call $core_bridge_package_id::publish_message::publish_message @$core_bridge_state empty_gas_coin message_ticket @0x06 \
        --transfer-objects sender [prepare_transfer_result.1] \
        --gas-budget 10000000
}

echo "[*] mint_and_transfer 100 COIN_10 tokens"
res=`mint_and_transfer_token $treasury_cap_10 $coin_package_id::coin_10::COIN_10 100_0000000000`

# mint_and_transfer_unsafe_imbalanced is a helper function that uses the unsafe `prepare_transfer_unsafe` function
# to send X amount of tokens to the token bridge, but request Y amount of tokens out of it.
# This is used to trigger the invariant where less tokens are deposited into the bridge than requested out.
mint_and_transfer_unsafe_imbalanced() {
    local treasury_cap=$1
    local coin_type=$2
    local amount=$3
    local amount_to_bridge=$4
    sui client ptb \
        --move-call sui::tx_context::sender \
        --assign sender \
        --move-call sui::coin::mint "<$coin_type>" @$treasury_cap $amount \
        --assign minted_coin \
        --move-call $token_bridge_package_id::state::verified_asset "<$coin_type>" @$token_bridge_state \
        --assign verified_asset \
        --make-move-vec "<u8>" [] \
        --assign recipient \
        --split-coins gas [0] \
        --assign empty_gas_coin \
        --move-call $token_bridge_package_id::transfer_tokens::prepare_transfer_unsafe "<$coin_type>" verified_asset minted_coin $amount_to_bridge 1u16 recipient 0u64 1u32 \
        --assign prepare_transfer_result \
        --move-call $token_bridge_package_id::transfer_tokens::transfer_tokens "<$coin_type>" @$token_bridge_state prepare_transfer_result.0 \
        --assign message_ticket \
        --move-call $core_bridge_package_id::publish_message::publish_message @$core_bridge_state empty_gas_coin message_ticket @0x06 \
        --transfer-objects sender [prepare_transfer_result.1] \
        --gas-budget 10000000
}

# transfer_without_deposit_unsafe is a helper function that uses the unsafe `transfer_tokens_unsafe` function to
# transfer funds out of the bridge without actually making the deposit.
# This is used to trigger the invariant where no tokens are deposited into the bridge, but an amount is requested out.
transfer_without_deposit_unsafe() {
    local treasury_cap=$1
    local coin_type=$2
    local amount=0
    local amount_to_bridge=$3
    sui client ptb \
        --move-call sui::tx_context::sender \
        --assign sender \
        --move-call sui::coin::mint "<$coin_type>" @$treasury_cap $amount \
        --assign minted_coin \
        --move-call $token_bridge_package_id::state::verified_asset "<$coin_type>" @$token_bridge_state \
        --assign verified_asset \
        --make-move-vec "<u8>" [] \
        --assign recipient \
        --split-coins gas [0] \
        --assign empty_gas_coin \
        --move-call $token_bridge_package_id::transfer_tokens::prepare_transfer_unsafe "<$coin_type>" verified_asset minted_coin $amount_to_bridge 1u16 recipient 0u64 1u32 \
        --assign prepare_transfer_result \
        --move-call $token_bridge_package_id::transfer_tokens::transfer_tokens_unsafe "<$coin_type>" @$token_bridge_state prepare_transfer_result.0 \
        --assign message_ticket \
        --move-call $core_bridge_package_id::publish_message::publish_message @$core_bridge_state empty_gas_coin message_ticket.0 @0x06 \
        --transfer-objects sender [prepare_transfer_result.1] \
        --transfer-objects sender [message_ticket.1] \
        --gas-budget 10000000
}

echo "[*] running testcases pre-start" # these tests are aimed at the suiProcessInitialEvents flag
# testcase 1 - do a normal token bridge transfer
res=`mint_and_transfer_token $treasury_cap_10 $coin_package_id::coin_10::COIN_10 100_0000000000`
sleep 1

# testcase 2 - do a token bridge transfer where the deposited amount does not cover the full bridge amount
res=`mint_and_transfer_unsafe_imbalanced $treasury_cap_10 $coin_package_id::coin_10::COIN_10 100_0000000000 200_0000000000`
sleep 1

# testcase 3 - do a token bridge transfer where the token is not deposited at all
resp=`transfer_without_deposit_unsafe $treasury_cap_10 $coin_package_id::coin_10::COIN_10 200_0000000000`
sleep 1

echo "[*] starting the sui transfer verifier"
/guardiand transfer-verifier \
    sui \
    --suiRPC "${RPC}" \
    --suiCoreContract "${core_bridge_package_id}" \
    --suiTokenBridgeContract "${token_bridge_package_id}" \
    --suiTokenBridgeEmitter "${token_bridge_emitter_cap}" \
    --logLevel=debug \
    --suiProcessInitialEvents=true \
    2> /tmp/error.log &

echo "[*] running testcases post-start"

# testcase 1 - do a normal token bridge transfer
res=`mint_and_transfer_token $treasury_cap_10 $coin_package_id::coin_10::COIN_10 100_0000000000`
sleep 1

# testcase 2 - do a token bridge transfer where the deposited amount does not cover the full bridge amount
res=`mint_and_transfer_unsafe_imbalanced $treasury_cap_10 $coin_package_id::coin_10::COIN_10 100_0000000000 200_0000000000`
sleep 1

# testcase 3 - do a token bridge transfer where the token is not deposited at all
res=`transfer_without_deposit_unsafe $treasury_cap_10 $coin_package_id::coin_10::COIN_10 200_0000000000`
sleep 10

# There should be two of each test - one for pre-start and one for post-start.
echo "[*] verifying that tests succeeded"

cat /tmp/error.log

if [ $(cat /tmp/error.log | grep "bridge transfer requested for more tokens than were deposited" | wc -l) -ne 2 ]; then
    echo " [-] amount out > amount in test failed"
    exit 1
fi

if [ $(cat /tmp/error.log | grep "bridge transfer requested for tokens that were never deposited" | wc -l) -ne 2 ]; then
    echo " [-] amount in == 0 test failed"
    exit 1
fi

echo "[+] tests passed"
touch /tmp/success