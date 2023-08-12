#!/usr/bin/env fish

# note: the first 5 testnets (avalanche, celo, bsc, mumbai, moonbeam) were deployed with evm_version London

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
set contracts_file "ts-scripts/relayer/config/$ENV/contracts.json"

set chain_ids (string split \n --no-empty -- (jq '.chains[] | .chainId' $chains_file))

for chain in $chain_ids
    # Klaytn, Karura and Acala don't have a verification API yet
    if test 11 -le $chain && test $chain -le 13
        continue
    end

    # We need addresses to be unquoted when passed to `cast` and `forge verify-contract`
    set create2_factory_address (jq --raw-output ".create2Factories[] | select(.chainId == $chain) | .address" $contracts_file)

    # These two are documented in `forge verify-contract` as accepted environment variables.
    # We need the token to be unquoted when passed to `forge verify-contract`
    set --export ETHERSCAN_API_KEY (jq --raw-output ".[] | select(.chainId == $chain) | .token" $scan_tokens_file)
    set --export CHAIN (jq ".chains[] | select(.chainId == $chain) | .evmNetworkId" $chains_file)

    # We're using the production profile for delivery providers on mainnet and testnet
    set --export FOUNDRY_PROFILE production

    # We need to compute the address of the Init contract since it is used as a constructor argument for the creation of the proxy.
    # `Init` is created through CREATE which uses the address + nonce derivation for its address.
    # Contract accounts start with their nonce at 1. See https://eips.ethereum.org/EIPS/eip-161#specification.
    set init_contract_address (cast compute-address $create2_factory_address --nonce 1)
    # `cast compute-address` prints out "Computed Address: 0x..." so we have to split the string here.
    set init_contract_address (string split ' ' $init_contract_address)[-1]

    # Celo has a verification API but it currently doesn't work with `forge verify-contract`
    # We print the compiler input to a file instead for manual verification
    if test $chain -eq 14
        forge verify-contract $create2_factory_address contracts/relayer/create2Factory/Create2Factory.sol:Create2Factory --watch --show-standard-json-input > Create2Factory.compiler-input.json
        forge verify-contract $init_contract_address contracts/relayer/create2Factory/Create2Factory.sol:Init --watch --show-standard-json-input > Init.compiler-input.json

        echo "Please manually submit the compiler input files at celoscan.io"
        echo "- $create2_factory_address: Create2Factory.compiler-input.json"
        echo "- $init_contract_address: Init.compiler-input.json"
    else
        forge verify-contract $create2_factory_address contracts/relayer/create2Factory/Create2Factory.sol:Create2Factory --watch
        forge verify-contract $init_contract_address contracts/relayer/create2Factory/Create2Factory.sol:Init --watch
    end
end
