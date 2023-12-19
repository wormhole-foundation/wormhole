FROM const-gen AS const-export
# This is a multi-stage docker file:
#  1. The first stage builds the contracts
#  2. The second is an empty image with only the wasm files (useful for exporting)
FROM cosmwasm/workspace-optimizer:0.15.0@sha256:e1ba559282ea88cedeb67dd6dd541ad57d1b75c2fb1ce5f476d423458da91b4d AS builder
COPY Cargo.lock /code/
COPY Cargo.toml /code/
COPY contracts /code/contracts

RUN --mount=type=cache,target=/target,id=terra_classic_target --mount=type=cache,target=/usr/local/cargo/registry optimize.sh .

FROM scratch as artifacts
COPY --from=builder /code/artifacts /

# Contract deployment stage
FROM node:16-buster-slim@sha256:93c9fc3550f5f7d159f282027228e90e3a7f8bf38544758024f005e82607f546

RUN apt update && apt install netcat curl jq -y

WORKDIR /app/tools

COPY --from=artifacts / /app/artifacts
COPY ./artifacts/cw20_base.wasm /app/artifacts/

COPY ./tools/package.json ./tools/package-lock.json /app/tools/
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
    npm ci
COPY ./tools /app/tools
COPY --from=const-export .env /app/tools/.env
