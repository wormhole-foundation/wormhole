FROM ghcr.io/wormhole-foundation/sui:0.29.1@sha256:8508a601746c886a87ef5eec59b4154dc19a228c52bbd256473e1be98e6ee4be as sui

RUN dnf -y install make git

COPY README.md cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi
RUN if [ -e /certs/cert.pem ]; then git config --global http.sslCAInfo /certs/cert.pem; fi

RUN sui genesis -f

COPY devnet/ /root/.sui/sui_config/

WORKDIR /tmp

COPY scripts/ scripts
COPY wormhole/ wormhole
COPY token_bridge/ token_bridge
COPY testing/ testing
COPY examples/ examples
COPY Makefile Makefile

FROM sui AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache make test
