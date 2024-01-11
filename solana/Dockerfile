#syntax=docker/dockerfile:1.2@sha256:e2a8561e419ab1ba6b2fe6cbdf49fd92b95912df1cf7d313c3e2230a333fdbcc
FROM ghcr.io/wormhole-foundation/solana:1.10.31@sha256:d31e8db926a1d3fbaa9d9211d9979023692614b7b64912651aba0383e8c01bad AS solana

# Add bridge contract sources
WORKDIR /usr/src/bridge

COPY bridge bridge
COPY modules modules
COPY migration migration
COPY Cargo.toml Cargo.toml
COPY Cargo.lock Cargo.lock
COPY solitaire solitaire
COPY external external

ENV RUST_LOG="solana_runtime::system_instruction_processor=trace,solana_runtime::message_processor=trace,solana_bpf_loader=debug,solana_rbpf=debug"
ENV RUST_BACKTRACE=1

FROM solana AS builder

RUN mkdir -p /opt/solana/deps

ARG EMITTER_ADDRESS="11111111111111111111111111111115"
ARG BRIDGE_ADDRESS
RUN [ -n "${BRIDGE_ADDRESS}" ]

# Build Wormhole Solana programs
RUN --mount=type=cache,target=target,id=build \
    --mount=type=cache,target=/usr/local/cargo/registry,id=cargo_registry \
    cargo build-bpf --manifest-path "bridge/program/Cargo.toml" -- --locked && \
    cargo build-bpf --manifest-path "bridge/cpi_poster/Cargo.toml" -- --locked && \
    cargo build-bpf --manifest-path "modules/token_bridge/program/Cargo.toml" -- --locked && \
    cargo build-bpf --manifest-path "modules/nft_bridge/program/Cargo.toml" -- --locked && \
    cargo build-bpf --manifest-path "migration/Cargo.toml" -- --locked && \
    cp target/deploy/bridge.so /opt/solana/deps/bridge.so && \
    cp target/deploy/cpi_poster.so /opt/solana/deps/cpi_poster.so && \
    cp target/deploy/wormhole_migration.so /opt/solana/deps/wormhole_migration.so && \
    cp target/deploy/token_bridge.so /opt/solana/deps/token_bridge.so && \
    cp target/deploy/nft_bridge.so /opt/solana/deps/nft_bridge.so && \
    cp external/mpl_token_metadata.so /opt/solana/deps/mpl_token_metadata.so

FROM scratch AS export-stage
COPY --from=builder /opt/solana/deps /

