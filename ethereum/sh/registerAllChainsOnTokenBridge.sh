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

[[ -z ${MNEMONIC:-} ]] && { echo "Missing MNEMONIC"; exit 1; }

network=$1
chain=$2
token_bridge_address=$3

# Figure out which VAA file to use.
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

# Do not continue with an empty registration batch when the CSV is unavailable.
[[ -r "$input_file" ]] || { echo "Unable to read $input_file" >&2; exit 1; }

# Configuration must be supplied by the operator environment.
[[ -z ${RPC_URL:-} ]] && { echo "Missing RPC_URL"; exit 1; }

# Match the target by Wormhole chain ID, not an ambiguous name substring.
worm_chain_id=$(worm info chain-id "$chain") || exit 1

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
	if [[ "$tag" == *"($worm_chain_id)"* ]]; then
		found_us=$((found_us + 1))
		continue
	fi

	# The VAAs should be comma separated.
	if ! [ -z "${vaas}" ]; then
		vaas="${vaas},"
	fi

	vaas="${vaas}0x${vaa}"
	count=$(($count+1))
done < "$input_file"

# The target row is required as a sanity check and must be unambiguous.
[ "$found_us" -eq 1 ] || { echo "Expected exactly one VAA for $chain" >&2; exit 1; }

# Make it look like an array.
vaas="[${vaas}]"
echo $vaas

echo "Submitting ${count} VAAs to ${network} ${chain} token bridge at address ${token_bridge_address} and rpc ${RPC_URL}"
# Keep the repository-derived VAA list inside one Forge argument.
forge script ./forge-scripts/RegisterChainsTokenBridge.s.sol:RegisterChainsTokenBridge \
	--sig "run(address,bytes[])" "$token_bridge_address" "$vaas" \
	--rpc-url "$RPC_URL" \
	--private-key "$MNEMONIC" \
	--broadcast ${FORGE_ARGS}
