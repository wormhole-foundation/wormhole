#!/bin/bash
# Start the peer discovery server for DKG coordination.

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
REPO_ROOT="${SCRIPT_DIR}/../.."

log_error() { echo "[ERROR] $1" >&2; }

usage() {
  cat >&2 <<'EOF'
Usage:
  script \
    --server-port=PORT \
    --ethereum-rpc-url=URL \
    --output-peers-file=FILE

Required:
  --server-port=PORT        Port for the peer server to listen on.
  --ethereum-rpc-url=URL    Ethereum mainnet RPC URL. Must be HTTP(S).
  --output-peers-file=FILE  Output file where peers will be stored.

Optional:
  --wormhole-address=ADDR   Wormhole contract address for testnet/devnet.
  --threshold               Amount of guardians to reach quorum for a given signing session.
EOF
}

# Defaults
SERVER_PORT=""
ETHEREUM_RPC_URL=""
OUTPUT_PEERS_FILE=""
WORMHOLE_ADDRESS="0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
THRESHOLD=13

for arg in "$@"; do
  case "$arg" in
    --server-port=*)
      SERVER_PORT="${arg#*=}" ;;
    --ethereum-rpc-url=*)
      ETHEREUM_RPC_URL="${arg#*=}" ;;
    --output-peers-file=*)
      OUTPUT_PEERS_FILE="${arg#*=}" ;;
    --wormhole-address=*)
      WORMHOLE_ADDRESS="${arg#*=}" ;;
    --threshold=*)
      THRESHOLD="${arg#*=}" ;;
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
[ -n "$SERVER_PORT"       ] || missing+=("--server-port"      )
[ -n "$ETHEREUM_RPC_URL"  ] || missing+=("--ethereum-rpc-url" )
[ -n "$OUTPUT_PEERS_FILE" ] || missing+=("--output-peers-file")

OUTPUT_PEERS_DIR=$(dirname ${OUTPUT_PEERS_FILE})

if (( ${#missing[@]} )); then
  log_error "Missing required option(s): ${missing[*]}"
  echo >&2
  usage
  exit 1
fi

mkdir -p $(dirname "${OUTPUT_PEERS_FILE}")
if [ ! -f "${OUTPUT_PEERS_FILE}" ]; then
    # We want to create the file here because docker would create it
    # with the user of the daemon which could be different from the current user.
    # Also, the server needs it to be valid JSON.
    echo "[]"  > "${OUTPUT_PEERS_FILE}"
fi

# TSS_E2E_DOCKER_NETWORK should NOT be used in production
run_options=""
if [ -n "${TSS_E2E_DOCKER_NETWORK:-}" ]; then
    run_options+="--network=${TSS_E2E_DOCKER_NETWORK} "
else
    run_options+="--publish ${SERVER_PORT}:${SERVER_PORT} "
fi

docker build \
    --tag peer-server \
    --file "${REPO_ROOT}/ts-pkgs/peer-server/Dockerfile" \
    "${REPO_ROOT}"

if [ -z "${NON_INTERACTIVE:-}" ]; then
    run_options+="--interactive --tty "
fi

cat > "${OUTPUT_PEERS_DIR}/peer-server-config.json" <<EOT
{
  "port": ${SERVER_PORT},
  "ethereum": {
    "rpcUrl": "${ETHEREUM_RPC_URL}"
  },
  "wormholeContractAddress": "${WORMHOLE_ADDRESS}",
  "threshold": ${THRESHOLD},
  "peerListStore": "/peerGuardians.json"
}
EOT

docker run \
    --rm \
    --name peer-server \
    ${run_options} \
    --volume "${OUTPUT_PEERS_DIR}/peer-server-config.json":/verification-v2/ts-pkgs/peer-server/config.json \
    --volume "${OUTPUT_PEERS_FILE}":/peerGuardians.json \
    peer-server

