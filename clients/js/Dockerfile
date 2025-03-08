# syntax=docker.io/docker/dockerfile:1.3@sha256:42399d4635eddd7a9b8a24be879d2f9a930d0ed040a61324cfdf59ef1357b3b2
FROM node:18-alpine@sha256:6eb9c3d9bd191bd2cc6ce7ec3d5ec4c2127616140c8586af96a6bec8f28689d1 as cli-build

RUN apk update && apk add g++ make python3

# Copy package.json & package-lock.json by themselves to create a cache layer
COPY package.json package-lock.json /clients/js/

WORKDIR /clients/js

RUN npm ci

# Copy the rest of the source files, as a layer on top of the deps
COPY . /clients/js

# Build CLI
RUN npm run build

FROM scratch AS cli-export

COPY --from=cli-build clients/js/build/main.js clients/js/build/main.js
COPY --from=cli-build clients/js/package.json clients/js/package.json
