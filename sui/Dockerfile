FROM ghcr.io/wormhole-foundation/sui:0.29.0@sha256:e6da5712374e59c75a72f9b5393c81e5abe1867a09d5190f0d04bf365aaf67e5 as sui

RUN dnf -y install make git

COPY README.md cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi
RUN if [ -e /certs/cert.pem ]; then git config --global http.sslCAInfo /certs/cert.pem; fi

WORKDIR /tmp

RUN sui genesis -f
COPY scripts/start_node.sh  .
COPY scripts/import_default_account.sh .

COPY wormhole/ wormhole
COPY token_bridge/ token_bridge
COPY Makefile Makefile

RUN ./import_default_account.sh

FROM sui AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache \
    make test
