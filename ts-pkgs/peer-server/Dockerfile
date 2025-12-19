FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN apt-get update && apt-get -y install git

ARG SERVER_PORT
ARG ETHEREUM_RPC_URL
ARG WORMHOLE_ADDRESS=0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B

RUN test -n "${SERVER_PORT}" && \
    test -n "${ETHEREUM_RPC_URL}"

# TODO: Pin the commit
RUN git clone -b feat/dkg-docker --depth 1 https://github.com/XLabs/core-bridge.git
WORKDIR /core-bridge/ts-pkgs/peer-server
RUN yarnpkg install --immutable
RUN yarnpkg build
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
