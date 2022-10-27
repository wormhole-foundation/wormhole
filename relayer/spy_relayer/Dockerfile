# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2

FROM node:lts-alpine3.15@sha256:a2c7f8ebdec79619fba306cec38150db44a45b48380d09603d3602139c5a5f92 as base

RUN mkdir -p /app
WORKDIR /app

RUN apk add python3 \
    make \
    g++

COPY package.json package.json
COPY package-lock.json package-lock.json
COPY tsconfig.json tsconfig.json
COPY jestconfig.json jestconfig.json

FROM base AS builder

RUN --mount=type=cache,target=/root/.npm \
    npm ci

COPY src src
RUN npm run build && npm prune --production

FROM base AS application

LABEL org.opencontainers.image.source="https://github.com/wormhole-foundation/wormhole/tree/dev.v2/relayer/spy_relayer#readme"

COPY --from=builder /app/node_modules node_modules
COPY --from=builder /app/lib lib
COPY .env.tilt.listener /app/
COPY .env.tilt.relayer /app/
COPY .env.tilt.wallet-monitor /app/

CMD [ "node", "lib/main.js" ]

#TODO don't hardcode for tilt but accept env file
# RUN --mount=type=cache,uid=1000,gid=1000,target=/home/node/.npm \
#   npm run tilt_relay
