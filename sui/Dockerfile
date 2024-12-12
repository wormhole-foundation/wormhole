FROM cli-gen AS cli-export
FROM const-gen AS const-export
FROM ghcr.io/wormhole-foundation/sui:1.19.1-mainnet@sha256:97e2e62d43262883b7a5bed5c9a9dd83d766a00a9476473eb275948fb53d376e AS sui

# initial run
# COPY sui/devnet/genesis_config genesis_config
# RUN sui genesis -f --from-config genesis_config

# subsequent runs after committing files from /root/.sui/sui_config/
COPY devnet/ /root/.sui/sui_config/

WORKDIR /tmp

COPY scripts/ scripts
COPY wormhole/ wormhole
COPY token_bridge/ token_bridge
COPY examples/ examples
COPY Makefile Makefile

# Copy .env and CLI
COPY --from=const-export .env .env
COPY --from=cli-export clients/js /cli

# Link `worm`
WORKDIR /cli

RUN npm link

FROM sui AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache make test
