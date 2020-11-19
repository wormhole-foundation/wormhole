#!/bin/bash
# This script submits a guardian set update using the VAA injection admin command.
# First argument is node to submit to.
set -e

node=$1
path=/tmp/new-guardianset.prototxt
sock=/tmp/admin.sock

# Create a no-op update that sets the same 1-node guardian set again.
kubectl exec guardian-${node} -c guardiand -- /guardiand admin guardian-set-update-template --num=1 $path

# Verify and print resulting result. The digest incorporates the current time and is NOT deterministic.
kubectl exec guardian-${node} -c guardiand -- /guardiand admin guardian-set-update-verify $path

# Submit to node
kubectl exec guardian-${node} -c guardiand -- /guardiand admin guardian-set-update-inject --socket $sock $path
