#!/bin/bash

# Useful to debug issues with the e2e test (KMS version).
set -meuo pipefail
export DOCKER_BUILDKIT=1

TLS_HOSTNAME="Guardian0"
TLS_PUBLIC_IP="127.0.0.1"
TLS_PORT="3001"
PEER_SERVER_URL="http://peer-server:3000"
ETHEREUM_RPC_URL="http://anvil-with-verifier:8545"
WORMHOLE_ADDRESS="0x5FbDB2315678afecb367f032d93F642f64180aa3"
# TODO: Fill in with actual ARN
GUARDIAN_PRIVATE_KEY_ARN="arn:aws:kms:us-east-2:581679387567:key/d8e672c3-401a-4322-a86a-f089b5ae077c"

export TSS_E2E_DOCKER_BUILDER="dkg-builder"
export TSS_E2E_DOCKER_NETWORK="dkg-test"
export NON_INTERACTIVE=1
export FORCE_OVERWRITE=1
export SKIP_NEXT_STEP_HINT=1

# Paths relative to ts-pkgs/peer-e2e/tests/e2e/ (the expected working directory)
ROLLOUT_SCRIPTS_DIR="../../../rollout-scripts"
OUTPUT_DIR="./rollout-scripts/out/0/keys"

mkdir -p "${OUTPUT_DIR}"

until docker logs peer-server 2>/dev/null | grep "Peer server running on"
do
  sleep 1
done

"${ROLLOUT_SCRIPTS_DIR}/setup-peer-kms.sh" \
  "${TLS_HOSTNAME}" \
  "${TLS_PUBLIC_IP}" \
  "${OUTPUT_DIR}" \
  "${GUARDIAN_PRIVATE_KEY_ARN}" \
  "${TLS_PORT}" \
  "${PEER_SERVER_URL}"

"${ROLLOUT_SCRIPTS_DIR}/run-dkg.sh" \
  "${OUTPUT_DIR}" \
  "${TLS_HOSTNAME}" \
  "${TLS_PORT}" \
  "${PEER_SERVER_URL}" \
  "${ETHEREUM_RPC_URL}" \
  "${WORMHOLE_ADDRESS}" \
  "${GUARDIAN_PRIVATE_KEY_ARN}"

