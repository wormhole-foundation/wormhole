#!/usr/bin/env fish

# To link Proxy and Implementation, go to the proxyContractChecker of the chain's etherscan

# Equivalent to `set -x` in bash, this prints out commands with variables substituted before executing them
# set fish_trace true

# TODO: add option to specify one or more chain ids and avoid verifying already verified contracts
set options (string join '' (fish_opt --short t --long scan-tokens --required-val) '!jq . "$_flag_value" > /dev/null')
argparse $options -- $argv

if test -z $_flag_scan_tokens
    echo "--scan-tokens option is missing or invalid. Please specify a json file containing the token APIs for each block explorer."
    echo 'JSON format: [{"chainId": <chain id>, "etherscan": <token>}, ...]'
    exit 1
end
set scan_tokens_file $_flag_scan_tokens

set chains_file "ts-scripts/relayer/config/$ENV/chains.json"
set contracts_file "ts-scripts/relayer/config/$ENV/contracts.json"
if not test -e $contracts_file
    echo "$contracts_file does not exist. Delivery provider addresses are read from this file."
    exit 1
end

set chain_ids (string split \n --no-empty -- (jq '.operatingChains[]' $chains_file))

for chain in $chain_ids
    # Klaytn, Karura and Acala don't have a verification API yet
    if test 11 -le $chain && test $chain -le 13
        continue
    end

    # We need addresses to be unquoted when passed to `cast` and `forge verify-contract`
    set implementation_address (jq --raw-output ".deliveryProviderImplementations[] | select(.chainId == $chain) | .address" $contracts_file)
    set setup_address (jq --raw-output ".deliveryProviderSetups[] | select(.chainId == $chain) | .address" $contracts_file)
    set proxy_address (jq --raw-output ".deliveryProviders[] | select(.chainId == $chain) | .address" $contracts_file)

    # These two are documented in `forge verify-contract` as accepted environment variables.
    # We need the token to be unquoted when passed to `forge verify-contract`
    set --export ETHERSCAN_API_KEY (jq --raw-output ".[] | select(.chainId == $chain) | .etherscan" $scan_tokens_file)

    # Some explorers like mantle.explorer.xyz don't have an API token
    if test -z $ETHERSCAN_API_KEY
        echo "Warning: No scan token found for chain $chain."
    end

    set --export CHAIN (jq ".chains[] | select(.chainId == $chain) | .evmNetworkId" $chains_file)

    # We're using the production profile for delivery providers on mainnet and testnet
    set --export FOUNDRY_PROFILE production
    set proxy_constructor_args (cast abi-encode "constructor(address,bytes)" $setup_address (cast calldata "setup(address,uint16)" $implementation_address $chain))

    # Celo has a verification API but it currently doesn't work with `forge verify-contract`
    # We print the compiler input to a file instead for manual verification
    if test $chain -eq 14
        forge verify-contract --watch --show-standard-json-input \
            $implementation_address contracts/relayer/deliveryProvider/DeliveryProviderImplementation.sol:DeliveryProviderImplementation  > DeliveryProviderImplementation.compiler-input.json
        forge verify-contract --watch --show-standard-json-input \
            $setup_address contracts/relayer/deliveryProvider/DeliveryProviderSetup.sol:DeliveryProviderSetup > DeliveryProviderSetup.compiler-input.json
        forge verify-contract --watch --show-standard-json-input --constructor-args $proxy_constructor_args \
            $proxy_address contracts/relayer/deliveryProvider/DeliveryProviderProxy.sol:DeliveryProviderProxy > DeliveryProviderProxy.compiler-input.json

        echo "Please manually submit the compiler input files at celoscan.io"
        echo "- $implementation_address: DeliveryProviderImplementation.compiler-input.json"
        echo "- $setup_address: DeliveryProviderSetup.compiler-input.json"
        echo "- $proxy_address: DeliveryProviderProxy.compiler-input.json"
    else if test $chain -eq 35
        set mantle_explorer_url "https://explorer.mantle.xyz/api?module=contract&action=verify"

        forge verify-contract --verifier blockscout --verifier-url $mantle_explorer_url --watch \
            $implementation_address contracts/relayer/deliveryProvider/DeliveryProviderImplementation.sol:DeliveryProviderImplementation
        forge verify-contract --verifier blockscout --verifier-url $mantle_explorer_url --watch \
            $setup_address contracts/relayer/deliveryProvider/DeliveryProviderSetup.sol:DeliveryProviderSetup
        forge verify-contract --verifier blockscout --verifier-url $mantle_explorer_url --watch \
            $proxy_address contracts/relayer/deliveryProvider/DeliveryProviderProxy.sol:DeliveryProviderProxy
    else if test $chain -eq 37
        set xlayer_explorer_url "https://www.oklink.com/api/v5/explorer/contract/verify-source-code-plugin/XLAYER"

        forge verify-contract --verifier oklink --verifier-url $xlayer_explorer_url --watch \
            $implementation_address contracts/relayer/deliveryProvider/DeliveryProviderImplementation.sol:DeliveryProviderImplementation
        forge verify-contract --verifier oklink --verifier-url $xlayer_explorer_url --watch \
            $setup_address contracts/relayer/deliveryProvider/DeliveryProviderSetup.sol:DeliveryProviderSetup
        forge verify-contract --verifier oklink --verifier-url $xlayer_explorer_url --watch \
            $proxy_address contracts/relayer/deliveryProvider/DeliveryProviderProxy.sol:DeliveryProviderProxy
    else
        forge verify-contract --watch \
            $implementation_address contracts/relayer/deliveryProvider/DeliveryProviderImplementation.sol:DeliveryProviderImplementation
        forge verify-contract --watch \
            $setup_address contracts/relayer/deliveryProvider/DeliveryProviderSetup.sol:DeliveryProviderSetup
        forge verify-contract --watch --constructor-args $proxy_constructor_args \
            $proxy_address contracts/relayer/deliveryProvider/DeliveryProviderProxy.sol:DeliveryProviderProxy
    end
end