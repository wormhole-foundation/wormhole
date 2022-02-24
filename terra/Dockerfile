# This is a multi-stage docker file, first stage builds contracts
# And the second one creates node.js environment to deploy them
FROM cosmwasm/workspace-optimizer:0.12.1@sha256:1508cf7545f4b656ecafa34e29c1acf200cdab47fced85c2bc076c0c158b1338 AS builder
COPY Cargo.lock /code/
COPY Cargo.toml /code/
COPY contracts /code/contracts
COPY packages /code/packages
RUN optimize_workspace.sh

# Contract deployment stage
FROM node:16-buster-slim@sha256:93c9fc3550f5f7d159f282027228e90e3a7f8bf38544758024f005e82607f546

RUN apt update && apt install netcat curl jq -y

WORKDIR /app/tools

COPY --from=builder /code/artifacts /app/artifacts
COPY ./artifacts/cw20_base.wasm /app/artifacts/

COPY ./tools/package.json ./tools/package-lock.json /app/tools/
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
    npm ci
COPY ./tools /app/tools
