# This is a multi-stage docker file, first stage builds contracts
# And the second one creates node.js environment to deploy them
FROM cosmwasm/workspace-optimizer:0.10.4@sha256:a976db4ee7add887a6af26724b804bbd9e9d534554506447e72ac57e65357db9 AS builder
ADD Cargo.lock /code/
ADD Cargo.toml /code/
ADD contracts /code/contracts
RUN optimize_workspace.sh

# Contract deployment stage
FROM python:slim-buster@sha256:ec7a755e6313da2f7db02d8e82f6b0813b176f06c5622174c8ab45feefc8096d

RUN apt update && apt install netcat curl jq -y

WORKDIR /app/tools

COPY --from=builder /code/artifacts /app/artifacts
ADD ./artifacts/cw20_base.wasm /app/artifacts/
ADD ./tools /app/tools

RUN chmod +x /app/tools/deploy.sh

RUN pip install -r /app/tools/requirements.txt

ENTRYPOINT /app/tools/deploy.sh
