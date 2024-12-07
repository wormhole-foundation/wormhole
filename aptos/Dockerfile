FROM cli-gen AS cli-export
FROM const-gen AS const-export
FROM ghcr.io/wormhole-foundation/aptos:4.5.0@sha256:6d83c0ca3d434f06854ff16aa319db94b6d5857353b519e38f2096cfd0ab5df7 as aptos

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
