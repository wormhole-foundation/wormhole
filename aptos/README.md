# Wormhole on Aptos

This folder contains the reference implementation of the Wormhole cross-chain
messaging protocol smart contracts on the [Aptos](https://aptoslabs.com/)
blockchain, implemented in the [Move](https://move-book.com/) programming
language.

# Project structure

The project is laid out as follows:

- [wormhole](./wormhole) the core messaging layer
- [token_bridge](./token_bridge) the asset transfer layer
- [nft_bridge](./nft_bridge) NFT transfer layer
- [examples](./examples) various example contracts

To see a minimal example of how to integrate with wormhole, check out
[sender.move](./examples/core_messages/sources/sender.move).

# Hacking

The project is under active development, and the development workflow is in
constant flux, so these steps are subject to change.

## Prerequisites

Install the `aptos` CLI. This tool is used to compile the contracts and run the tests.

``` sh
$ git clone https://github.com/aptos-labs/aptos-core.git
$ cd aptos-core
aptos-core $ cargo build --package aptos --release
aptos-core $ mv target/release/aptos ~/.cargo/bin/aptos # move the binary to somewhere on your PATH
```


Next, install the [worm](../clients/js/README.md) CLI tool by running

``` sh
wormhole/clients/js $ make install
```


`worm` is the swiss army knife for interacting with wormhole contracts on all
supported chains, and generating signed messages (VAAs) for testing.

As an optional, but recommended step, install the
[move-analyzer](https://github.com/move-language/move/tree/main/language/move-analyzer)
Language Server (LSP):

``` sh
cargo install --git https://github.com/move-language/move.git move-analyzer --branch main --features "address32"
```

Note the `--features "address32"` flag. This is important, because the default
build only supports 16 byte addresses, but Aptos uses 32 bytes, so
`move-analyzer` needs to be built with that feature flag to support 32 byte
address literals.

This installs the LSP backend which is then supported by most popular editors such as [emacs](https://github.com/emacs-lsp/lsp-mode), [vim](https://github.com/neoclide/coc.nvim), and even [vscode](https://marketplace.visualstudio.com/items?itemName=move.move-analyzer).

<details>
    <summary>For emacs, you may need to add the following to your config file:</summary>

``` lisp
;; Move
(define-derived-mode move-mode rust-mode "Move"
  :group 'move-mode)

(add-to-list 'auto-mode-alist '("\\.move\\'" . move-mode))

(with-eval-after-load 'lsp-mode
  (add-to-list 'lsp-language-id-configuration
    '(move-mode . "move"))

  (lsp-register-client
    (make-lsp-client :new-connection (lsp-stdio-connection "move-analyzer")
                     :activation-fn (lsp-activate-on "move")
                     :server-id 'move-analyzer)))
```

</details>

## Building & running tests

The project uses a simple `make`-based build system for building and running
tests. Running `make test` in this directory will run the tests for each
contract. If you only want to run the tests for, say, the token bridge contract,
then you can run `make test` in the `token_bridge` directory, or run `make -C
token_bridge test` from this directory.

Additionally, `make test-docker` runs the tests in a docker container which is
set up with all the necessary dependencies. This is the command that runs in CI.

## Running a local validator and deploying the contracts to it

Simply run

``` sh
worm aptos start-validator
```

which will start a local aptos validator with an RPC endpoint at `0.0.0.0:8080`
and the faucet endpoint at `0.0.0.0:8081`. Note that the faucet takes a few
(~10) seconds to come up, so only proceed when you see the following:

``` text
Faucet is running.  Faucet endpoint: 0.0.0.0:8081
```


Once the validator is running, the contracts are ready to deploy. In the
[scripts](./scripts) directory, run

``` sh
scripts $ ./deploy devnet
```

This will deploy the core contract, the token bridge, and an example contract
for sending messages through wormhole.

When you make a change to the contract, you can simply restart the validator and
run the deploy script again. However, a better way is to run one of the following scripts:

``` sh
scripts $ ./upgrade devnet Core # for upgrading the wormhole contract
scripts $ ./upgrade devnet TokenBridge # for upgrading the token bridge contract
scripts $ ./upgrade devnet NFTBridge # for upgrading the NFT bridge contract
```

Behind the scenes, these scripts exercise the whole contract upgrade code path
(see below), including generating and verifying a signed governance action, and
the Move bytecode verifier checking ABI compatibility. If an upgrade here fails
due to incompatibility, it will likely on mainnet too. (TODO: add CI action to
simulate upgrades against main when there's a stable version)

# Implementation notes / coding practices

In this section, we describe some of the implementation design decisions and
coding practices we converged on along the way. Note that the coding guidelines
are prescriptive rather than descriptive, and the goal is for the contracts to
ultimately follow these, but they might not during earlier development phases.

## Signers

In Move, each entry point function may take an optional first argument of type
`signer` or `&signer`, as follows

``` rust
public entry fun foo(user: &signer) {
  // do stuff
}
```

When a user signs a transaction that calls `foo`, their wallet will effectively
be passed in as a `signer` to `foo`. This `signer` value can then be used to
authorise *arbitrary* actions on behalf of the user, such as withdrawing their
coins:

``` rust
use aptos_framework::coin;
use aptos_framework::aptos_coin::AptosCoin;

public entry fun foo(user: &signer) {
    let coins = coin::withdraw<AptosCoin>(user, 100);

    // ...
}
```

The `user` value can even be passed on to other functions down the call stack,
so the user has to fully trust `foo` and potentially understand its
implementation to be sure that the transaction is safe to sign. Since the
`signer` object can be passed arbitrarily deep into the call stack, tracing the
exact path is onerous. This hurts composability too, because composing contracts
that take `signer`s now places additional burden on each caller to ensure that
the callee contract is non-malicious and trust that it won't turn malicious in
the future through an upgrade. Thus we consider taking `signer` arguments an
*anti-pattern*, and avoid it wherever possible.

Here, `foo` requires the user's `signer` to be able to withdraw 100 aptos coins.
A clearer and safer way to achieve this is by writing `foo` in the following way:

``` rust
use aptos_framework::coin::{Self, Coin};
use aptos_framework::aptos_coin::AptosCoin;

public fun foo(coins: Coin<AptosCoin>) {
    assert!(coin::value(&coins) == 100, SOME_ERROR_CODE);

    // ...
}
```

Just the type of this version itself makes it extremely clear what `foo` really
needs, and before calling this function the caller can just withdraw their coins
themselves. As a convenience function, we may introduce a wrapper that _does_
take a signer:

``` rust
public entry fun foo_with_signer(user: &signer) {
    foo(coin::withdraw(user, 100))
}
```

which might be the preferred version for EOAs (externally owned accounts, aka
user wallets), but never for other contracts.

The general rule of thumb is that a function that takes a `signer` should never
pass that `signer` on to another function (except standard library functions
like `coin::withdraw`). This way, deciding the safety of a function becomes much
simpler for users and integrators.

## Access control: fine-grained capabilities

`signer` objects can also be used for access control, because the existence of a
`signer` value with a given address proves that the address authorised that
transaction. The key observation is that a `signer` is an unforgeable token of
authority, also known as a _capability_. The issue is, as described in the above
section, is that the `signer` capability is too general, as it can authorise
arbitrary actions on behalf of the user. For this reason, we don't use `signer`s
to implement access control, and instead turn to more fine-grained capabilities.

Thanks to Move's module system and linear type system, it is possible to
implement first-class capabilities, i.e. non-forgeable objects of authority. For
example, when sending a message, the wormhole contract needs to record the
identity of the message sender in a way that cannot be forged by malicious
actors. A potential solution would be to simply take the sender's `signer`
object and encode its address into the message:

``` rust
public fun publish_message(
    sender: &signer,
    nonce: u64,
    payload: vector<u8>,
    message_fee: Coin<AptosCoin>
): u64 {
// ...
}
```

However, again, this is not a great solution because the sender now needs to
fully trust `publish_message`. Instead, we define a capability called
`EmitterCapability` and require that instead:

``` rust
public fun publish_message(
    emitter_cap: &mut emitter::EmitterCapability,
    nonce: u64,
    payload: vector<u8>,
    message_fee: Coin<AptosCoin>
): u64 {
```

The `EmitterCapability` type is defined in
[emitter.move](./wormhole/sources/emitter.move)
as

``` rust
struct EmitterCapability has store {
    emitter: u128,
    sequence: u64
}
```

note that it has no `drop` or `copy` abilities, only `store`, which means that
once created, the object cannot be destroyed or copied, but it can be stored in
the storage space of a smart contract. Before being able to send messages
through wormhole, integrators must obtain such an `EmitterCapability` by calling

``` rust
public fun register_emitter(): emitter::EmitterCapability
```

in `./wormhole/sources/wormhole.move`. Note that this function does not take any
arguments (in particular no signer), and returns an `EmitterCapability`. The
contract can then store this and use as a unique identifier in the future when
sending messages through wormhole. Since the wormhole contract is the only
entity that can create new `EmitterCapability` objects (protected by a similar
capability mechanism, see the [emitter.move](./wormhole/source/emitter.move)
module for more details), it can guarantee that the `emitter` field is globally
unique for each new emitter.

An important safety property of Move is that structs (like `EmitterCapability`)
are fully opaque outside of the module that defines them. This means that
there's no way to introspect, modify, or transfer them outside of the defining
module.  In turn, the defining module may choose to expose an API that provides
restricted access to the contents. For example,
[emitter.move](./wormhole/source/emitter.move) defines a getter function for the
`emitter` field:

``` rust
public fun get_emitter(emitter_cap: &EmitterCapability): u128 {
    emitter_cap.emitter
}
```

notice the `emitter_cap.emitter` field accessor syntax, which is only legal in
the defining module of the struct. The only way to access the `sequence` field
is the following function:

``` rust
public(friend) fun use_sequence(emitter_cap: &mut EmitterCapability): u64 {
    let sequence = emitter_cap.sequence;
    emitter_cap.sequence = sequence + 1;
    sequence
}
```

That is, the emitter capability's sequence counter can only be incremented
outside of this module, but not modified arbitrarily. As a further security
measure, this function is marked as `public(friend)`, which means it's only
accessible from modules that are declared as a "friend" of the `emitter` module.

In practice, the `public_message` function will call this function to get and
increment the sequence number each time a message is sent.

The fact that the caller can produce a reference to an `EmitterCapability` is
proof that either they have direct access to the storage that owns it, or they
have been passed the reference from the actual owner. This pattern enables
better composability: `EmitterCapability` objects can be transferred in case a
non-upgradeable contract wants to migrate to a new version but still be able to
reuse the same wormhole emitter identity. They can also be passed by reference
down the callstack (through borrowing), which makes it possible for a contract
to send a message on behalf of another contract *with explicit permission*,
since the caller contract needs to pass in the reference. It is also possible
for a single application to have multiple emitter identities at the same time,
which uncovers new use cases that have not been easily possible in other chains.

## Contract deployment/upgrades

In order to support security patches and new features, the wormhole contracts
implement upgradeability through governance. For this to be possible without one
single entity having full control over the contracts, the contracts need to be
their own signing authority. For details on how this is implemented, see
[deployer.move](./deployer/sources/deployer.move) and
[contract_upgrade.move](./wormhole/sources/contract_upgrade.move).
