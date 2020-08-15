# syntax=docker/dockerfile:experimental
FROM node:lts-alpine

# Run as user, otherwise, npx explodes.
USER 1000
RUN mkdir -p /home/node/app
RUN mkdir -p /home/node/.npm
WORKDIR /home/node/app

ADD --chown=node:node . .

# We want to cache node_modules *and* incorporate it into the final image.
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  --mount=type=cache,uid=1000,gid=1000,target=node_modules \
  npm install && \
  cp -r node_modules node_modules_cache

# Amusingly, Debian's coreutils version has a bug where mv believes that
# the target is on a different fs and does a full recursive copy for what
# could be a renameat syscall. Alpine does not have this bug.
RUN rmdir node_modules && mv node_modules_cache node_modules
