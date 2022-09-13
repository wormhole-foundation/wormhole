FROM ghcr.io/wormhole-foundation/aptos:0.3.4@sha256:999c4e59e04dbfdc127c679a649019cbdeb665880299a47bf716a40d4457d5a4 as aptos

# Support additional root CAs
COPY cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

FROM aptos AS tests

WORKDIR /tmp

COPY wormhole/ wormhole
COPY token_bridge/ token_bridge
COPY deployer/ deployer
COPY coin/ coin
COPY examples/ examples
COPY Makefile Makefile


RUN --mount=type=cache,target=/root/.move,id=move_cache \
    make test
