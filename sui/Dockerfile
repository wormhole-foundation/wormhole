FROM ghcr.io/wormhole-foundation/sui:0.12.2@sha256:85f7337604b5ee08a4ca5535efc80aa08ee97c55868c1cbf03f1b8a88e341cf1 as sui-node
RUN sui genesis -f
COPY scripts/start_node.sh  /tmp
COPY scripts/funder.sh  /tmp
