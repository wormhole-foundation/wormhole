#!/bin/bash
# Generate TLS key pair and certificate for TLS during DKG.

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
REPO_ROOT="${SCRIPT_DIR}/../.."

log_info()  { echo "[INFO] $1" >&2;  }
log_error() { echo "[ERROR] $1" >&2; }

usage() {
  cat >&2 <<'EOF'
Usage:
  script \
    --tls-hostname=HOST \
    --tls-public-ip=IP \
    --output-dir=DIRECTORY

Required:
  --tls-hostname=HOSTNAME  Fully qualified hostname for this guardian
  --tls-public-ip=IP       Public IP address for this guardian
  --output-dir=DIRECTORY   Directory to store generated keys
EOF
}

# Defaults
TLS_HOSTNAME=""
TLS_PUBLIC_IP=""
OUTPUT_DIR=""

for arg in "$@"; do
  case "$arg" in
    --tls-hostname=*)
      TLS_HOSTNAME="${arg#*=}" ;;
    --tls-public-ip=*)
      TLS_PUBLIC_IP="${arg#*=}" ;;
    --output-dir=*)
      OUTPUT_DIR="${arg#*=}" ;;
    --help|-h)
      usage; exit 0 ;;
    *)
      echo "Unknown option: $arg" >&2
      echo >&2
      usage
      exit 1 ;;
  esac
done

missing=()
[ -n "$TLS_HOSTNAME"  ] || missing+=("--tls-hostname" )
[ -n "$TLS_PUBLIC_IP" ] || missing+=("--tls-public-ip")
[ -n "$OUTPUT_DIR"    ] || missing+=("--output-dir"   )

if (( ${#missing[@]} )); then
  log_error "Missing required option(s): ${missing[*]}"
  echo >&2
  usage
  exit 1
fi

mkdir -p "${OUTPUT_DIR}"

if [ -f "${OUTPUT_DIR}/key.pem" ] || [ -f "${OUTPUT_DIR}/cert.pem" ]; then
    if [ -n "${FORCE_OVERWRITE:-}" ]; then
        log_info "Overwriting existing TLS credentials (FORCE_OVERWRITE=1)"
    elif [ -n "${NON_INTERACTIVE:-}" ]; then
        log_error "TLS credentials already exist. Set FORCE_OVERWRITE=1 to overwrite."
        exit 1
    else
        read -p "TLS credentials already exist. Overwrite? (y/N) " -n 1 -r
        echo
        [[ ! $REPLY =~ ^[Yy]$ ]] && exit 0
    fi
fi

docker build \
    --tag tls-gen \
    --file "${REPO_ROOT}/ts-pkgs/peer-client/tls.Dockerfile" \
    "${REPO_ROOT}"

docker run \
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

if [ -z "${SKIP_NEXT_STEP_HINT:-}" ]; then
    echo ""
    echo "=============================================="
    echo "NEXT STEP: Register your peer with the discovery server"
    echo "=============================================="
    echo ""
    echo "Run the following command from the rollout-scripts directory:"
    echo ""
    echo "  ./register-peer.sh \\"
    echo "    <Guardian key option> \\"
    echo "    --tls-certificate=${OUTPUT_DIR}/cert.pem \\"
    echo "    --tls-hostname=${TLS_HOSTNAME} \\"
    echo "    --tls-port=<TLS_PORT> \\"
    echo "    --peer-server-url=<PEER_SERVER_URL>"
    echo ""
    echo "Where:"
    echo "  TLS_PORT            - Port your DKG server will listen on (e.g., 8443)"
    echo "  PEER_SERVER_URL     - URL of the peer discovery server"
    echo "Guardian key option must be exactly one of these:"
    echo "  --key=<KEY_PATH>    - Path to the guardian's Wormhole private key"
    echo "  --arn=<AWS_KMS_ARN> - ARN of AWS KMS key"
    echo ""
fi
