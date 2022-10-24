FROM ghcr.io/wormhole-foundation/aptos:0.4.0@sha256:37ceb6403c1f3ad2728ec0c71acb21a531c71d5156f37fc182f97940c982bfa7 as aptos

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
