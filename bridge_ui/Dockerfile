# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2

# Derivative of ethereum/Dockerfile, look there for an explanation on how it works.
FROM node:lts-alpine@sha256:fa2f00d08281d522aeeeeeab6b158548c7d0c25fc3b8025579583c87c04c4c6b

RUN mkdir -p /app
WORKDIR /app

ADD . .

# these are only needed if you are using the local version of the sdk
# RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
#   npm ci --prefix ethereum
# RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
#   npm ci --prefix sdk/js
# RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
#   npm run build --prefix sdk/js


WORKDIR ./bridge_ui

RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm ci

RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm i serve

RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm run build

