FROM const-gen AS const-export
# This is a multi-stage docker file:
#  1. The first stage builds the contracts
#  2. The second is an empty image with only the wasm files (useful for exporting)
FROM cosmwasm/workspace-optimizer:0.12.6@sha256:e6565a5e87c830ef3e8775a9035006b38ad0aaf0a96319158c802457b1dd1d08 AS builder
COPY Cargo.lock /code/
COPY Cargo.toml /code/
COPY contracts /code/contracts

# Support additional root CAs
COPY README.md cert.pem* /certs/
# Alpine
RUN if [ -e /certs/cert.pem ]; then cp /certs/cert.pem /etc/ssl/cert.pem; fi

RUN --mount=type=cache,target=/code/target,id=terra_classic_target --mount=type=cache,target=/usr/local/cargo/registry optimize_workspace.sh

FROM scratch as artifacts
COPY --from=builder /code/artifacts /

# Contract deployment stage
FROM node:16-buster-slim@sha256:93c9fc3550f5f7d159f282027228e90e3a7f8bf38544758024f005e82607f546

# Support additional root CAs
COPY README.md cert.pem* /certs/
# Node
ENV NODE_EXTRA_CA_CERTS=/certs/cert.pem
ENV NODE_OPTIONS=--use-openssl-ca
# npm
RUN if [ -e /certs/cert.pem ]; then npm config set cafile /certs/cert.pem; fi

RUN apt update && apt install netcat curl jq -y

WORKDIR /app/tools

COPY --from=artifacts / /app/artifacts
COPY ./artifacts/cw20_base.wasm /app/artifacts/

COPY ./tools/package.json ./tools/package-lock.json /app/tools/
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
    npm ci
COPY ./tools /app/tools
COPY --from=const-export .env /app/tools/.env
