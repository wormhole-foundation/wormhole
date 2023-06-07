#!/usr/bin/env bash
# This script submits a guardian set update using the VAA injection admin command.
# First argument is the number of guardians for the new guardian set.
set -e

# wait for the guardians to establish networking
sleep 20

newNumGuardians=$1
echo "new number of guardians: ${newNumGuardians}"

webHost=$2
echo "webHost ${webHost}"

namespace=$3
echo "namespace ${namespace}"

# file & path to save governance VAA
tmpFile=$(mktemp -q)
if [ $? -ne 0 ]; then
    echo "$0: Can't create temp file, bye.." 1>&2;
    exit 1
fi
trap 'rm -f -- "$tmpFile"' EXIT

# the admin socket of the devnet guardians. used for executing commands in guardian pods.
sock=/tmp/admin.sock

guardianPublicWebBaseUrl="${webHost}:7071"

currentGuardianSetUrl="${guardianPublicWebBaseUrl}/v1/guardianset/current"

# fetch result and parse json body:
guardianSet=$(curl ${currentGuardianSetUrl} | jq ".guardianSet")
currentIndex=$(echo ${guardianSet} | jq ".index")
currentNumGuardians=$(echo ${guardianSet} | jq ".addresses | length")
echo "currentIndex: ${currentIndex}"
echo "currentNumGuardians ${currentNumGuardians}"


if [ ${currentNumGuardians} == ${newNumGuardians} ]; then
    echo "number of guardians is as expected."
    exit 0
fi

echo "creating guardian set update governance message template prototext"
minikube kubectl -- exec -n ${namespace} guardian-0 -c guardiand -- /guardiand template guardian-set-update --num=${newNumGuardians} --idx=${currentIndex} > ${tmpFile}

# for i in $(seq ${newNumGuardians})
for i in $(seq ${currentNumGuardians})
do
  # create guardian index: [0-18]
  guardianIndex=$((i-1))

  # create the governance guardian set update prototxt file in the container
  echo "created governance file for guardian-${guardianIndex}"
  minikube kubectl -- cp ${tmpFile} ${namespace}/guardian-${guardianIndex}:${tmpFile} -c guardiand

  # inject the guardian set update
  minikube kubectl -- exec -n ${namespace} guardian-${guardianIndex} -c guardiand -- /guardiand admin governance-vaa-inject --socket $sock $tmpFile
  echo "injected governance VAA for guardian-${guardianIndex}"
done

# wait for the guardians to reach quorum about the new guardian set
sleep 30 # probably overkill, but some waiting is required.

function get_sequence_from_prototext {
    path=${1}
    while IFS= read -r line
    do
        parts=($line)
        if [ ${parts[0]} == "sequence:" ]; then
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
    echo $b64Str | base64 -d -i | hexdump -v -e '/1 "%02x" '
}

# transform base54 to hex
hexVaa=$(base64_to_hex ${b64Vaa})
echo "got hex VAA: ${hexVaa}"

# fire off the Golang script in clients/eth:
./scripts/send-vaa.sh $webHost $hexVaa

# give some time for guardians to observe the tx and update their state
sleep 30

# fetch result and parse json body:
echo "going to fetch current guardianset from ${currentGuardianSetUrl}"
nextGuardianSet=$(curl ${currentGuardianSetUrl} | jq ".guardianSet")
nextIndex=$(echo ${nexGuardianSet} | jq ".index")
nextNumGuardians=$(echo ${nextGuardianSet} | jq ".addresses | length")
echo "nextIndex: ${nextIndex}"
echo "nextNumGuardians ${nextNumGuardians}"

if [ ${nextNumGuardians} == ${newNumGuardians} ]; then
    echo "number of guardians is as expected."
else
    echo "number of guardians is not as expected. number of guardians in set: ${nextNumGuardians}."
    exit 1
fi

echo "update-guardian-set.sh succeeded."
