# Develop

## prerequsites

- Go >= 1.16
- Starport: `curl https://get.starport.network/starport@v0.19.5! | sudo bash
- nodejs >= 16

## Building the blockchain

Run

```shell
make
```

This command creates a `build` directory and in particular, the
`build/wormhole-chaind` binary, which can be used to run and interact with the
blockchain.

You can start a local development instance by running

```shell
make run
```

Or equivalently

```shell
./build/wormhole-chaind --home build
```

If you want to reset the blockchain, just run

```shell
make clean
```

Then you can `make run` again.

## Running tests

Golang tests

    make test

Client tests, run against the chain. Wormchain must be running via `starport chain serve`, `make run` or `tilt up`

    cd ./ts-sdk
    npm ci
    npm run build
    cd ../testing/js
    npm ci
    npm run test

## Interacting with the blockchain

You can interact with the blockchain by using the go binary:

```shell
./build/wormhole-chaind tx --from tiltGuardian --home build
```

Note the flags `--from tiltGuardian --home build`. These have to be passed
in each time you make a transaction (the `tiltGuardian` account is created in
`config.yml`). Queries don't need the `--from` flag.

## Scaffolding stuff with starport

TODO: expand explanation here

```shell
starport scaffold type guardian-key key:string --module wormhole --no-message
```

modify `proto/wormhole/guardian_key.proto` (string -> bytes)

```shell
starport scaffold message register-account-as-guardian guardian-pubkey:GuardianKey address-bech32:string signature:string --desc "Register a guardian public key with a wormhole chain address." --module wormhole --signer signer
```

Scaffold a query:

```shell
starport scaffold query latest_guardian_set_index --response LatestGuardianSetIndex --module wormhole
```

(then modify "wormhole_chain/x/wormhole/types/query.pb.go" to change the response type)
