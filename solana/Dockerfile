# syntax=docker/dockerfile:experimental
FROM rust:1.45

RUN apt-get update && apt-get install -y libssl-dev libudev-dev pkg-config zlib1g-dev llvm clang
RUN rustup component add rustfmt

WORKDIR /usr/src/bridge

COPY bpf-sdk-install.sh .
COPY do.sh .

RUN ./do.sh update

COPY bridge bridge
COPY agent agent
COPY cli cli
COPY Cargo.toml .
COPY Cargo.lock .

RUN --mount=type=cache,target=/usr/local,from=rust,source=/usr/local \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=bridge/target \
    --mount=type=cache,target=target \
    --mount=type=cache,target=bin,from=rust,source=bin \
    ./do.sh build bridge && \
    mkdir -p /opt/solana/deps && \
    cp target/bpfel-unknown-unknown/release/spl_bridge_debug.so /opt/solana/deps/spl_bridge.so
