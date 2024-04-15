# Wormhole on Sui

This folder contains the reference implementation of the Wormhole cross-chain
messaging protocol smart contracts on the [Sui](https://mystenlabs.com/)
blockchain, implemented in the [Move](https://move-book.com/) programming
language.

# Project structure

The project is laid out as follows:

- [wormhole](./wormhole) the core messaging layer
- [token_bridge](./token_bridge) the asset transfer layer
- [coin](./coin) a template for creating Wormhole wrapped coins

# Installation

Make sure your Cargo version is at least 1.65.0 and then follow the steps below:

- https://docs.sui.io/build/install

#https://docs.sui.io/guides/developer/getting-started/sui-install# Prerequisites

Install the `Sui` CLI. This tool is used to compile the contracts and run the tests.

```sh
cargo install --locked --git https://github.com/MystenLabs/sui.git --rev 041c5f2bae2fe52079e44b70514333532d69f4e6 sui
```

Some useful Sui CLI commands are

- `sui start` to spin up a local network
- `rpc-server` to start a server for handling rpc calls

Next, install the [worm](../clients/js/README.md) CLI tool by running

```sh
wormhole/clients/js $ make install
```

`worm` is the swiss army knife for interacting with wormhole contracts on all
supported chains, and generating signed messages (VAAs) for testing.

As an optional, but recommended step, install the
[move-analyzer](https://github.com/move-language/move/tree/main/language/move-analyzer)
Language Server (LSP):

```sh
cargo install --git https://github.com/move-language/move.git move-analyzer --branch main --features "address32"
```

This installs the LSP backend which is then supported by most popular editors such as [emacs](https://github.com/emacs-lsp/lsp-mode), [vim](https://github.com/neoclide/coc.nvim), and even [vscode](https://marketplace.visualstudio.com/items?itemName=move.move-analyzer).

<details>
    <summary>For emacs, you may need to add the following to your config file:</summary>

```lisp
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

```sh
worm start-validator sui
```

which will start a local sui validator with an RPC endpoint at `0.0.0.0:9000`.

Once the validator is running, the contracts are ready to deploy. In the
[scripts](./scripts) directory, run

```sh
scripts $ ./deploy.sh devnet
```

This will deploy the core contract and the token bridge.

When you make a change to the contract, you can simply restart the validator and
run the deploy script again.

<!-- However, a better way is to run one of the following scripts:

``` sh
scripts $ ./upgrade devnet Core # for upgrading the wormhole contract
scripts $ ./upgrade devnet TokenBridge # for upgrading the token bridge contract
scripts $ ./upgrade devnet NFTBridge # for upgrading the NFT bridge contract
```

Behind the scenes, these scripts exercise the whole contract upgrade code path
(see below), including generating and verifying a signed governance action, and
the Move bytecode verifier checking ABI compatibility. If an upgrade here fails
due to incompatibility, it will likely on mainnet too. (TODO: add CI action to
simulate upgrades against main when there's a stable version) -->

# Implementation notes / coding practices

In this section, we describe some of the implementation design decisions and
coding practices we converged on along the way. Note that the coding guidelines
are prescriptive rather than descriptive, and the goal is for the contracts to
ultimately follow these, but they might not during earlier development phases.

### TODO
