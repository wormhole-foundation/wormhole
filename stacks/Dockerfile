FROM rust@sha256:f6d52726c6c740d6ee8ea9ff348db1cb4bfbe325d1e86f3295388666500481ec AS builder

# todo: change back to the following, after branch is merged (https://github.com/stacks-network/stacks-core/pull/6575)
# ARG STACKS_CORE_REPO=https://github.com/stacks-network/stacks-core.git
# ARG STACKS_CORE_BASE_BRANCH=develop

ARG STACKS_CORE_REPO=https://github.com/rdeioris/stacks-core.git
ARG STACKS_CORE_BASE_BRANCH=fix/block_replay_unsuccessful

RUN echo "Building Stacks Core from branch: $STACKS_CORE_BASE_BRANCH"

RUN git clone --branch $STACKS_CORE_BASE_BRANCH --single-branch --depth=1 $STACKS_CORE_REPO /code/stacks-core
WORKDIR /code/stacks-core
RUN apt-get update && apt-get install -y git libclang-dev llvm

RUN cargo build --features monitoring_prom,slog_json --bin stacks-node --bin stacks-signer

FROM debian@sha256:c05911a54043f172af10d458a22451755e4b66a99915b0e97c52099ffd2038ed AS stacks-node
RUN apt-get update \
    && apt-get install -y ca-certificates --no-install-recommends \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
RUN mkdir /data
COPY --from=builder /code/stacks-core/target/debug/stacks-node /usr/local/bin/stacks-node
STOPSIGNAL SIGTERM

FROM debian@sha256:c05911a54043f172af10d458a22451755e4b66a99915b0e97c52099ffd2038ed AS stacks-signer
RUN apt-get update \
    && apt-get install -y ca-certificates --no-install-recommends \
    && apt-get clean && rm -rf /var/lib/apt/lists/*
RUN mkdir /data
COPY --from=builder /code/stacks-core/target/debug/stacks-signer /usr/local/bin/stacks-signer
STOPSIGNAL SIGTERM
