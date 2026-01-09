#!/usr/bin/env bash
# This script allows devnet initialization of the delegated guardian set configs.
# First argument is the delegated config. This is a map of key-value pairs where:
#   key = chain name
#   value = {"chain_id": <chain_id>, "ordinals": [<pod ordinals>], "threshold": <threshold>}
# eg:- 
# {
#   "evm2": {
#     "chain_id": 4,
#     "ordinals": [
#       1,
#       2
#     ],
#     "threshold": 2
#   }
# }
# 
set -e

# wait for the guardians to establish networking
sleep 20

delegated_config="$1"
echo "delegated_config ${delegated_config}"

webHost=$2
echo "webHost ${webHost}"

namespace=$3
echo "namespace ${namespace}"

key=0x4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d # one of the Ganche defaults
devnetRPC="http://${webHost}:8545"

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
guardianAddresses=$(echo ${guardianSet} | jq -c ".addresses")
currentNumGuardians=$(echo ${guardianSet} | jq ".addresses | length")
echo "currentIndex: ${currentIndex}"
echo "currentNumGuardians: ${currentNumGuardians}"
echo "guardianAddresses: ${guardianAddresses}"

# fetch addresses from the eth-devnet (delegated guardians is only deployed in eth-devnet)
wormholeAddress=$(kubectl exec -n "${namespace}" eth-devnet-0 -c tests -- \
  jq -r '.returns.deployedAddress.value' /home/node/app/ethereum/broadcast/DeployCore.s.sol/1337/run-latest.json)
echo "wormholeAddress: $wormholeAddress"

delegatedGuardiansAddress=$(kubectl exec -n "${namespace}" eth-devnet-0 -c tests -- \
  jq -r '.returns.deployedDelegatedGuardians.value' /home/node/app/ethereum/broadcast/DeployWormholeDelegatedGuardians.s.sol/1337/run-latest.json)
echo "delegatedGuardiansAddress: $delegatedGuardiansAddress"

# convert delegated_config into expected config format for delegate-guardians-config command
config="$(
  echo "${delegated_config}" | 
  jq -c --argjson guardians "${guardianAddresses}" '
    with_entries(.value | {
      key: (.chain_id | tostring),
      value: {
          keys: (.ordinals | map($guardians[.])),
          threshold
      }
    })
  '
)"
echo "config: ${config}"

echo "creating guardian set update governance message template prototext"
kubectl exec -n "${namespace}" guardian-0 -c guardiand -- /guardiand template delegated-guardians-config --config "${config}" --config-id 0 > ${tmpFile}

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

# transform base64 to hex
hexVaa=$(base64_to_hex ${b64Vaa})
echo "got hex VAA: ${hexVaa}"

txHash=$(cast send --rpc-url "${devnetRPC}" --private-key "${key}" "${delegatedGuardiansAddress}" "submitConfig(bytes)" "0x$hexVaa" --json | jq -r '.transactionHash')

# give some time for guardians to observe the tx and update their state
sleep 30

for chain_id in $(echo "${config}" | jq -r 'keys[]')
do
  # fetch and parse json body
  echo "going to fetch configuration for chain ID ${chain_id}"
  result=$(cast call "${delegatedGuardiansAddress}" "getConfig(uint16)((uint16,uint32,uint8,address[]))" "${chain_id}" --rpc-url "${devnetRPC}" --json | jq -r '
    .[0]
    | sub("^\\("; "") | sub("\\)$"; "")
    | capture("^(?<chain_id>\\d+),\\s*(?<timestamp>\\d+),\\s*(?<threshold>\\d+),\\s*\\[(?<keys>.*)\\]$")
    | [
        .chain_id,
        .timestamp,
        .threshold,
        (.keys | split(",") | map(gsub("^\\s+|\\s+$";"")) | sort | @json)
      ]
    | @tsv
    '
  )
  read -r actual_chain_id timestamp actual_threshold actual_guardians <<< "${result}"
  echo "chain_id: ${actual_chain_id}, timestamp: ${timestamp}, threshold: ${actual_threshold}, keys: ${actual_guardians}"

  # verify configuration is as expected
  if [ ${chain_id} -ne ${actual_chain_id} ]; then
    echo "invalid chain_id — expected: ${chain_id}; got ${actual_chain_id}"
    exit 1
  fi

  expected_threshold=$(echo "${config}" | jq -r --arg chain_id "${chain_id}" '.[$chain_id].threshold')
  if [ ${expected_threshold} -ne ${actual_threshold} ]; then
    echo "invalid threshold — expected: ${expected_threshold}; got ${actual_threshold}"
    exit 1
  fi
  
  expected_guardians=$(echo "${config}" | jq -c --arg chain_id "${chain_id}" '.[$chain_id].keys | sort')
  if [ "${expected_guardians}" != "${actual_guardians}" ]; then
    echo "invalid guardians — expected: ${expected_guardians}; got ${actual_guardians}"
    exit 1
  fi

  echo "configuration for chain ID ${chain_id} is as expected."
done

echo "delegated-guardian-set-preset.sh succeeded."

echo "Waiting for guardians to fully reload delegated guardian configuration..."
sleep 60
