# Cosmwasm Wormhole Contracts

**NOTE**: _This process is only Linux host compatible at this time._

## Build Contracts

The following command can be used to build optimized cosmwasm contracts via Docker.

```console
wormhole/cosmwasm $ make artifacts
```

Upon completion, the compiled bytecode for cosmwasm contracts will be placed
into the `artifacts` directory.

## Run tests

You can run the cargo unit tests.

```console
wormhole/cosmwasm $ cargo test --workspace --locked
```
