# Relayer - Ethereum Contracts

These smart contracts allow for relaying on EVM chains.

### Dependencies

The relayer contracts are built with Forge. See below for details on installing forge.

For the required version of `solc`, the required EVM version and the solidity library dependencies see the [config file](foundry.toml)

### Building

To build the contracts do:

```sh
wormhole/relayer/ethereum$ make build
```

### Deploying

For details on deploying the relayer contracts see the scripts [readme](ts-scripts/relayer/README.md).

### Testing

The tests for the relayer contracts reside in `forge-test`. To run the tests do:

```sh
wormhole/relayer/ethereum$ forge test
```

### Installing Forge

Some tests and scripts use [Foundry](https://getfoundry.sh/). It can be installed via the official installer, or by running

```sh
wormhole/relayer/ethereum$ ../../scripts/install-foundry
```

The installer script installs foundry and the appropriate solc version to build the contracts.
