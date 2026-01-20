#!/bin/bash
# Generate TLS key pair and certificate for mTLS during DKG.
# Usage: ./generate-tls.sh <TLS_HOSTNAME> <TLS_PUBLIC_IP> <OUTPUT_DIR>

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

log_info() { echo "[INFO] $1"; }
log_error() { echo "[ERROR] $1"; }

if [ $# -lt 3 ]; then
    echo "Usage: $0 <TLS_HOSTNAME> <TLS_PUBLIC_IP> <OUTPUT_DIR>"
    echo ""
    echo "Arguments:"
    echo "  TLS_HOSTNAME   - Fully qualified hostname for this guardian"
    echo "  TLS_PUBLIC_IP  - Public IP address for this guardian"
    echo "  OUTPUT_DIR     - Directory to store generated keys"
    exit 1
fi

TLS_HOSTNAME="$1"
TLS_PUBLIC_IP="$2"
OUTPUT_DIR="$3"

# Validate IP address (IPv4 or IPv6)
IPV4_REGEX='^[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+$'
IPV6_REGEX='^([0-9a-fA-F]{0,4}:){1,7}[0-9a-fA-F]{0,4}$|^::$|^::1$'
if ! [[ "$TLS_PUBLIC_IP" =~ $IPV4_REGEX ]] && ! [[ "$TLS_PUBLIC_IP" =~ $IPV6_REGEX ]]; then
    log_error "TLS_PUBLIC_IP must be a valid IPv4 or IPv6 address"
    exit 1
fi

mkdir -p "${OUTPUT_DIR}"
OUTPUT_DIR="$(cd "${OUTPUT_DIR}" && pwd)"

if [ -f "${OUTPUT_DIR}/key.pem" ] || [ -f "${OUTPUT_DIR}/cert.pem" ]; then
    read -p "TLS credentials already exist. Overwrite? (y/N) " -n 1 -r
    echo
    [[ ! $REPLY =~ ^[Yy]$ ]] && exit 0
fi

docker build \
    --tag tls-gen \
    --file "${REPO_ROOT}/ts-pkgs/peer-client/tls.Dockerfile" \
    "${REPO_ROOT}"

docker run \
    -it \
    --rm \
    --mount type=bind,src="${OUTPUT_DIR}",dst=/keys \
    --env TLS_HOSTNAME="${TLS_HOSTNAME}" \
    --env TLS_PUBLIC_IP="${TLS_PUBLIC_IP}" \
    tls-gen

if [ -f "${OUTPUT_DIR}/key.pem" ] && [ -f "${OUTPUT_DIR}/cert.pem" ]; then
    log_info "TLS credentials saved to ${OUTPUT_DIR}"
else
    log_error "Failed to generate TLS credentials"
    exit 1
fi

