FROM ghcr.io/wormhole-foundation/sui:0.16.0@sha256:010a785908452f1901d1179209d508135e0eff663c02c191a71d1f2fd9cf46f6 as sui

RUN dnf -y install make git

WORKDIR /tmp

RUN sui genesis -f
COPY scripts/start_node.sh  .
COPY scripts/funder.sh  .

FROM sui AS tests

WORKDIR /tmp

RUN --mount=type=cache,target=/root/.move,id=move_cache \
    make test
