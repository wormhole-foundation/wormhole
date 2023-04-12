FROM ghcr.io/wormhole-foundation/sui:0.31.0_1@sha256:5df513dc4fa0317f3068929811b4fa205d55d9378ca0e12721512486471bf6a9 as sui

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
COPY .env* .

FROM sui AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache make test
