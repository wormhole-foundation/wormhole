# syntax=docker.io/docker/dockerfile:experimental@sha256:de85b2f3a3e8a2f7fe48e8e84a65f6fdd5cd5183afa6412fff9caa6871649c44
FROM docker.io/library/rust:1.49@sha256:a50165ea96983c21832578afb1c8c028674c965bc1ed43b607871b1f362e06a5

RUN apt-get update && apt-get install -y libssl-dev libudev-dev pkg-config zlib1g-dev llvm clang
RUN rustup component add rustfmt

WORKDIR /usr/src/bridge

COPY bpf-sdk-install.sh .
COPY do.sh .

RUN ./do.sh update

COPY bridge bridge
COPY ee-vaa-program ee-vaa-program
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
    ./do.sh build ee-vaa-program && \
    mkdir -p /opt/solana/deps && \
    cp target/bpfel-unknown-unknown/release/spl_bridge_debug.so /opt/solana/deps/spl_bridge.so && \
    cp target/bpfel-unknown-unknown/release/ee_vaa_program_debug.so /opt/solana/deps/ee_vaa_program.so
