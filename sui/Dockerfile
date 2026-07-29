FROM cli-gen AS cli-export
FROM const-gen AS const-export
FROM ghcr.io/wormhole-foundation/sui:1.75.2-testnet@sha256:8c615701a008714bd1e9fcde40940c22083792a692494c6bc58739bb55a69efa AS sui

# Install git (needed by Sui CLI for package resolution)
RUN apt-get update && apt-get install -y git

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

RUN apt-get update && apt-get install -y make

RUN sui client new-env --alias testnet --rpc https://fullnode.testnet.sui.io:443 || true
RUN sui client switch --env testnet
RUN --mount=type=cache,target=/root/.move,id=move_cache make test
