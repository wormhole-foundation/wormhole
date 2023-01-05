FROM ghcr.io/wormhole-foundation/sui:0.20.0@sha256:793963650dcc76d8937f671213be53f39442cf0f74e5dd9697446fef4d093e9b as sui

RUN dnf -y install make git

COPY README.md cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi
RUN if [ -e /certs/cert.pem ]; then git config --global http.sslCAInfo /certs/cert.pem; fi

WORKDIR /tmp

RUN sui genesis -f
COPY scripts/start_node.sh  .
COPY scripts/funder.sh  .

COPY wormhole/ wormhole
COPY token_bridge/ token_bridge
# COPY examples/ examples
COPY Makefile Makefile

FROM sui AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache \
    make test
