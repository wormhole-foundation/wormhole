# Wormhole Core Shims

The intent of the following programs is to reduce the cost of Core Bridge
message emission and verification on Solana without making changes to the
existing core bridge.

- [Wormhole Post Message Shim]
- [Wormhole Verify VAA Shim]

The following are provided for example purposes only

- [Wormhole Integrator Example]

## Verifiable Build

The build commands require that [solana-verify] is installed on your machine. An
example of how to build for Solana mainnet:

```sh
NETWORK=mainnet SVM=solana make build-artifacts
```

This example command will result compiled verifiable programs in the
*artifacts-mainnet* directory. This command will not run if this directory
already exists.

## Tests

To perform unit, doc and integration tests, run:

```sh
make test
```

Integration tests are run using `cargo test-sbf`, but this requires having the
Solana toolchain installed via [agave-install].

Programs are built using Solana version 2.1.11, which is the current CLI
available at the time these programs were written.

**The `make` command above will initialize the Solana CLI version needed to
build and test. After running the tests, your CLI will still be configured to
this version. Please note your Solana CLI version before running this command.**

There are separate Anchor tests found in the [anchor directory].

For initial end-to-end (e2e) testing the post message shim with the guardian,
the programs were built with the following:

```sh
NETWORK=localnet SVM=solana make build
```

Please see the [anchor directory] to build the examples. The resulting program
binaries were then

- Copied to [../../solana/tests/artifacts]
- Copied into the test validator dockerfile in
  [../../solana/Dockerfile.test-validator]
- Loaded into the test validator at startup in [../../devnet/solana-devnet.yaml]

[../../devnet/solana-devnet.yaml]: ../../devnet/solana-devnet.yaml
[../../solana/Dockerfile.test-validator]: ../../solana/Dockerfile.test-validator
[../../solana/tests/artifacts]: ../../solana/tests/artifacts
[agave-install]: https://docs.anza.xyz/cli/install#use-the-solana-install-tool
[anchor directory]: anchor
[solana-verify]: https://solana.com/developers/guides/advanced/verified-builds
[Wormhole Integrator Example]: anchor/programs/wormhole-integrator-example/src/lib.rs
[Wormhole Post Message Shim]: programs/post-message/README.md
[Wormhole Verify VAA Shim]: programs/verify-vaa/README.md
