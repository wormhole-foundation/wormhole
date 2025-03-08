# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM const-gen AS const-export
FROM ghcr.io/wormholelabs-xyz/foundry:nightly-55bf41564f605cae3ca4c95ac5d468b1f14447f9@sha256:50ffe8f3ba79274eaab85ce65e769e3d66b1c680f5a9acb9c25a393c4daeeeeb as foundry
FROM node:19.6.1-slim@sha256:e684615bdfb71cb676b3d0dfcc538c416f7254697d8f9639bd87255062fd1681

COPY --from=foundry /usr/local/bin/anvil /bin/anvil
COPY --from=foundry /usr/local/bin/forge /bin/forge

RUN apt-get update && apt-get -y install git python3 build-essential curl netcat vim jq

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
