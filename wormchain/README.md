# Wormchain

**Wormchain** is a blockchain built using Cosmos SDK and Tendermint and initially created with [Ignite](https://github.com/ignite).

## Building

Build and install wormchain.  You will need golang version 1.16+ installed.

```
make build/wormchaind
```

## Develop

See [development.md](./development.md)

## How to run the tests

    run "tilt up -- --wormchain"
    cd ./ts-sdk
    npm ci
    npm run build
    cd ../testing/js
    npm ci
    npm run test

## Learn more about Cosmos & Ignite

- [Ignite](https://github.com/ignite)
- [Ignite Docs](https://docs.ignite.com/)
- [Cosmos SDK documentation](https://docs.cosmos.network)
- [Cosmos SDK Tutorials](https://tutorials.cosmos.network)
- [Discord](https://discord.gg/cosmosnetwork)
