FROM ghcr.io/wormhole-foundation/sui:0.16.0@sha256:010a785908452f1901d1179209d508135e0eff663c02c191a71d1f2fd9cf46f6 as sui

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
