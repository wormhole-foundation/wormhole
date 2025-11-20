FROM rust@sha256:234b03300ecaae65919eb8cb13ca9b36588b8dff5b1c69581e361f9f25654db1 AS builder

ARG STACKS_CORE_REPO=https://github.com/stacks-network/stacks-core.git
ARG STACKS_CORE_BASE_BRANCH=develop

RUN echo "Building Stacks Core from branch: $STACKS_CORE_BASE_BRANCH"

RUN git clone --branch $STACKS_CORE_BASE_BRANCH --single-branch --depth=1 $STACKS_CORE_REPO /code/stacks-core
WORKDIR /code/stacks-core
RUN apt-get update && apt-get install -y git libclang-dev llvm

RUN cargo build --features monitoring_prom,slog_json --bin stacks-node --bin stacks-signer

FROM debian@sha256:936abff852736f951dab72d91a1b6337cf04217b2a77a5eaadc7c0f2f1ec1758 AS stacks-node
RUN apt-get update \
    && apt-get install -y ca-certificates --no-install-recommends \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
RUN mkdir /data
COPY --from=builder /code/stacks-core/target/debug/stacks-node /usr/local/bin/stacks-node
STOPSIGNAL SIGTERM

FROM debian@sha256:936abff852736f951dab72d91a1b6337cf04217b2a77a5eaadc7c0f2f1ec1758 AS stacks-signer
RUN apt-get update \
    && apt-get install -y ca-certificates --no-install-recommends \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
RUN mkdir /data
COPY --from=builder /code/stacks-core/target/debug/stacks-signer /usr/local/bin/stacks-signer
STOPSIGNAL SIGTERM
