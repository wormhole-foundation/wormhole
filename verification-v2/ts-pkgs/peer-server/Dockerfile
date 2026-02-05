FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

ARG SERVER_PORT
ARG ETHEREUM_RPC_URL
ARG WORMHOLE_ADDRESS=0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B

RUN mkdir --parents verification-v2/ts-pkgs/peer-server verification-v2/ts-pkgs/peer-lib
COPY .yarn /verification-v2/.yarn
COPY package.json yarn.lock .yarnrc.yml /verification-v2/
COPY ts-pkgs/peer-server/package.json /verification-v2/ts-pkgs/peer-server/
COPY ts-pkgs/peer-lib/package.json /verification-v2/ts-pkgs/peer-lib/
WORKDIR /verification-v2
RUN yarnpkg workspaces focus --all

COPY ts-pkgs/config/ ts-pkgs/config/
COPY ts-pkgs/peer-lib/tsconfig.json ts-pkgs/peer-lib/
COPY ts-pkgs/peer-lib/src ts-pkgs/peer-lib/src
COPY ts-pkgs/peer-server/tsconfig.json ts-pkgs/peer-server/
COPY ts-pkgs/peer-server/src ts-pkgs/peer-server/src

WORKDIR /verification-v2/ts-pkgs/peer-server
RUN yarnpkg build

RUN test -n "${SERVER_PORT}" && \
    test -n "${ETHEREUM_RPC_URL}"

# This is the config for the peer server
COPY <<EOT config.json
{
  "port": ${SERVER_PORT},
  "ethereum": {
    "rpcUrl": "${ETHEREUM_RPC_URL}"
  },
  "wormholeContractAddress": "${WORMHOLE_ADDRESS}",
  "threshold": 13,
  "peerListStore": "/peerGuardians.json"
}
EOT

ENTRYPOINT ["yarnpkg", "start:server"]
