FROM ghcr.io/wormhole-foundation/sui:0.14.0@sha256:90252fb0856b59dae0e8e9ff06425d91a86f24bf0a13550ae410982a00530efb as sui-node
RUN sui genesis -f
COPY scripts/start_node.sh  /tmp
COPY scripts/funder.sh  /tmp
