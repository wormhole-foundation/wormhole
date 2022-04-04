# Wormhole Contract Test Suite

## Running Local Terra Node

In order to run these tests, you need to have a local Terra node running. These tests are meant to be run using [LocalTerra](https://github.com/terra-money/LocalTerra). This requires [Docker Compose](https://docs.docker.com/compose/install/) to run. You can also run _terrad_ with the same set up Tilt uses (see configuration [here](../../devnet/terra-devnet.yaml)).

## Build

In the [terra root directory](../), run the following:
```sh
docker build -f Dockerfile.build -o artifacts .
```

## Run the Test Suite

First install dependencies:
```sh
npm ci
```

To run:
```sh
npm run test
```

These tests are built using Jest and is meant to be structured very similarly to the [ethereum unit tests](../../ethereum), which requires running a local node via ganache before _truffle_ can run any of the testing scripts in the [test directory](../../ethereum/test).

**Currently the only test that exists is for the token bridge's transfer with payload.**