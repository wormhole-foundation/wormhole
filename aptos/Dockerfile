FROM cli-gen AS cli-export
FROM const-gen AS const-export
FROM ghcr.io/wormhole-foundation/aptos:2.0.3@sha256:1e79c5a615b10073a780122abfaaf9b90735912df560b4e7b6a056fe45816392 as aptos

# Support additional root CAs
COPY cert.pem* /certs/
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

# Install nodejs
# todo(aki): move this into base image?
RUN apt-get update
RUN apt-get install -y ca-certificates curl gnupg
RUN mkdir -p /etc/apt/keyrings
RUN curl -fsSL https://deb.nodesource.com/gpgkey/nodesource-repo.gpg.key | gpg --dearmor -o /etc/apt/keyrings/nodesource.gpg

ARG NODE_MAJOR=18
RUN echo "deb [signed-by=/etc/apt/keyrings/nodesource.gpg] https://deb.nodesource.com/node_$NODE_MAJOR.x nodistro main" | tee /etc/apt/sources.list.d/nodesource.list

RUN apt-get update
RUN apt-get install nodejs -y

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
