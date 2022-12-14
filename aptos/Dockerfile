FROM ghcr.io/wormhole-foundation/aptos:1.1.0@sha256:9240135c950207e0bbe70d188999343b59d50d506dcc67c1b5586c1dd57b443c as aptos

# Support additional root CAs
COPY cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

WORKDIR /tmp

COPY wormhole/ wormhole
COPY token_bridge/ token_bridge
COPY nft_bridge/ nft_bridge
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
