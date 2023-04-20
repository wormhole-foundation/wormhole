FROM const-gen AS const-export
FROM ghcr.io/wormhole-foundation/aptos:1.0.4@sha256:0610c97ba32aab7113765b05a0c71d75b3842f7ee11ff8d7f75bd0ea03ad95b2 as aptos

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
COPY start_node.sh .
COPY --from=const-export .env .

FROM aptos AS tests

WORKDIR /tmp

RUN    make test
