#!/bin/bash
# Generate TLS credentials and register peer with the discovery server using AWS KMS.
# This is a unified script that runs generate-tls.sh and register-peer-kms.sh.
# Usage: ./setup-peer-kms.sh <TLS_HOSTNAME> <TLS_PUBLIC_IP> <OUTPUT_DIR> <GUARDIAN_KEY_ARN> <TLS_PORT> <PEER_SERVER_URL>

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"

if [ $# -lt 6 ]; then
    echo "Usage: $0 <TLS_HOSTNAME> <TLS_PUBLIC_IP> <OUTPUT_DIR> <GUARDIAN_KEY_ARN> <TLS_PORT> <PEER_SERVER_URL>"
    echo ""
    echo "Arguments:"
    echo "  TLS_HOSTNAME      - Fully qualified hostname for this guardian"
    echo "  TLS_PUBLIC_IP     - Public IP address for this guardian"
    echo "  OUTPUT_DIR        - Directory to store generated keys"
    echo "  GUARDIAN_KEY_ARN  - AWS KMS ARN for the guardian's private key"
    echo "  TLS_PORT          - Port for this guardian's DKG server"
    echo "  PEER_SERVER_URL   - URL of the peer discovery server"
    exit 1
fi

TLS_HOSTNAME="$1"
TLS_PUBLIC_IP="$2"
OUTPUT_DIR="$3"
GUARDIAN_KEY_ARN="$4"
TLS_PORT="$5"
PEER_SERVER_URL="$6"

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
echo "STEP 2/2: Registering peer with discovery server (KMS)"
echo "=============================================="
echo ""

# Run register-peer-kms.sh (its hint will show the next step: run-dkg.sh)
"${SCRIPT_DIR}/register-peer-kms.sh" \
    "${GUARDIAN_KEY_ARN}" \
    "${OUTPUT_DIR}/cert.pem" \
    "${TLS_HOSTNAME}" \
    "${TLS_PORT}" \
    "${PEER_SERVER_URL}"

