#!/bin/bash
# Generate TLS credentials and register peer with the discovery server.
# This is a unified script that runs generate-tls.sh and register-peer.sh.

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"

if [ $# -lt 7 ]; then
    echo "Usage: $0 <Guardian key option> <TLS_HOSTNAME> <TLS_PUBLIC_IP> <OUTPUT_DIR> <TLS_PORT> <PEER_SERVER_URL>"
    echo ""
    echo "Arguments:"
    echo "  TLS_HOSTNAME      - Fully qualified hostname for this guardian"
    echo "  TLS_PUBLIC_IP     - Public IP address for this guardian"
    echo "  OUTPUT_DIR        - Directory to store generated keys"
    echo "  TLS_PORT          - Port for this guardian's DKG process"
    echo "  PEER_SERVER_URL   - URL of the peer discovery server"
    echo "Guardian key option must be exactly one of these:"
    echo "  --key <KEY_PATH>    - Path to the guardian's Wormhole private key"
    echo "  --arn <AWS_KMS_ARN> - ARN of AWS KMS key"
    exit 1
fi

GUARDIAN_KEY_OPTION="$1"
GUARDIAN_KEY_VALUE="$2"
TLS_HOSTNAME="$3"
TLS_PUBLIC_IP="$4"
OUTPUT_DIR="$5"
TLS_PORT="$6"
PEER_SERVER_URL="$7"

echo ""
echo "=============================================="
echo "STEP 1/2: Generating TLS credentials"
echo "=============================================="
echo ""

# Run generate-tls.sh with hint suppressed
SKIP_NEXT_STEP_HINT=1 "${SCRIPT_DIR}/generate-tls.sh" \
    "${TLS_HOSTNAME}" \
    "${TLS_PUBLIC_IP}" \
    "${OUTPUT_DIR}"

echo ""
echo "=============================================="
echo "STEP 2/2: Registering peer with discovery server"
echo "=============================================="
echo ""

# Run register-peer.sh (its hint will show the next step: run-dkg.sh)
"${SCRIPT_DIR}/register-peer.sh" \
    "${GUARDIAN_KEY_OPTION}" "${GUARDIAN_KEY_VALUE}" \
    "${OUTPUT_DIR}/cert.pem" \
    "${TLS_HOSTNAME}" \
    "${TLS_PORT}" \
    "${PEER_SERVER_URL}"

