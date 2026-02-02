#!/bin/bash
# Sign and upload guardian peer data to the peer discovery server using AWS KMS.
# Usage: ./register-peer-kms.sh <GUARDIAN_KEY_ARN> <CERT_PATH> <TLS_HOSTNAME> <TLS_PORT> <PEER_SERVER_URL>

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
PROJECT_ROOT="${SCRIPT_DIR}/../.."

log_info() { echo "[INFO] $1"; }
log_error() { echo "[ERROR] $1"; }

if [ $# -lt 5 ]; then
    echo "Usage: $0 <GUARDIAN_KEY_ARN> <CERT_PATH> <TLS_HOSTNAME> <TLS_PORT> <PEER_SERVER_URL>"
    echo ""
    echo "Arguments:"
    echo "  GUARDIAN_KEY_ARN  - AWS KMS ARN for the guardian's private key"
    echo "  CERT_PATH         - Path to the TLS certificate"
    echo "  TLS_HOSTNAME      - Hostname for this guardian's DKG server"
    echo "  TLS_PORT          - Port for this guardian's DKG server"
    echo "  PEER_SERVER_URL   - URL of the peer discovery server"
    exit 1
fi

GUARDIAN_KEY_ARN="$1"
CERT_PATH="$2"
TLS_HOSTNAME="$3"
TLS_PORT="$4"
PEER_SERVER_URL="$5"

if [ ! -f "${CERT_PATH}" ]; then
    log_error "Certificate file not found: ${CERT_PATH}"
    exit 1
fi

# Export AWS credentials from SSO session
log_info "Exporting AWS credentials from SSO session..."
if ! command -v aws &> /dev/null; then
    log_error "AWS CLI not found. Please install AWS CLI."
    exit 1
fi

# Use AWS_PROFILE if set, otherwise use default
profile_option=""
if [ -n "${AWS_PROFILE:-}" ]; then
    profile_option="--profile ${AWS_PROFILE}"
    log_info "Using AWS profile: ${AWS_PROFILE}"
fi

log_info "Running: aws configure export-credentials --format env-no-export ${profile_option}"
AWS_CREDS_OUTPUT=$(aws configure export-credentials --format env-no-export ${profile_option} 2>&1) || {
    log_error "Failed to export AWS credentials:"
    log_error "${AWS_CREDS_OUTPUT}"
    log_error "Please run 'aws sso login --profile <profile>' first."
    exit 1
}

log_info "Parsing credentials..."
log_info "Raw output has ${#AWS_CREDS_OUTPUT} characters"

# Parse credentials from env-no-export format
# Use || true to prevent set -e from killing the script if grep finds nothing
AWS_ACCESS_KEY_ID=$(echo "$AWS_CREDS_OUTPUT" | grep "^AWS_ACCESS_KEY_ID=" | cut -d= -f2- || true)
AWS_SECRET_ACCESS_KEY=$(echo "$AWS_CREDS_OUTPUT" | grep "^AWS_SECRET_ACCESS_KEY=" | cut -d= -f2- || true)
AWS_SESSION_TOKEN=$(echo "$AWS_CREDS_OUTPUT" | grep "^AWS_SESSION_TOKEN=" | cut -d= -f2- || true)

log_info "Parsed AWS_ACCESS_KEY_ID length: ${#AWS_ACCESS_KEY_ID}"
log_info "Parsed AWS_SECRET_ACCESS_KEY length: ${#AWS_SECRET_ACCESS_KEY}"
log_info "Parsed AWS_SESSION_TOKEN length: ${#AWS_SESSION_TOKEN}"

if [ -z "${AWS_ACCESS_KEY_ID}" ] || [ -z "${AWS_SECRET_ACCESS_KEY}" ]; then
    log_error "Failed to parse AWS credentials from output:"
    log_error "${AWS_CREDS_OUTPUT}"
    log_error "Please run 'aws sso login' first."
    exit 1
fi

log_info "AWS credentials exported successfully (Key ID: ${AWS_ACCESS_KEY_ID:0:8}...)"

export DOCKER_BUILDKIT=1

# TSS_E2E_DOCKER_BUILDER should NOT be used in production.
builder_option=""
if [ -n "${TSS_E2E_DOCKER_BUILDER:-}" ]; then
    builder_option="--builder ${TSS_E2E_DOCKER_BUILDER} --network=host"
fi

# Create temporary files for AWS credentials secrets
AWS_CREDS_DIR=$(mktemp -d)
trap "rm -rf ${AWS_CREDS_DIR}" EXIT
echo -n "${AWS_ACCESS_KEY_ID}" > "${AWS_CREDS_DIR}/aws_access_key_id"
echo -n "${AWS_SECRET_ACCESS_KEY}" > "${AWS_CREDS_DIR}/aws_secret_access_key"
echo -n "${AWS_SESSION_TOKEN}" > "${AWS_CREDS_DIR}/aws_session_token"

docker build ${builder_option} \
    --file "${PROJECT_ROOT}/ts-pkgs/peer-client/Dockerfile" \
    --secret id=cert.pem,src="${CERT_PATH}" \
    --secret id=aws_access_key_id,src="${AWS_CREDS_DIR}/aws_access_key_id" \
    --secret id=aws_secret_access_key,src="${AWS_CREDS_DIR}/aws_secret_access_key" \
    --secret id=aws_session_token,src="${AWS_CREDS_DIR}/aws_session_token" \
    --build-arg TLS_HOSTNAME="${TLS_HOSTNAME}" \
    --build-arg TLS_PORT="${TLS_PORT}" \
    --build-arg PEER_SERVER_URL="${PEER_SERVER_URL}" \
    --build-arg GUARDIAN_PRIVATE_KEY_ARN="${GUARDIAN_KEY_ARN}" \
    "${PROJECT_ROOT}"

log_info "Registration complete"

TLS_KEYS_DIR="$(dirname "${CERT_PATH}")"

if [ -z "${SKIP_NEXT_STEP_HINT:-}" ]; then
    echo ""
    echo "=============================================="
    echo "NEXT STEP: Run the DKG ceremony"
    echo "=============================================="
    echo ""
    echo "Run the following command from the rollout-scripts directory:"
    echo ""
    echo "  ./run-dkg.sh \\"
    echo "    ${TLS_KEYS_DIR} \\"
    echo "    ${TLS_HOSTNAME} \\"
    echo "    ${TLS_PORT} \\"
    echo "    ${PEER_SERVER_URL} \\"
    echo "    <ETHEREUM_RPC_URL> \\"
    echo "    <WORMHOLE_ADDRESS> \\"
    echo "    ${GUARDIAN_KEY_ARN}"
    echo ""
    echo "Where:"
    echo "  ETHEREUM_RPC_URL  - Ethereum mainnet RPC URL"
    echo "  WORMHOLE_ADDRESS  - Wormhole contract address"
    echo ""
fi

