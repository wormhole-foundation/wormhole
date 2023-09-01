#!/usr/bin/env fish

# To link Proxy and Implementation, go to the proxyContractChecker of the chain's etherscan

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
# TODO: add implementation addresses to `contracts.json` to allow using it instead of lastrun.json
set last_run_file "ts-scripts/relayer/output/$ENV/deployWormholeRelayer/lastrun.json"
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
    set implementation_address (jq --raw-output ".wormholeRelayerImplementations[] | select(.chainId == $chain) | .address" $last_run_file)
    set proxy_address (jq --raw-output ".wormholeRelayerProxies[] | select(.chainId == $chain) | .address" $last_run_file)
    set create2_factory_address (jq --raw-output ".create2Factories[] | select(.chainId == $chain) | .address" $contracts_file)
    # TODO: actually consult this from `worm` CLI
    # Perhaps the value present in the chains file can be used as a fallback when the current version of the `worm` program doesn't know about
    # a particular wormhole deployment
    set wormhole_address (jq --raw-output ".chains[] | select(.chainId == $chain) | .wormholeAddress" $chains_file)
    # This actually pads the address to 32 bytes with 12 zero bytes at the start
    # And we discard the "0x"
    set wormhole_address (cast to-uint256 $wormhole_address | sed 's/^0x//g' -)

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
    # This actually pads the address to 32 bytes with 12 zero bytes at the start
    # And we discard the "0x"
    set init_contract_address (cast to-uint256 $init_contract_address | sed 's/^0x//g' -)

    # Celo has a verification API but it currently doesn't work with `forge verify-contract`
    # We print the compiler input to a file instead for manual verification
    if test $chain -eq 14
        forge verify-contract $proxy_address contracts/relayer/create2Factory/Create2Factory.sol:SimpleProxy --watch --constructor-args $init_contract_address --show-standard-json-input > WormholeRelayerProxy.compiler-input.json
        forge verify-contract $implementation_address WormholeRelayer --watch --constructor-args $wormhole_address --show-standard-json-input > WormholeRelayerImplementation.compiler-input.json

        echo "Please manually submit the compiler input files at celoscan.io"
        echo "- $implementation_address: WormholeRelayerImplementation.compiler-input.json"
        echo "- $proxy_address: WormholeRelayerProxy.compiler-input.json"
    else
        forge verify-contract $proxy_address contracts/relayer/create2Factory/Create2Factory.sol:SimpleProxy --watch --constructor-args $init_contract_address
        forge verify-contract $implementation_address WormholeRelayer --watch --constructor-args $wormhole_address
    end
end

# TODO: print proxy contract URLs so it's easy to navigate to them and verify they're proxies
