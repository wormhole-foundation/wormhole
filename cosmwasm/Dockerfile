# This is a multi-stage docker file:
#  1. The first stage builds the contracts
#  2. The second is an empty image with only the wasm files (useful for exporting)
#  3. The third creates a node.js environment to deploy the contracts to devnet
FROM cosmwasm/workspace-optimizer:0.13.0@sha256:d868e239f73fb45ba98dd088c0a6a15effd0b87b7b193701f02c3913ecb8a196 AS builder

COPY cosmwasm/Cargo.lock /code/
COPY cosmwasm/Cargo.toml /code/
COPY cosmwasm/contracts /code/contracts
COPY cosmwasm/packages /code/packages
COPY cosmwasm/artifacts /code/artifacts
COPY sdk/rust /sdk/rust

RUN --mount=type=cache,target=/target,id=cosmwasm_target --mount=type=cache,target=/usr/local/cargo/registry optimize_workspace.sh

FROM scratch as artifacts
COPY --from=builder /code/artifacts /
