#!/bin/bash

set -meuo pipefail
export DOCKER_BUILDKIT=1

TLS_HOSTNAME="Guardian"
TLS_PUBLIC_IP="127.0.0.1"
TLS_BASE_PORT="3001"
PEER_SERVER_URL="http://peer-server:3000"
ETHEREUM_RPC_URL="http://anvil-with-verifier:8545"
WORMHOLE_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"

export DOCKER_BUILDER="dkg-builder"
export DOCKER_BUILD_NETWORK="host"
export DOCKER_NETWORK="dkg-test"
export NON_INTERACTIVE=1
export FORCE_OVERWRITE=1
export SKIP_NEXT_STEP_HINT=1

# AWS KMS ARNs for guardian private keys
# TODO: Fill in with actual ARNs
GUARDIAN_PRIVATE_KEY_ARNS=(
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_0"   # Guardian 0
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_1"   # Guardian 1
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_2"   # Guardian 2
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_3"   # Guardian 3
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_4"   # Guardian 4
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_5"   # Guardian 5
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_6"   # Guardian 6
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_7"   # Guardian 7
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_8"   # Guardian 8
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_9"   # Guardian 9
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_10"  # Guardian 10
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_11"  # Guardian 11
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_12"  # Guardian 12
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_13"  # Guardian 13
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_14"  # Guardian 14
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_15"  # Guardian 15
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_16"  # Guardian 16
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_17"  # Guardian 17
  "arn:aws:kms:REGION:ACCOUNT_ID:key/KEY_ID_18"  # Guardian 18
)

for i in "${!GUARDIAN_PRIVATE_KEY_ARNS[@]}"
do
  mkdir -p "out/$i/keys"
done

until docker logs peer-server 2>/dev/null | grep "Peer server running on"
do
  sleep 1
done

for i in "${!GUARDIAN_PRIVATE_KEY_ARNS[@]}"
do
  # TODO: Create/update setup-peer-kms.sh that accepts ARN instead of key path
  ../../../../rollout-scripts/setup-peer-kms.sh \
    "${TLS_HOSTNAME}$i" \
    "${TLS_PUBLIC_IP}" \
    "out/$i/keys" \
    "${GUARDIAN_PRIVATE_KEY_ARNS[$i]}" \
    "$((TLS_BASE_PORT + i))" \
    "${PEER_SERVER_URL}" &
done

wait

docker build --tag dkg-client --file ../../../../peer-client/dkg.Dockerfile ../../../../..

for i in "${!GUARDIAN_PRIVATE_KEY_ARNS[@]}"
do
  # TODO: Create/update run-dkg-kms.sh that accepts ARN instead of key path
  ../../../../rollout-scripts/run-dkg-kms.sh \
    "out/$i/keys" \
    "${TLS_HOSTNAME}$i" \
    "$((TLS_BASE_PORT + i))" \
    "${PEER_SERVER_URL}" \
    "${ETHEREUM_RPC_URL}" \
    "${WORMHOLE_ADDRESS}" \
    "${GUARDIAN_PRIVATE_KEY_ARNS[$i]}" &
done

wait

