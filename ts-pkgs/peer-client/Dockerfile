FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN apt-get update && apt-get -y install git

# TODO: Pin the commit
RUN git clone -b feat/dkg-docker --depth 1 https://github.com/XLabs/core-bridge.git
WORKDIR /core-bridge/ts-pkgs/peer-client
RUN yarnpkg install --immutable
RUN yarnpkg build

ARG TLS_HOSTNAME
ARG TLS_PORT
ARG PEER_SERVER_URL

RUN test -n "${TLS_HOSTNAME}" && \
    test -n "${TLS_PORT}" && \
    test -n "${PEER_SERVER_URL}"

# This is the config for the peer client
COPY <<EOT self_config.json
{
  "guardianPrivateKeyPath": "/run/secrets/guardian_pk",
  "serverUrl": "${PEER_SERVER_URL}",
  "peer": {
    "hostname": "${TLS_HOSTNAME}",
    "port": ${TLS_PORT},
    "tlsX509": "/run/secrets/cert.pem"
  }
}
EOT

# The cert is not really a secret, but it doesn't hurt to use the same mount type for both
RUN --mount=type=secret,id=guardian_pk --mount=type=secret,id=cert.pem yarnpkg start:client upload