# Develop

## prerequisites

- Go >= 1.16
- nodejs >= 16

## Building the blockchain

Run

```shell
make
```

This command creates a `build` directory and in particular, the
`build/wormchaind` binary, which can be used to run and interact with the
blockchain.

You can start a local development instance by running

```shell
make run
```

Or equivalently

```shell
./build/wormchaind --home build
```

If you want to reset the blockchain, just run

```shell
make clean
```

Then you can `make run` again.

## Running tests

Golang tests

    make test

Client tests, run against the chain. Wormchain must be running via `make run` or `tilt up -- --wormchain`

    cd ./ts-sdk
    npm ci
    npm run build
    cd ../testing/js
    npm ci
    npm run test

## Interacting with the blockchain

You can interact with the blockchain by using the go binary:

```shell
./build/wormchaind tx --from tiltGuardian --home build
```

Note the flags `--from tiltGuardian --home build`. These have to be passed
in each time you make a transaction (the `tiltGuardian` account is created in
`config.yml`). Queries don't need the `--from` flag.

## Scaffolding stuff with Ignite

Wormchain was initially scaffolded using the [Ignite CLI](https://github.com/ignite) (formerly Starport). Now, we only use Ignite for generating code from protobuf definitions.

To avoid system compatibility issues, we run Ignite using docker. The below commands should be run using the ignite docker container (see the Makefile recipes for examples).

```shell
ignite scaffold type guardian-key key:string --module wormhole --no-message
```

modify `proto/wormhole/guardian_key.proto` (string -> bytes)

```shell
ignite scaffold message register-account-as-guardian guardian-pubkey:GuardianKey address-bech32:string signature:string --desc "Register a guardian public key with a wormhole chain address." --module wormhole --signer signer
```

Scaffold a query:

```shell
ignite scaffold query latest_guardian_set_index --response LatestGuardianSetIndex --module wormhole
```

(then modify "wormchain/x/wormhole/types/query.pb.go" to change the response type)
