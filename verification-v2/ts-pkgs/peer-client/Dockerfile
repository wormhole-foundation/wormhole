FROM node:22.21-trixie-slim@sha256:1ddaeddded05b2edeaf35fac720a18019e1044a6791509c8670c53c2308301bb

RUN mkdir --parents core-bridge/ts-pkgs/peer-client core-bridge/ts-pkgs/peer-lib
COPY --link .yarn /core-bridge/.yarn
COPY --link package.json yarn.lock .yarnrc.yml /core-bridge/
COPY --link ts-pkgs/peer-client/package.json /core-bridge/ts-pkgs/peer-client/
COPY --link ts-pkgs/peer-lib/package.json /core-bridge/ts-pkgs/peer-lib/
WORKDIR /core-bridge
RUN yarnpkg workspaces focus --all

COPY --link ts-pkgs/config/ ts-pkgs/config/
COPY --link ts-pkgs/peer-lib/tsconfig.json ts-pkgs/peer-lib/
COPY --link ts-pkgs/peer-lib/src ts-pkgs/peer-lib/src
COPY --link ts-pkgs/peer-client/tsconfig.json ts-pkgs/peer-client/
COPY --link ts-pkgs/peer-client/src ts-pkgs/peer-client/src
WORKDIR /core-bridge/ts-pkgs/peer-client
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