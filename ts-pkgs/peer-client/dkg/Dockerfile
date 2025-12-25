FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN apt-get update && apt-get -y install git golang jq

# TODO: Pin the commit
RUN git clone -b feat/dkg-docker --depth 1 https://github.com/XLabs/core-bridge.git
RUN git clone -b schnorr --depth 1 https://github.com/XLabs/wormhole.git

WORKDIR /wormhole/node/pkg/tss/internal/cmd
RUN go build -o=./server ./dkg

WORKDIR /core-bridge/ts-pkgs/peer-client
RUN yarnpkg install --immutable
RUN yarnpkg build

COPY --chmod="ugo+rx" <<EOT poll_guardians.sh
#!/bin/bash

set -euo pipefail

if [ -z "\${TLS_HOSTNAME}" ]; then
  echo "TLS_HOSTNAME is not set"
  exit 1
fi
if [ -z "\${TLS_PORT}" ]; then
  echo "TLS_PORT is not set"
  exit 1
fi
if [ -z "\${PEER_SERVER_URL}" ]; then
  echo "PEER_SERVER_URL is not set"
  exit 1
fi
if [ -z "\${ETHEREUM_RPC_URL}" ]; then
  echo "ETHEREUM_RPC_URL is not set"
  exit 1
fi
if [ -z "\${WORMHOLE_CONTRACT_ADDRESS}" ]; then
  echo "WORMHOLE_CONTRACT_ADDRESS is not set"
  exit 1
fi

# This is the config for the peer client
cat <<EOF > self_config.json
{
  "serverUrl": "\${PEER_SERVER_URL}",
  "peer": {
    "hostname": "\${TLS_HOSTNAME}",
    "port": \${TLS_PORT},
    "tlsX509": "/keys/cert.pem"
  },
  "wormhole": {
    "ethereum": {
      "rpcUrl": "\${ETHEREUM_RPC_URL}",
      "chainId": 1
    },
    "wormholeContractAddress": "\${WORMHOLE_CONTRACT_ADDRESS}"
  }
}
EOF

yarnpkg start:client poll

# TODO: Change the SelfSecret to a file once we have the option in the dkg binary
GUARDIAN_TLS_KEY=$(cat /keys/key.pem | base64 -w 0)
jq ".StorageLocation = \\"/keys/\\" | .SelfSecret = \\"\${GUARDIAN_TLS_KEY}\\"" peer_config.json > /keys/dkg_config.json

/wormhole/node/pkg/tss/internal/cmd/server -cnfg=/keys/dkg_config.json

EOT

ENTRYPOINT ["./poll_guardians.sh"]