FROM node:16-alpine@sha256:004dbac84fed48e20f9888a23e32fa7cf83c2995e174a78d41d9a9dd1e051a20

# npm needs a Python for some of the deps
RUN apk add git python3 make build-base

# Build ETH
WORKDIR /usr/src/ethereum
ADD ethereum .
RUN --mount=type=cache,target=/home/node/.npm \
  npm ci

# Build Wormhole SDK
WORKDIR /usr/src/sdk/js
ADD sdk/js/ .
RUN --mount=type=cache,target=/home/node/.npm \
  npm ci && npm run build

# Build p2w-sdk in dir preserving directory structure
WORKDIR /usr/src/third_party/pyth/p2w-sdk
COPY third_party/pyth/p2w-sdk/package.json third_party/pyth/p2w-sdk/package-lock.json .
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/root/.npm \
    npm ci

COPY third_party/pyth/p2w-sdk .
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/root/.npm \
    npm run build

# Build p2w-relay
WORKDIR /usr/src/third_party/pyth/p2w-relay
COPY third_party/pyth/p2w-relay/package.json third_party/pyth/p2w-relay/package-lock.json .
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/root/.npm \
    npm ci

COPY third_party/pyth/p2w-relay .
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/root/.npm \
    npm run build
