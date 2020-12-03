# This is a multi-stage docker file, first stage builds contracts
# And the second one creates node.js environment to deploy them
FROM cosmwasm/workspace-optimizer:0.10.4 AS builder
ADD Cargo.lock /code/
ADD Cargo.toml /code/
ADD contracts /code/contracts
RUN optimize_workspace.sh

# Contract deployment stage
FROM node:14

RUN npm update && npm i -g typescript ts-node

WORKDIR /app/tools

COPY --from=builder /code/artifacts /app/artifacts
ADD ./artifacts/cw20_base.wasm /app/artifacts/
ADD ./tools /app/tools

RUN chmod +x /app/tools/deploy.sh

RUN npm install

RUN ts-node --version

ENTRYPOINT /app/tools/deploy.sh