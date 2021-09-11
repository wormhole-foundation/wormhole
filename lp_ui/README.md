## Prerequisites

- Docker
- NodeJS v14+

Run the following from the root of this repo

```bash
DOCKER_BUILDKIT=1 docker build -- --target node-export -f Dockerfile.proto -o type=local,dest=. .
DOCKER_BUILDKIT=1 docker build -- -f Dockerfile.wasm -o type=local,dest=.. .
npm ci --prefix ethereum
npm ci --prefix sdk/js
npm run build --prefix sdk/js
```

The remaining steps can be run from this folder

## Install

```bash
npm ci
```

## Develop

```bash
npm start
```
