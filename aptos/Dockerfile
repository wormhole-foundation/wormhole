FROM cli-gen AS cli-export
FROM const-gen AS const-export
FROM ghcr.io/wormhole-foundation/aptos:1.0.4_1@sha256:8d61205efab33e03375a146cdcf4838e2de4797ff2618b08a22cf74ec0843779 as aptos

# Support additional root CAs
COPY cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

# Install nodejs 
# todo(aki): move this into base image?
RUN curl -fsSL https://deb.nodesource.com/setup_18.x | bash - && apt-get update && apt-get install -y nodejs

WORKDIR /tmp

COPY wormhole/ wormhole
COPY token_bridge/ token_bridge
COPY nft_bridge/ nft_bridge
COPY deployer/ deployer
COPY coin/ coin
COPY examples/ examples
COPY Makefile Makefile
COPY scripts scripts
COPY start_node.sh start_node.sh
COPY --from=const-export .env .env
COPY --from=cli-export clients/js /cli

WORKDIR /cli

RUN npm link

FROM aptos AS tests

WORKDIR /tmp

RUN make test
