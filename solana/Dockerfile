# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM docker.io/library/rust:1.58@sha256:e4979d36d5d30838126ea5ef05eb59c4c25ede7f064985e676feb21402d0661b

RUN apt-get update && apt-get install -y libssl-dev libudev-dev pkg-config zlib1g-dev llvm clang
RUN rustup component add rustfmt

WORKDIR /usr/src/bridge

RUN sh -c "$(curl -sSfL https://release.solana.com/v1.9.4/install)"

ENV PATH="/root/.local/share/solana/install/active_release/bin:$PATH"
ENV RUST_LOG="solana_runtime::system_instruction_processor=trace,solana_runtime::message_processor=trace,solana_bpf_loader=debug,solana_rbpf=debug"

# TODO: Solana build no longer works - instead, we cheat and copy the binary artifact

RUN mkdir -p /opt/solana/deps && \
	curl -Lo /opt/solana/deps/spl_bridge.so https://github.com/certusone/wormhole-networks/raw/master/mainnetv1/artifacts/solana/bridge.so

# COPY bridge bridge
# COPY agent agent
# COPY cli cli
# COPY Cargo.toml .
# COPY Cargo.lock .
#
# RUN --mount=type=cache,target=/usr/local,from=rust,source=/usr/local \
#     --mount=type=cache,target=/root/.cache \
#     --mount=type=cache,target=bridge/target \
#     --mount=type=cache,target=target \
#     --mount=type=cache,target=bin,from=rust,source=bin \
#     cargo build-bpf --manifest-path "bridge/Cargo.toml" && \
#     mkdir -p /opt/solana/deps && \
#     cp target/deploy/spl_bridge.so /opt/solana/deps/spl_bridge.so
