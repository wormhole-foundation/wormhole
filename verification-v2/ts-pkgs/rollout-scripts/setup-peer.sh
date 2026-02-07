#!/bin/bash
# Generate TLS credentials and register peer with the discovery server.
# This is a unified script that runs generate-tls.sh and register-peer.sh.

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"

log_error() { echo "[ERROR] $1" >&2; }

usage() {
  cat >&2 <<EOF
Usage:
  $0 \
    <Guardian key option> \
    --tls-hostname=HOST \
    --tls-port=PORT \
    --tls-public-ip=IP \
    --output-dir=DIRECTORY \
    --peer-server-url=URL

Required:
  --tls-hostname=HOST      Hostname for this guardian's DKG server.
  --tls-public-ip=IP       Public IP address for this guardian
  --tls-port=PORT          Port for this guardian's DKG server.
  --output-dir=DIRECTORY   Directory to store generated keys
  --peer-server-url=URL    URL of the peer discovery server.
Guardian key option must be exactly one of these:
  --key=KEY_PATH           Path to the guardian's Wormhole private key.
  --arn=AWS_KMS_ARN        ARN of AWS KMS key.
EOF
}

check_guardian_option_is_undefined() {
  if [ -n "${GUARDIAN_KEY_OPTION:-}" ]; then
    log_error "Guardian key option must be provided exactly once."
    echo >&2
    usage
    exit 1
  fi
}

for arg in "$@"; do
  case "$arg" in
    --tls-hostname=*)
      TLS_HOSTNAME="${arg#*=}" ;;
    --tls-public-ip=*)
      TLS_PUBLIC_IP="${arg#*=}" ;;
    --tls-port=*)
      TLS_PORT="${arg#*=}" ;;
    --output-dir=*)
      OUTPUT_DIR="${arg#*=}" ;;
    --peer-server-url=*)
      PEER_SERVER_URL="${arg#*=}" ;;
    --key=*)
      check_guardian_option_is_undefined
      GUARDIAN_KEY_OPTION=--key
      GUARDIAN_KEY_VALUE="${arg#*=}" ;;
    --arn=*)
      check_guardian_option_is_undefined
      GUARDIAN_KEY_OPTION=--arn
      GUARDIAN_KEY_VALUE="${arg#*=}" ;;
    --help|-h)
      usage; exit 0 ;;
    *)
      log_error "Unknown option: $arg"
      echo >&2
      usage
      exit 1 ;;
  esac
done

echo ""
echo "=============================================="
echo "STEP 1/2: Generating TLS credentials"
echo "=============================================="
echo ""

SKIP_NEXT_STEP_HINT=1 "${SCRIPT_DIR}/generate-tls.sh" \
  --tls-hostname="${TLS_HOSTNAME}" \
  --tls-public-ip="${TLS_PUBLIC_IP}" \
  --output-dir="${OUTPUT_DIR}"

echo ""
echo "=============================================="
echo "STEP 2/2: Registering peer with discovery server"
echo "=============================================="
echo ""

"${SCRIPT_DIR}/register-peer.sh" \
  "${GUARDIAN_KEY_OPTION}"="${GUARDIAN_KEY_VALUE}" \
  --tls-certificate="${OUTPUT_DIR}/cert.pem" \
  --tls-hostname="${TLS_HOSTNAME}" \
  --tls-port="${TLS_PORT}" \
  --peer-server-url="${PEER_SERVER_URL}"

