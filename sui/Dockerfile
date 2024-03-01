FROM cli-gen AS cli-export
FROM const-gen AS const-export
FROM ghcr.io/wormhole-foundation/sui:1.19.1-mainnet@sha256:544a1b2aa5701fae25a19aed3c5e8c24e0caf7d1c9f511b6844d339a8f0b2a00 as sui

RUN sui genesis -f

# COPY sui/devnet/client.yaml /root/.sui/sui_config/
# COPY sui/devnet/sui.keystore /root/.sui/sui_config/

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
