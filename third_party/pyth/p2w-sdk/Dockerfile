FROM node:16-alpine@sha256:004dbac84fed48e20f9888a23e32fa7cf83c2995e174a78d41d9a9dd1e051a20

# Build ETH
WORKDIR /usr/src/ethereum
ADD ethereum .
RUN --mount=type=cache,target=/home/node/.npm \
  npm ci

# Build Wormhole SDK
WORKDIR /usr/src/sdk/js
ADD sdk/js/ .
RUN --mount=type=cache,target=/home/node/.npm \
  npm ci

# Build p2w-sdk
WORKDIR /usr/src/third_party/pyth/p2w-sdk
COPY third_party/pyth/p2w-sdk .
RUN --mount=type=cache,target=/root/.cache \
    --mount=type=cache,target=/root/.npm \
    npm ci && npm run build-test && npm run test
