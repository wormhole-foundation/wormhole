#!/bin/bash
# Run the DKG ceremony. Polls for all peers, then generates key shards.

set -euo pipefail

SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
REPO_ROOT="${SCRIPT_DIR}/../.."

log_error() { echo "[ERROR] $1" >&2; }

usage() {
  cat >&2 <<EOF
Usage:
  $0 \
    --tls-keys-dir=DIR \
    --tls-hostname=HOST \
    --tls-port=PORT \
    --peer-server-url=URL \
    --ethereum-rpc-url=URL

Required:
  --tls-keys-dir=DIR        Directory containing key.pem and cert.pem (also used for DKG outputs).
  --tls-hostname=HOST       Hostname for this guardian's DKG server.
  --tls-port=PORT           Port for this guardian's DKG server.
  --peer-server-url=URL     URL of the peer discovery server.
  --ethereum-rpc-url=URL    Ethereum RPC URL. Must be HTTP(S).

Optional:
  --wormhole-address=ADDR   Wormhole contract address for testnet/devnet.
  --threshold=INT           Amount of guardians to reach quorum for a given signing session.
  --etc-hosts-override=FILE Override of /etc/hosts for container running DKG.
EOF
# TODO: add option to override /etc/hosts in container
}

# Defaults
TLS_KEYS_DIR=""
TLS_HOSTNAME=""
TLS_PORT=""
PEER_SERVER_URL=""
ETHEREUM_RPC_URL=""
WORMHOLE_ADDRESS="0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
THRESHOLD=13

for arg in "$@"; do
  case "$arg" in
    --tls-keys-dir=*)
      TLS_KEYS_DIR="${arg#*=}" ;;
    --tls-hostname=*)
      TLS_HOSTNAME="${arg#*=}" ;;
    --tls-port=*)
      TLS_PORT="${arg#*=}" ;;
    --peer-server-url=*)
      PEER_SERVER_URL="${arg#*=}" ;;
    --ethereum-rpc-url=*)
      ETHEREUM_RPC_URL="${arg#*=}" ;;
    --wormhole-address=*)
      WORMHOLE_ADDRESS="${arg#*=}" ;;
    --threshold=*)
      THRESHOLD="${arg#*=}" ;;
    --etc-hosts-override=*)
      ETC_HOSTS_OVERRIDE="${arg#*=}" ;;
    --help|-h)
      usage; exit 0 ;;
    *)
      log_error "Unknown option: $arg"
      echo >&2
      usage
      exit 1 ;;
  esac
done

missing=()
[ -n "$TLS_KEYS_DIR"     ] || missing+=("--tls-keys-dir"    )
[ -n "$TLS_HOSTNAME"     ] || missing+=("--tls-hostname"    )
[ -n "$TLS_PORT"         ] || missing+=("--tls-port"        )
[ -n "$PEER_SERVER_URL"  ] || missing+=("--peer-server-url" )
[ -n "$ETHEREUM_RPC_URL" ] || missing+=("--ethereum-rpc-url")

if (( ${#missing[@]} )); then
  log_error "Missing required option(s): ${missing[*]}"
  echo >&2
  usage
  exit 1
fi

if [ ! -d "${TLS_KEYS_DIR}" ]; then
  log_error "TLS keys directory not found: ${TLS_KEYS_DIR}"
  exit 1
fi

if [ ! -f "${TLS_KEYS_DIR}/key.pem" ]; then
  log_error "TLS private key not found: ${TLS_KEYS_DIR}/key.pem"
  exit 1
fi

if [ ! -f "${TLS_KEYS_DIR}/cert.pem" ]; then
  log_error "TLS certificate not found: ${TLS_KEYS_DIR}/cert.pem"
  exit 1
fi

# TSS_E2E_DOCKER_NETWORK should NOT be used in production
run_options=""
if [ -n "${TSS_E2E_DOCKER_NETWORK:-}" ]; then
  run_options+="--network=${TSS_E2E_DOCKER_NETWORK} "
else
  run_options+="--publish ${TLS_PORT}:${TLS_PORT} "
fi

if [ -n "${ETC_HOSTS_OVERRIDE:-}" ]; then
  if [ ! -f "${ETC_HOSTS_OVERRIDE}" ]; then
    log_error "Hosts override not found: ${ETC_HOSTS_OVERRIDE}"
    exit 1
  fi
  run_options+="--volume "${ETC_HOSTS_OVERRIDE}":dst=/etc/hosts:ro "
fi

docker build --tag "dkg-client${TSS_E2E_GUARDIAN_ID:-}" --file "${REPO_ROOT}/ts-pkgs/peer-client/dkg.Dockerfile" "${REPO_ROOT}"

peer_client_config="${TLS_KEYS_DIR}/peer-client-config.json"
cat > ${peer_client_config} <<EOF
{
  "serverUrl": "${PEER_SERVER_URL}",
  "peer": {
    "hostname": "${TLS_HOSTNAME}",
    "port": ${TLS_PORT},
    "tlsX509": "/keys/cert.pem"
  },
  "threshold": ${THRESHOLD},
  "wormhole": {
    "ethereum": {
      "rpcUrl": "${ETHEREUM_RPC_URL}"
    },
    "wormholeContractAddress": "${WORMHOLE_ADDRESS}"
  }
}
EOF

if [ -z "${NON_INTERACTIVE:-}" ]; then
  run_options+="--interactive --tty "
fi

docker run \
  --rm \
  --name "${TLS_HOSTNAME}" \
  ${run_options} \
  --mount type=bind,src="${TLS_KEYS_DIR}",dst=/keys \
  --volume ${peer_client_config}:/verification-v2/ts-pkgs/peer-client/self_config.json:ro \
  "dkg-client${TSS_E2E_GUARDIAN_ID:-}"


echo ""
echo "=============================================="
echo "DKG CEREMONY COMPLETED SUCCESSFULLY"
echo "=============================================="
echo ""
echo "Your DKG key shards have been generated and saved to ${TLS_KEYS_DIR}."
echo ""
echo "Please verify the generated files and securely back them up."
echo ""
