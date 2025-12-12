# From https://github.com/wormhole-foundation/wormhole/blob/main/ethereum/Dockerfile
FROM node:22.16-bullseye-slim@sha256:550b434f7edc3a1875860657a3e306752358029c957280809ae6395ab296faeb as dkg

RUN apt-get update && apt-get -y install git golang

# TODO: Pin the commit
RUN git clone -b feat/dkg-docker --depth 1 https://github.com/XLabs/core-bridge.git
RUN git clone -b schnorr --depth 1 https://github.com/XLabs/wormhole.git

WORKDIR /core-bridge/ts-pkgs/peer-server
RUN yarnpkg install --immutable
RUN yarnpkg build

WORKDIR /wormhole/node/pkg/tss/internal/cmd
RUN go build -o=./server ./dkg

ARG TLS_HOSTNAME
ARG TLS_PORT
ARG TLS_PUBLIC_IP
ARG PEER_SERVER_URL

WORKDIR /core-bridge/ts-pkgs/peer-server

RUN test -n "${TLS_HOSTNAME}" && \
    test -n "${TLS_PORT}" && \
    test -n "${TLS_PUBLIC_IP}" && \
    test -n "${PEER_SERVER_URL}"

# This is the config for the peer client
COPY <<EOT self_config.json
{
  "guardianPrivateKeyPath": "/run/secrets/guardian_pk",
  "serverUrl": "${PEER_SERVER_URL}",
  "peer": {
    "hostname": "${TLS_HOSTNAME}",
    "port": ${TLS_PORT},
    "tlsX509": "/cert.pem"
  },
  "wormhole": {
    "ethereum": {
      "rpcUrl": "https://eth.llamarpc.com",
      "chainId": 1
    },
    "wormholeContractAddress": "0x98f3c9e6E3fAce36bAAd05FE09d375Ef1464288B"
  }
}
EOT

RUN --mount=type=secret,id=guardian_pk yarnpkg start:client poll

WORKDIR /wormhole/node/pkg/tss/internal/cmd
RUN ./server -cnfg=/core-bridge/ts-pkgs/peer-server/peer_config.json

FROM scratch

# TODO We probably need to use a bind mount instead
COPY --from=dkg /wormhole/node/pkg/tss/internal/cmd/secrets.json /