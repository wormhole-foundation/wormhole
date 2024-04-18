#!/bin/bash

# This script registers all the token bridges listed in the deployment file on the specified chain.

# Note that this script skips registering the specified chain on itself, but it does expect to
# find a VAA for that chain in the CSV file (as a sanity check). Please be sure to generate
# the registation VAA for this chain and add it to the file before running this script.

# MNEMONIC=<redacted> ./sh/registerAllChainsOnTokenBridge.sh testnet blast

if [ $# != 2 ]; then
	echo "Usage: $0 testnet blast" >&2
	exit 1
fi

[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }

network=$1
chain=$2

# Figure out which file of VAAs to use.
input_file=""
case "$network" in
    mainnet)
        input_file="../deployments/mainnet/tokenBridgeVAAs.csv"
    ;;
    testnet)
        input_file="../deployments/testnet/tokenBridgeVAAs.csv"
		;;
		*) echo "unknown network $network, must be testnet or mainnet" >&2
		exit 1
    ;;
esac

# Use the worm cli to get the chain parameters.
if ! command -v worm &> /dev/null
then
	echo "worm binary could not be found. See installation instructions in clients/js/README.md"
	exit 1
fi

chain_id=$(worm info chain-id "$chain")
if [ $? != 0 ]; then
	echo -e "\nERROR: failed to look up the chain id for ${chain}, please make sure the worm binary is current!" >&2
	exit 1
fi

rpc_url=$(worm info rpc "$network" "$chain")
if [ $? != 0 ]; then
	echo -e "\nERROR: failed to look up the RPC for ${chain}, please make sure the worm binary is current!" >&2
	exit 1
fi

token_bridge_address=$(worm info contract "$network" "$chain" TokenBridge)
if [ $? != 0 ]; then
	echo -e "\nERROR: failed to look up the token bridge address for ${chain}, please make sure the worm binary is current!" >&2
	exit 1
fi

# Build one long string of all the vaas in the input file.
vaas=""
found_us=0
count=0
while IFS= read -r line
do
	# Skip comment lines.
	echo $line | grep "^#" > /dev/null
	if [ $? == 0 ]; then
		continue
	fi

	tag=`echo $line | cut -d, -f1`
	vaa=`echo $line | cut -d, -f2`

	# Skip this chain. (We don't want to register this chain on itself.)
	echo $tag | grep "(${chain_id})" > /dev/null
	if [ $? == 0 ]; then
		found_us=1
		continue
	fi

	# The VAAs should be comma separated.
	if ! [ -z "${vaas}" ]; then
		vaas="${vaas},"
	fi

	vaas="${vaas}0x${vaa}"
	count=$(($count+1))  
done < "$input_file"

if [ $found_us == 0 ]; then
	echo "ERROR: failed to find chain id ${chain_id} in ${input_file}, something is not right!" >&2
	exit 1
fi

# Make it look like an array.
vaas="[${vaas}]"
echo $vaas

echo "Submitting ${count} VAAs to ${network} ${chain} token bridge at address ${token_bridge_address} and rpc ${rpc_url}"
forge script ./forge-scripts/RegisterChainsTokenBridge.s.sol:RegisterChainsTokenBridge \
	--sig "run(address,bytes[])" $token_bridge_address $vaas \
	--rpc-url $rpc_url \
	--private-key $MNEMONIC \
	--broadcast
