#syntax=docker/dockerfile:1.2@sha256:e2a8561e419ab1ba6b2fe6cbdf49fd92b95912df1cf7d313c3e2230a333fdbcc
FROM ghcr.io/certusone/solana:1.9.4@sha256:5389eccba0ba59ae119bc1438b61be8904707e26df8f0732116a8dce0f64eb52 AS solana

# Support additional root CAs
COPY cert.pem* /certs/
# Debian
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/certs/ca-certificates.crt; fi

# Add bridge contract sources
WORKDIR /usr/src/bridge

COPY . .

ENV RUST_LOG="solana_runtime::system_instruction_processor=trace,solana_runtime::message_processor=trace,solana_bpf_loader=debug,solana_rbpf=debug"
ENV RUST_BACKTRACE=1

FROM solana AS builder

RUN mkdir -p /opt/solana/deps

ENV EMITTER_ADDRESS="11111111111111111111111111111115"
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
    cp modules/token_bridge/token-metadata/spl_token_metadata.so /opt/solana/deps/spl_token_metadata.so

# This stage is skipped in normal builds and needs to be explicitly invoked
# (like `DOCKER_BUILDKIT=1 docker build --target ci_tests .`).
FROM solana AS ci_tests

# This emitter address is necessary for the governance tests.
ENV EMITTER_ADDRESS="CiByUvEcx7w2HA4VHcPCBUAFQ73Won9kB36zW9VjirSr"

ARG BRIDGE_ADDRESS
RUN [ -n "${BRIDGE_ADDRESS}" ]

# This is a pre-built contract.
RUN --mount=type=cache,target=target,id=test \
    mkdir -p target/deploy && \
    cp modules/token_bridge/token-metadata/spl_token_metadata.so target/deploy/

RUN --mount=type=cache,target=target,id=test \
    cargo test-bpf \
        --manifest-path bridge/program/Cargo.toml \
        --features trace,instructions

RUN --mount=type=cache,target=target,id=test \
    cargo test-bpf \
        --manifest-path modules/token_bridge/program/Cargo.toml \
        --features trace,instructions

RUN --mount=type=cache,target=target,id=test \
    cargo test-bpf \
        --manifest-path modules/nft_bridge/program/Cargo.toml \
        --features trace,instructions

FROM scratch AS export-stage
COPY --from=builder /opt/solana/deps /

