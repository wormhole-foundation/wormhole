#!/usr/bin/env bash
# This script submits a guardian set update using the VAA injection admin command.
# First argument is node to submit to. Second argument is current set index.
set -e

node=$1
idx=$2
path=/tmp/new-guardianset.prototxt
sock=/tmp/admin.sock

# Create a no-op update that sets the same 1-node guardian set again.
kubectl exec -n wormhole guardian-${node} -c guardiand -- /guardiand template guardian-set-update --num=1 --idx=${idx} $path

# Verify and print resulting result. The digest incorporates the current time and is NOT deterministic.
kubectl exec -n wormhole guardian-${node} -c guardiand -- /guardiand admin governance-vaa-verify $path

# Submit to node
kubectl exec -n wormhole guardian-${node} -c guardiand -- /guardiand admin governance-vaa-inject --socket $sock $path
