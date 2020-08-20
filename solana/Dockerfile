# syntax=docker/dockerfile:experimental
FROM rust:1.45

RUN apt-get update && apt-get install -y libssl-dev libudev-dev pkg-config zlib1g-dev llvm clang
RUN rustup component add rustfmt

WORKDIR /usr/src/solana

COPY bpf-sdk-install.sh .
COPY do.sh .

RUN ./do.sh update

COPY bridge bridge
COPY agent agent
COPY cli cli
COPY Cargo.toml .
COPY Cargo.lock .

RUN ./do.sh build bridge

RUN --mount=type=cache,target=/usr/local/cargo,from=rust,source=/usr/local/cargo \
    --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=bridge/target \
    ./do.sh build bridge && \
    mkdir -p /opt/solana/deps && \
    cp target/bpfel-unknown-unknown/release/spl_bridge.so /opt/solana/deps
