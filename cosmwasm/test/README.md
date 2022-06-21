# Wormhole Contract Test Suite

## Running Local Terra Node

In order to run these tests, you need to have a local Terra node running. These tests are meant to be run using [LocalTerra](https://github.com/terra-money/LocalTerra). This requires [Docker Compose](https://docs.docker.com/compose/install/) to run. You can also run _terrad_ with the same set up Tilt uses (see configuration [here](../../devnet/terra-devnet.yaml)).

## Build

In the [terra root directory](../), run the following:
```sh
make artifacts
```

## Run the Test Suite

The easy way would be to navigate to the [terra root directory](../), run the following:
```sh
make test
```

If you plan on adding new tests and plan on persisting LocalTerra, make sure dependencies are installed:
```sh
npm ci
```

And run in this directory:
```sh
npm run test
```

These tests are built using Jest and is meant to be structured very similarly to the [ethereum unit tests](../../ethereum), which requires running a local node via ganache before _truffle_ can run any of the testing scripts in the [test directory](../../ethereum/test).

**Currently the only test that exists is for the token bridge's transfer and transfer with payload.**
