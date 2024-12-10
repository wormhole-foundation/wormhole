FROM cli-gen AS cli-export
FROM const-gen AS const-export
FROM ghcr.io/wormhole-foundation/sui:1.19.1-mainnet@sha256:ad490b9c8c237b6b44d84feacfa45df3d0adeef55fcbec9586c0d9951e3dce95 AS sui

# initial run
# COPY sui/devnet/genesis_config genesis_config
# RUN sui genesis -f --from-config genesis_config

# subsequent runs after committing files from /root/.sui/sui_config/
COPY sui/devnet/ /root/.sui/sui_config/

WORKDIR /tmp

COPY sui/scripts/ scripts
COPY sui/wormhole/ wormhole
COPY sui/token_bridge/ token_bridge
COPY sui/examples/ examples
COPY sui/Makefile Makefile

# Copy .env and CLI
COPY --from=const-export .env .env
COPY --from=cli-export clients/js /cli

# Link `worm`
WORKDIR /cli

RUN npm link

FROM sui AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache make test
