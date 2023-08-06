# Solana SDK upgrade

This document describes the process of upgrading the Solana SDK dependency,
using the upgrade from `1.9.4` to `1.10.31` as a running example.

## rust-toolchain

The first step is to ensure that the rust toolchain version used to compile the
wormhole contracts matches the version that the solana SDK was tested against.
This is especially important since wormhole uses nightly toolchains, which are
particularly sensitive to changes between versions.

For `v1.10.31`, the version can be found at
https://github.com/solana-labs/solana/blob/v1.10.31/ci/rust-version.sh. We
update the `rust-toolchain` file to reflect the version there, in this case
`"nightly-2022-02-24"`.

## Dependencies

Once the rust toolchain is set, we can proceed with upgrading the solana sdk.
The process is somewhat interactive in that it might require upgrading multiple
dependencies until the project compiles (due to potential incompatibility with
the nightly compiler). Since the solana-sdk version is pinned in the
`Cargo.toml` files, we need to edit those files manually.

Running the following script will change the `1.9.4` version numbers to `1.10.31`.

```sh
find . -name "Cargo.toml" -exec sed -i 's/=1.9.4/=1.10.31/g' {} \;
```

As a sanity check, make sure that only the solana-sdk packages were changed in this way.

Now attempt to build the project by running

```sh
cargo build --locked
```

The `--locked` flag is important, as it will make sure to not change any of the
dependencies in the `Cargo.lock` file outside of the ones we just pinned to
newer versions.

In this particular case, the build fails because `borsh` is pinned to `0.9.1`,
but the new solana sdk needs `0.9.3`. Since we're upgrading a pinned library, we
must look at the changelog to ensure there's no inadvertent change we're
introducing to the code. `borsh` is a very popular library, so it is reasonable
to expect they follow semantic versioning properly, but it's still good practice to look at the changelog:
https://github.com/near/borsh-rs/blob/master/CHANGELOG.md

Once verified, we can upgrade borsh by running

```sh
find . -name "Cargo.toml" -exec sed -i 's/borsh = "=0.9.1"/borsh = "=0.9.3"/g' {} \;
```

running `cargo build --locked` again now fails with `spl-token` being out of date, which we can upgrade by running

```sh
find . -name "Cargo.toml" -exec sed -i 's/spl-token = { version = "=3.2.0"/spl-token = { version = "=3.3.0"/g' {} \;
```

Now the dependency versions are all fixed as far as cargo's dependency resolver
is concerned, but running `cargo build --locked` produces compilation errors
when building some upstream dependencies. In this case the culprit is that the
nightly compiler upgrade changed some feature guards so packages that used to
build fine with the old toolchain now fail to compile. Since these are transitive dependencies of the solana sdk, the best way to move forward is to simply upgrade or downgrade them to whatever version they're pinned at in solana:
https://github.com/solana-labs/solana/blob/v1.10.31/Cargo.lock

The two packages in question are `crossbeam-epoch` and `lock_api`, so we pin them to match solana's version by running

```sh
cargo update -p crossbeam-epoch --precise 0.9.5
cargo update -p lock_api --precise 0.4.6
```

With this, `cargo build --locked` finished cleanly, and the `Cargo.toml` files
along with `Cargo.lock` may now be checked in.

## cargo audit

As a sanity check, install `cargo-audit` by running

```sh
cargo install cargo-audit --locked
```

NOTE: you might want to install this outside of the repo using a stable compiler toolchain.

Then run

```sh
cargo audit
```

and investigate any potential vulnerabilities in the dependencies. In this case,
`cargo audit` found two issues, both are potential segfaults. Since these both
come from upstream dependencies through solana, it is reasonable to leave them
at those versions. This requires case by case assessment in general.
