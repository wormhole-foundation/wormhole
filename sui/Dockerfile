FROM ghcr.io/wormhole-foundation/sui:0.32.0@sha256:a07bc5fccb81c56f27d38aa4422a16aafe71949f0dd8ed791908b68f22badab4 as sui

RUN dnf -y install make git npm

COPY sui/README.md sui/cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi
RUN if [ -e /certs/cert.pem ]; then git config --global http.sslCAInfo /certs/cert.pem; fi

RUN sui genesis -f

COPY sui/devnet/ /root/.sui/sui_config/

# Build CLI, TODO(aki): move this to base image before merging into main
RUN npm install -g n typescript ts-node
RUN n stable
COPY clients/js /tmp/clients/js
WORKDIR /tmp/clients/js
RUN make install

WORKDIR /tmp

COPY sui/scripts/ scripts
COPY sui/wormhole/ wormhole
COPY sui/token_bridge/ token_bridge
COPY sui/examples/ examples
COPY sui/Makefile Makefile
COPY sui/.env* .

FROM sui AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache make test
