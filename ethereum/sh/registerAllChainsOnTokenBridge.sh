#!/bin/bash

# This script registers all the token bridges listed in the deployment file on the specified chain.

# Note that this script skips registering the specified chain on itself, but it does expect to
# find a VAA for that chain in the CSV file (as a sanity check). Please be sure to generate
# the registation VAA for this chain and add it to the file before running this script.

# MNEMONIC=<redacted> ./sh/registerAllChainsOnTokenBridge.sh <network> <chainName> <tokenBridgeAddress>

if [ $# != 3 ]; then
	echo "Usage: $0 <network> <chainName> <tokenBridgeAddress>" >&2
	exit 1
fi

[[ -z $MNEMONIC ]] && { echo "Missing MNEMONIC"; exit 1; }

network=$1
chain=$2
token_bridge_address=$3

# Figure out which env file and VAA files to use.
env_file=""
input_file=""
case "$network" in
    mainnet)
				env_file="env/.env.${chain}"
        input_file="../deployments/mainnet/tokenBridgeVAAs.csv"
    ;;
    testnet)
				env_file="env/.env.${chain}.testnet"
        input_file="../deployments/testnet/tokenBridgeVAAs.csv"
		;;
		*) echo "unknown network $network, must be testnet or mainnet" >&2
		exit 1
    ;;
esac

# Source in the env file to get the RPC and forge arguments.
. ${env_file}

[[ -z $RPC_URL ]] && { echo "Missing RPC_URL"; exit 1; }

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
	echo $tag | grep -i ${chain} > /dev/null
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
	echo "ERROR: failed to find chain ${chain} in ${input_file}, something is not right!" >&2
	exit 1
fi

# Make it look like an array.
vaas="[${vaas}]"
echo $vaas

echo "Submitting ${count} VAAs to ${network} ${chain} token bridge at address ${token_bridge_address} and rpc ${RPC_URL}"
forge script ./forge-scripts/RegisterChainsTokenBridge.s.sol:RegisterChainsTokenBridge \
	--sig "run(address,bytes[])" $token_bridge_address $vaas \
	--rpc-url $RPC_URL \
	--private-key $MNEMONIC \
	--broadcast ${FORGE_ARGS}
