# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
# Devnet image for the Stellar integration, built by Tilt as `stellar-deploy`.
# Builds the core and devnet-emitter contracts to wasm and packages them with the stellar
# CLI, the const-gen devnet constants and the devnet deploy / smoke-test scripts.
FROM const-gen AS const-export

FROM rust:1.89.0-slim-bookworm@sha256:d7fc7de78bb8c1469933aeecbf801314d30d7d6e9f0578bba4cfa285bfa37fe6 AS build
# The image toolchain matches rust-toolchain.toml (1.89.0). rustup adds the wasm target here and
# fetches the rustfmt/clippy components the toolchain file lists on first use.
RUN rustup target add wasm32v1-none
WORKDIR /src
COPY . .
RUN cargo build --locked --release --target wasm32v1-none -p wormhole-contract -p devnet-emitter

# Keep the CLI major version equal to --protocol-version in devnet/stellar-devnet.yaml: the CLI
# and the quickstart's stellar-rpc must agree on the protocol generation for simulation.
FROM stellar/stellar-cli:28.0.0@sha256:bc1073aecfc5dd0974b76331162c7fff09b7793cd8aaa79b3a1d918363937f05
# The CLI image switches to `USER stellar`; the devnet image runs as root so apt works and the
# scripts can write /app/success.
USER root
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl jq && \
    rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /src/target/wasm32v1-none/release/wormhole_contract.wasm \
                  /src/target/wasm32v1-none/release/devnet_emitter.wasm ./
COPY --from=const-export .env devnet-consts.json ./
COPY scripts/devnet_deploy.sh scripts/devnet_smoke_test.sh ./
RUN chmod +x devnet_deploy.sh devnet_smoke_test.sh
# The CLI image's entrypoint is `stellar`; the k8s manifests run the scripts instead.
ENTRYPOINT []
CMD ["/bin/bash"]
