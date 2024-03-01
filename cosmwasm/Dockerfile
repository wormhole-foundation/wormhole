# This is a multi-stage docker file:
#  1. The first stage builds the contracts
#  2. The second is an empty image with only the wasm files (useful for exporting)
#  3. The third creates a node.js environment to deploy the contracts to devnet
FROM cosmwasm/workspace-optimizer:0.15.0@sha256:e1ba559282ea88cedeb67dd6dd541ad57d1b75c2fb1ce5f476d423458da91b4d AS builder

COPY cosmwasm/Cargo.lock /code/
COPY cosmwasm/Cargo.toml /code/
COPY cosmwasm/contracts /code/contracts
COPY cosmwasm/packages /code/packages
COPY cosmwasm/artifacts /code/artifacts
COPY sdk/rust /sdk/rust

RUN --mount=type=cache,target=/target,id=cosmwasm_target --mount=type=cache,target=/usr/local/cargo/registry optimize.sh .

FROM scratch as artifacts
COPY --from=builder /code/artifacts /
