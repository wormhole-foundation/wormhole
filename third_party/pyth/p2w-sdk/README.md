# Pyth2wormhole SDK
This project contains a library for interacting with pyth2wormhole and adjacent APIs.

# Install
For now, the in-house dependencies are referenced by relative
path. The commands below will build those. For an automated version of
this process, please refer to `p2w-relay`'s Dockerfile and/or our [Tilt](https://tilt.dev)
devnet with `pyth` enabled.

```shell
# Run the commands in this README's directory for --prefix to work
$ npm --prefix ../../../ethereum ci && npm --prefix ../../../ethereum run build # ETH contracts
$ npm --prefix ../../../sdk/js ci # Wormhole SDK
$ npm ci && npm run build # Pyth2wormhole SDK
```
