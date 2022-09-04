FROM ghcr.io/wormhole-foundation/aptos:0.3.3n@sha256:d321c61d6da379357e3e1e0765cd8a0c0d9d706cf6f34f8fb6e9f0abf55f04e6 as aptos

# Support additional root CAs
COPY cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

FROM aptos as tests

WORKDIR /tmp

COPY wormhole/ wormhole
COPY token_bridge/ token_bridge
COPY deployer/ deployer
COPY examples/ examples
COPY Makefile Makefile


RUN --mount=type=cache,target=/root/.move,id=move_cache \
    make test
