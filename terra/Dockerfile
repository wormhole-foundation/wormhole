# This is a multi-stage docker file, first stage builds contracts
# And the second one creates node.js environment to deploy them
FROM cosmwasm/workspace-optimizer:0.10.4@sha256:a976db4ee7add887a6af26724b804bbd9e9d534554506447e72ac57e65357db9 AS builder
ADD Cargo.lock /code/
ADD Cargo.toml /code/
ADD contracts /code/contracts
RUN optimize_workspace.sh

# Contract deployment stage
FROM node:16-buster-slim@sha256:93c9fc3550f5f7d159f282027228e90e3a7f8bf38544758024f005e82607f546

RUN apt update && apt install netcat curl jq -y

WORKDIR /app/tools

COPY --from=builder /code/artifacts /app/artifacts
ADD ./artifacts/cw20_base.wasm /app/artifacts/
ADD ./tools /app/tools

RUN chmod +x /app/tools/deploy.sh

ENTRYPOINT /app/tools/deploy.sh
