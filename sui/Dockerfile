FROM ghcr.io/wormhole-foundation/sui:0.25.0_stable@sha256:36924716d6d2d61fc948468ef0ebde801bb7c92bec37f4034fbdfe3593eaac34 as sui

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
