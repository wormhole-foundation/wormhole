# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM const-gen AS const-export
FROM --platform=linux/amd64 ghcr.io/foundry-rs/foundry:v1.0.0@sha256:d12a373ec950de170d5461014ef9320ba0fb6e0db6f87835999d0fcf3820370e as foundry
FROM --platform=linux/amd64 node:22.16-bullseye-slim@sha256:550b434f7edc3a1875860657a3e306752358029c957280809ae6395ab296faeb

RUN echo "Acquire::http::Pipeline-Depth 0;" >> /etc/apt/apt.conf.d/fixbadproxy
RUN echo "Acquire::http::No-Cache true;" >> /etc/apt/apt.conf.d/fixbadproxy
RUN echo "Acquire::BrokenProxy    true;" >> /etc/apt/apt.conf.d/fixbadproxy

# npm wants to clone random Git repositories - lovely.
# RUN apk add git python make build-base
# RUN apk update && apk add bash
RUN apt-get update && apt-get -y install \
  git python make curl netcat vim

RUN npm i typescript -g
RUN apt-get -y install jq

COPY --from=foundry /usr/local/bin/anvil /bin/anvil
COPY --from=foundry /usr/local/bin/forge /bin/forge

# Run as user, otherwise, npx explodes.
USER 1000


RUN mkdir -p /home/node/app
RUN mkdir -p /home/node/.npm

WORKDIR /home/node/app

# Fix git ssh error
RUN git config --global url."https://".insteadOf ssh://

WORKDIR /home/node/app/ethereum

# Only invalidate the npm install step if package.json changed
COPY --chown=node:node ethereum/package.json .
COPY --chown=node:node ethereum/package-lock.json .
COPY --from=const-export --chown=node:node .env.0x .env


# We want to cache node_modules *and* incorporate it into the final image.
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  --mount=type=cache,uid=1000,gid=1000,target=node_modules \
  npm ci && \
  cp -R node_modules node_modules_cache


# Amusingly, Debian's coreutils version has a bug where mv believes that
# the target is on a different fs and does a full recursive copy for what
# could be a renameat syscall. Alpine does not have this bug.
RUN rm -rf node_modules && mv node_modules_cache node_modules

ARG dev
ENV DEV=$dev

COPY --chown=node:node ethereum .
RUN make build

### RELAYER ###

WORKDIR /home/node/app/relayer/ethereum

COPY --chown=node:node relayer/ethereum/package.json .
COPY --chown=node:node relayer/ethereum/package-lock.json .
COPY --from=const-export --chown=node:node .env.0x .env


RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  --mount=type=cache,uid=1000,gid=1000,target=node_modules \
  npm ci && \
  cp -R node_modules node_modules_cache

RUN rm -rf node_modules && mv node_modules_cache node_modules

ARG dev
ENV DEV=$dev

COPY --chown=node:node relayer/ethereum .
RUN make build

ARG num_guardians
ENV NUM_GUARDIANS=$num_guardians
