FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN apt-get --quiet update && apt-get --quiet --no-install-recommends --yes install \
  git \
  golang \
  jq \
  ca-certificates \
  && rm -rf /var/lib/apt/lists

# TODO: Pin the commit
RUN git clone -b tss-server --depth 1 https://github.com/XLabs/tss-lib.git

WORKDIR /tss-lib/tss/internal/cmd
RUN go build -o=./server ./dkg

WORKDIR /
RUN mkdir --parents core-bridge/ts-pkgs/peer-client core-bridge/ts-pkgs/peer-lib
COPY .yarn core-bridge/.yarn
COPY package.json yarn.lock .yarnrc.yml core-bridge/
COPY ts-pkgs/peer-client/package.json core-bridge/ts-pkgs/peer-client/
COPY ts-pkgs/peer-lib/package.json core-bridge/ts-pkgs/peer-lib/
WORKDIR /core-bridge
RUN yarnpkg workspaces focus --all

COPY ts-pkgs/config/ ts-pkgs/config/
COPY ts-pkgs/peer-lib/tsconfig.json ts-pkgs/peer-lib/
COPY ts-pkgs/peer-lib/src ts-pkgs/peer-lib/src
COPY ts-pkgs/peer-client/tsconfig.json ts-pkgs/peer-client/
COPY ts-pkgs/peer-client/src ts-pkgs/peer-client/src
WORKDIR /core-bridge/ts-pkgs/peer-client
RUN yarnpkg build

COPY --chmod=555 <<EOT poll_guardians.sh
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
      "rpcUrl": "\${ETHEREUM_RPC_URL}"
    },
    "wormholeContractAddress": "\${WORMHOLE_CONTRACT_ADDRESS}"
  }
}
EOF

yarnpkg start:client poll

jq ".StorageLocation = \\"/keys/\\"" peer_config.json > /keys/dkg_config.json

/tss-lib/tss/internal/cmd/server -cnfg=/keys/dkg_config.json -protocol=FROST:DKG -sk=/keys/key.pem

EOT

ENTRYPOINT ["./poll_guardians.sh"]
