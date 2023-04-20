FROM cli-gen AS cli-export
FROM const-gen AS const-export
FROM ghcr.io/wormhole-foundation/sui:1.0.0-testnet@sha256:63a8094590ddb90320aa1c86414f17cc73c759ecbdfaf2fe78f135b7c08ec536 as sui

RUN dnf -y install make git npm

COPY sui/README.md sui/cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi
RUN if [ -e /certs/cert.pem ]; then git config --global http.sslCAInfo /certs/cert.pem; fi

RUN sui genesis -f

COPY sui/devnet/ /root/.sui/sui_config/

COPY --from=cli-export ./clients/js/build/main.js /app/

WORKDIR /tmp

COPY sui/scripts/ scripts
COPY sui/wormhole/ wormhole
COPY sui/token_bridge/ token_bridge
COPY sui/examples/ examples
COPY sui/Makefile Makefile
COPY --from=const-export .env .

FROM sui AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache make test
