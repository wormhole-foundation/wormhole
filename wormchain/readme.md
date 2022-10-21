# Wormchain

**Wormchain** is a blockchain built using Cosmos SDK and Tendermint and created with [Starport](https://github.com/tendermint/starport).

## Building

We use [ignite](https://docs.ignite.com/) to build the protobuf.  Install the latest version we've pinned.

```
curl https://get.ignite.com/cli@v0.23.0 | bash
cp ignite ~/.local/bin/
```

Build the protobuf.

```
ignite generate proto-go
```

Build and install wormchain.

```
go install ./...
```

## Develop

See [development.md](./development.md)

## How to run the tests

    run either "starport chain serve" or "tilt up"
    cd ./ts-sdk
    npm ci
    npm run build
    cd ../testing/js
    npm ci
    npm run test

## Learn more about Cosmos & Starport

- [Starport](https://github.com/tendermint/starport)
- [Starport Docs](https://docs.starport.network)
- [Cosmos SDK documentation](https://docs.cosmos.network)
- [Cosmos SDK Tutorials](https://tutorials.cosmos.network)
- [Discord](https://discord.gg/cosmosnetwork)
