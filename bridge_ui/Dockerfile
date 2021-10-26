# syntax=docker.io/docker/dockerfile:experimental@sha256:de85b2f3a3e8a2f7fe48e8e84a65f6fdd5cd5183afa6412fff9caa6871649c44

# Derivative of ethereum/Dockerfile, look there for an explanation on how it works.
FROM node:16-alpine@sha256:004dbac84fed48e20f9888a23e32fa7cf83c2995e174a78d41d9a9dd1e051a20

RUN mkdir -p /app
WORKDIR /app

ADD . .

RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm ci --prefix ethereum
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm ci --prefix sdk/js
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm run build --prefix sdk/js


WORKDIR ./bridge_ui

RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm ci

RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm i serve

RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
  npm run build

