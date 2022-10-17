FROM ghcr.io/wormhole-foundation/aptos:0.4.0@sha256:19252a9d4f6d9f7e20074c3cfc98e6a81bea5736f8f8c3a4fb1bb09737a8e044 as aptos

# Support additional root CAs
COPY cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

WORKDIR /tmp

COPY wormhole/ wormhole
COPY token_bridge/ token_bridge
COPY deployer/ deployer
COPY coin/ coin
COPY examples/ examples
COPY Makefile Makefile
COPY scripts scripts
# In this step we optionally copy the .env file too
COPY start_node.sh .env* .

FROM aptos AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache \
    make test
