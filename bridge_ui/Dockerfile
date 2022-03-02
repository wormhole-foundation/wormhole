# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2

# Derivative of ethereum/Dockerfile, look there for an explanation on how it works.
FROM node:16-alpine@sha256:f21f35732964a96306a84a8c4b5a829f6d3a0c5163237ff4b6b8b34f8d70064b

RUN mkdir -p /app
WORKDIR /app

COPY bridge_ui/package.json bridge_ui/package-lock.json ./
RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
    npm ci
COPY bridge_ui .
