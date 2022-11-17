FROM ghcr.io/wormhole-foundation/sui:0.15.0@sha256:ba5740e42ac41306719114ce96d9752a12bbfa500749ede64fa76a250113993b as sui-node
RUN sui genesis -f
COPY scripts/start_node.sh  /tmp
COPY scripts/funder.sh  /tmp
