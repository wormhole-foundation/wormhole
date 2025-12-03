# From https://github.com/wormhole-foundation/wormhole/blob/main/ethereum/Dockerfile
FROM node:22.16-bullseye-slim@sha256:550b434f7edc3a1875860657a3e306752358029c957280809ae6395ab296faeb as dkg

ARG TLS_HOSTNAME
ARG TLS_PORT
ARG TLS_PUBLIC_IP
ARG PEER_SERVER_URL

RUN test -n "${TLS_HOSTNAME}" && \
    test -n "${TLS_PORT}" && \
    test -n "${TLS_PUBLIC_IP}" && \
    test -n "${PEER_SERVER_URL}"

RUN apt-get update && apt-get -y install openssl git npm

RUN openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:prime256v1 -out key.pem
RUN openssl req -x509 -key key.pem -out cert.pem -days 365 \
  -subj "/CN=${TLS_HOSTNAME}" \
  -addext "subjectAltName=DNS:${TLS_HOSTNAME},IP:${TLS_PUBLIC_IP}" \
  -addext "keyUsage=digitalSignature" \
  -addext "extendedKeyUsage=serverAuth,clientAuth"

# TODO: Fix the commit
RUN git clone -b feat/peer-server --depth 1 https://github.com/XLabs/core-bridge.git
WORKDIR core-bridge/peer-server/
# This is the config for the peer client
COPY <<EOT self_config.json
{
  "guardianPrivateKeyPath": "/run/secrets/guardian_pk",
  "serverUrl": "${PEER_SERVER_URL}",
  "peer": {
    "hostname": "${TLS_HOSTNAME}",
    "port": ${TLS_PORT},
    "tlsX509": "/cert.pem"
  }
}
EOT
RUN npm ci
RUN npm run build
RUN --mount=type=secret,id=guardian_pk npm run start:client upload

FROM scratch

COPY --from=dkg /cert.pem /
