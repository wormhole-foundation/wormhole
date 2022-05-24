# Example Token Bridge UI

## Prerequisites

- Docker
- NodeJS v14+
- NPM v7.18+

Run the following from the root of this repo

```bash
DOCKER_BUILDKIT=1 docker build --target node-export -f Dockerfile.proto -o type=local,dest=. .
DOCKER_BUILDKIT=1 docker build -f solana/Dockerfile.wasm -o type=local,dest=. solana
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

## Build for local tilt network

```bash
npm run build
```

## Build for testnet

```bash
REACT_APP_CLUSTER=testnet npm run build
```

## Build for mainnet

```bash
REACT_APP_CLUSTER=mainnet REACT_APP_COVALENT_API_KEY=YOUR_API_KEY REACT_APP_SOLANA_API_URL=YOUR_CUSTOM_RPC npm run build
```

## Test Server

```bash
npx serve -s build
```

## Environment Variables (optional)

Create `.env` from the sample file, then add your Covalent API key:

```bash
cp .env.sample .env
```

## Run Project
```bash
npm i
npm start
```

### Custom Design And Text Changes Example on .env

REACT_APP_PRIMARY_COLOR="#2abfff"
REACT_APP_SECONDARY_COLOR="#ffffff12"
REACT_APP_BODY_COLOR="#16171b"
REACT_APP_TEXT_COLOR="#ffffff"
REACT_APP_LOGO="cloud image link"
REACT_APP_TITLE="Token Bridge"
REACT_APP_SUBTITLE="Token Bridge"
REACT_APP_LINK_NAME=""
REACT_APP_LINK_ADDRESS=""
