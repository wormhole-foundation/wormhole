#!/usr/bin/env fish

# TODO:
# This script is very similar to verifyDeliveryProxy.sh, but it only verifies the implementation contract
# instead of all the contracts (implementation, setup and proxy). When performing an upgrade you'll only
# need to verify the new implementation since the setup and proxy were already verified during the deployment
# We should refactor this script to avoid code duplication.

# To link Proxy and Implementation, go to the proxyContractChecker of the chain's etherscan

# Equivalent to `set -x` in bash, this prints out commands with variables substituted before executing them
# set fish_trace true

# TODO: add option to specify one or more chain ids and avoid verifying already verified contracts
set options (string join '' (fish_opt --short t --long scan-tokens --required-val) '!jq . "$_flag_value" > /dev/null')
argparse $options -- $argv

if test -z $_flag_scan_tokens
    echo "--scan-tokens option is missing or invalid. Please specify a json file containing the token APIs for each block explorer."
    echo 'JSON format: [{"chainId": <chain id>, "token": <token>}, ...]'
    exit 1
end
set scan_tokens_file $_flag_scan_tokens

set chains_file "ts-scripts/relayer/config/$ENV/chains.json"
set last_run_file "ts-scripts/relayer/output/$ENV/upgradeDeliveryProvider/lastrun.json"
if not test -e $last_run_file
    echo "$last_run_file does not exist. Delivery provider addresses are read from this file."
    exit 1
end

set chain_ids (string split \n --no-empty -- (jq '.chains[] | .chainId' $chains_file))

for chain in $chain_ids
    # Klaytn, Karura and Acala don't have a verification API yet
    if test 11 -le $chain && test $chain -le 13
        continue
    end

    # We need addresses to be unquoted when passed to `cast` and `forge verify-contract`
    set implementation_address (jq --raw-output ".deliveryProviderImplementations[] | select(.chainId == $chain) | .address" $last_run_file)

    # We need the token to be unquoted when passed to `forge verify-contract`
    set scan_token (jq --raw-output ".[] | select(.chainId == $chain) | .token" $scan_tokens_file)

    # if we dont have a scan token echo a warning and continue
    if test -z $scan_token
        echo "Error: No scan token found for chain $chain. Chain will not be verified."
        continue
    end

    echo "Verifying delivery provider contract ($implementation_address) on chain $chain"

    set evm_chain_id (jq ".chains[] | select(.chainId == $chain) | .evmNetworkId" $chains_file)

    # We're using the production profile for delivery providers on mainnet and testnet
    set --export FOUNDRY_PROFILE production

    # Celo has a verification API but it currently doesn't work with `forge verify-contract`
    # We print the compiler input to a file instead for manual verification
    if test $chain -eq 14
        forge verify-contract $implementation_address contracts/relayer/deliveryProvider/DeliveryProviderImplementation.sol:DeliveryProviderImplementation --chain-id $evm_chain_id --watch --etherscan-api-key $scan_token --show-standard-json-input > DeliveryProviderImplementation.compiler-input.json

        echo "Please manually submit the compiler input files at celoscan.io"
        echo "- $implementation_address: DeliveryProviderImplementation.compiler-input.json"
    else
        forge verify-contract $implementation_address contracts/relayer/deliveryProvider/DeliveryProviderImplementation.sol:DeliveryProviderImplementation --chain-id $evm_chain_id --watch --etherscan-api-key $scan_token
    end
end