#!/usr/bin/env bash
set -e
sleep 20

echo "DELEGATED PRESET"

num_guardians=$1
echo "num_guardians ${num_guardians}"

webHost=$2
echo "webHost ${webHost}"

namespace=$3
echo "namespace ${namespace}"

echo "chain id"
echo $CHAIN_ID

PRIV_KEY=0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d
RPC_URL=http://localhost:8545

# file & path to save governance VAA
# Use /tmp with a fixed name so the path works both locally and inside containers
tmpFile=/tmp/governance-vaa-$$.prototxt
trap 'rm -f -- "$tmpFile"' EXIT

sock=/tmp/admin.sock

guardianPublicWebBaseUrl="${webHost}:7071"
currentGuardianSetUrl="${guardianPublicWebBaseUrl}/v1/guardianset/current"

# fetch current guardian set info
guardianSet=$(curl ${currentGuardianSetUrl} | jq ".guardianSet")
currentIndex=$(echo ${guardianSet} | jq ".index")
address0=$(echo ${guardianSet} | jq ".addresses[0]")
address1=$(echo ${guardianSet} | jq ".addresses[1]")
address2=$(echo ${guardianSet} | jq ".addresses[2]")
currentNumGuardians=$(echo ${guardianSet} | jq ".addresses | length")
newNumGuardians=${num_guardians}
echo "guardianSet: ${guardianSet}"
echo "currentIndex: ${currentIndex}"
echo "currentNumGuardians: ${currentNumGuardians}"
echo "newNumGuardians: ${newNumGuardians}"
echo "address0: ${address0}"
echo "address1: ${address1}"
echo "address2: ${address2}"

# Fetch addresses from the eth-devnet (delegated guardians is only deployed in eth-devnet)
WORMHOLE_ADDRESS=$(kubectl exec -n "$namespace" eth-devnet-0 -c tests -- \
  jq -r '.returns.deployedAddress.value' /home/node/app/ethereum/broadcast/DeployCore.s.sol/1337/run-latest.json)

DELEGATED_GUARDIANS_ADDRESS=$(kubectl exec -n "$namespace" eth-devnet-0 -c tests -- \
  jq -r '.returns.deployedDelegatedGuardians.value' /home/node/app/ethereum/broadcast/DeployWormholeDelegatedGuardians.s.sol/1337/run-latest.json)

echo "WORMHOLE_ADDRESS: $WORMHOLE_ADDRESS"
echo "DELEGATED_GUARDIANS_ADDRESS: $DELEGATED_GUARDIANS_ADDRESS"

echo "creating guardian set update governance message template prototext"
# We're creating a preset where eth-devnet-2 is a delegated chain
# Guardian 0 is a canonical guardian
# Guardian 1 and 2 are delegated guardians
# Quorum is 2/3
kubectl exec -n ${namespace} guardian-0 -c guardiand -- /guardiand template delegated-guardians-config --config "{\"4\": {\"keys\": [${address1}, ${address2}], \"threshold\": 2}}" --config-id 0 > ${tmpFile}

for i in $(seq ${currentNumGuardians})
do
  # create guardian index: [0-18]
  guardianIndex=$((i-1))

  # create the governance guardian set update prototxt file in the container
  echo "created governance file for guardian-${guardianIndex}"
  kubectl cp ${tmpFile} ${namespace}/guardian-${guardianIndex}:/tmp/governance-vaa.prototxt -c guardiand

  # inject the guardian set update
  kubectl exec -n ${namespace} guardian-${guardianIndex} -c guardiand -- /guardiand admin governance-vaa-inject --socket $sock /tmp/governance-vaa.prototxt
  echo "injected governance VAA for guardian-${guardianIndex}"
done

# wait for the guardians to reach quorum about the new guardian set
sleep 30 # probably overkill, but some waiting is required.


function get_sequence_from_prototext {
    path=${1}
    while IFS= read -r line
    do
        parts=($line)
        if [ "${parts[0]}" == "sequence:" ]; then
            echo "${parts[1]}"
        fi
    done < "$path"
}
sequence=$(get_sequence_from_prototext ${tmpFile})
echo "got sequence: ${sequence} from ${tmpFile}"

# get vaa
governanceChain="1"
governanceAddress="0000000000000000000000000000000000000000000000000000000000000004"

vaaUrl="${guardianPublicWebBaseUrl}/v1/signed_vaa/${governanceChain}/${governanceAddress}/${sequence}"
echo "going to call to fetch VAA: ${vaaUrl}"

# proto endpoints supply a base64 encoded VAA
b64Vaa=$(curl ${vaaUrl} | jq ".vaaBytes")
echo "got bas64 VAA: ${b64Vaa}"

function base64_to_hex {
    b64Str=${1}
    echo "$b64Str" | tr -d '"' | base64 -d | hexdump -v -e '/1 "%02x" '
}

# transform base54 to hex
hexVaa=$(base64_to_hex ${b64Vaa})
echo "got hex VAA: ${hexVaa}"

txHash=$(cast send --rpc-url $RPC_URL --private-key $PRIV_KEY $DELEGATED_GUARDIANS_ADDRESS "submitConfig(bytes)" "0x$hexVaa" --json | jq -r '.transactionHash')
echo "tx hash: ${txHash} . waiting 30 secs..." 
sleep 30

echo "submitted VAA to ${DELEGATED_GUARDIANS_ADDRESS}"

echo "Configuration for chain ID 4:"
cast call $DELEGATED_GUARDIANS_ADDRESS "getConfig(uint16)((uint16,uint32,uint8,address[]))" 4 --rpc-url $RPC_URL
