#!/usr/bin/env bash
# This script submits a guardian set update using the VAA injection admin command.
# First argument is node to submit to. Second argument is current set index.
set -e

node=$1
idx=$2
localPath=./scripts/new-guardianset.prototxt
containerPath=/tmp/new-guardianset.prototxt
sock=/tmp/admin.sock

# Create a guardian set update VAA, pipe stdout to a local file.
kubectl exec -n wormhole guardian-${node} -c guardiand -- /guardiand template guardian-set-update --num=1 --idx=${idx} > ${localPath}

# Copy the local VAA prototext to a pod's file system.
kubectl cp ${localPath} wormhole/guardian-${node}:${containerPath} -c guardiand

# Verify the contents of the VAA prototext file and print the result. The digest incorporates the current time and is NOT deterministic.
kubectl exec -n wormhole guardian-${node} -c guardiand -- /guardiand admin governance-vaa-verify $containerPath

# Submit to node
kubectl exec -n wormhole guardian-${node} -c guardiand -- /guardiand admin governance-vaa-inject --socket $sock $containerPath
