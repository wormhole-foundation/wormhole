FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

ARG SERVER_PORT
ARG ETHEREUM_RPC_URL
ARG WORMHOLE_ADDRESS=0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B

RUN mkdir --parents core-bridge/ts-pkgs/peer-server core-bridge/ts-pkgs/peer-lib
COPY --link .yarn core-bridge/.yarn
COPY --link package.json yarn.lock .yarnrc.yml core-bridge/
COPY --link ts-pkgs/peer-server/package.json core-bridge/ts-pkgs/peer-server/
COPY --link ts-pkgs/peer-lib/package.json core-bridge/ts-pkgs/peer-lib/
WORKDIR /core-bridge
RUN yarnpkg workspaces focus --all

COPY --link ts-pkgs/config/ ts-pkgs/config/
COPY --link ts-pkgs/peer-lib/tsconfig.json ts-pkgs/peer-lib/
COPY --link ts-pkgs/peer-lib/src ts-pkgs/peer-lib/src
COPY --link ts-pkgs/peer-server/tsconfig.json ts-pkgs/peer-server/
COPY --link ts-pkgs/peer-server/src ts-pkgs/peer-server/src

WORKDIR /core-bridge/ts-pkgs/peer-server
RUN yarnpkg build

RUN test -n "${SERVER_PORT}" && \
    test -n "${ETHEREUM_RPC_URL}"

# This is the config for the peer server
COPY <<EOT config.json
{
  "port": ${SERVER_PORT},
  "ethereum": {
    "rpcUrl": "${ETHEREUM_RPC_URL}",
    "chainId": 1
  },
  "wormholeContractAddress": "${WORMHOLE_ADDRESS}",
  "threshold": 13
}
EOT
ENTRYPOINT ["yarnpkg", "start:server"]
