# Cw721 Basic

This is a basic implementation of a cw721 NFT contract. It implements
the [CW721 spec](../../packages/cw721/README.md) and is designed to
be deployed as is, or imported into other contracts to easily build
cw721-compatible NFTs with custom logic.

Implements:

- [x] CW721 Base
- [x] Metadata extension
- [ ] Enumerable extension (AllTokens done, but not Tokens - requires [#81](https://github.com/CosmWasm/cw-plus/issues/81))

## Implementation

The `ExecuteMsg` and `QueryMsg` implementations follow the [CW721 spec](../../packages/cw721/README.md) and are described there.
Beyond that, we make a few additions:

* `InstantiateMsg` takes name and symbol (for metadata), as well as a **Minter** address. This is a special address that has full 
power to mint new NFTs (but not modify existing ones)
* `ExecuteMsg::Mint{token_id, owner, token_uri}` - creates a new token with given owner and (optional) metadata. It can only be called by
the Minter set in `instantiate`.
* `QueryMsg::Minter{}` - returns the minter address for this contract.

It requires all tokens to have defined metadata in the standard format (with no extensions). For generic NFTs this may
often be enough.

The *Minter* can either be an external actor (eg. web server, using PubKey) or another contract. If you just want to customize
the minting behavior but not other functionality, you could extend this contract (importing code and wiring it together)
or just create a custom contract as the owner and use that contract to Mint.

If provided, it is expected that the _token_uri_ points to a JSON file following the [ERC721 Metadata JSON Schema](https://eips.ethereum.org/EIPS/eip-721).

## Running this contract

You will need Rust 1.44.1+ with `wasm32-unknown-unknown` target installed.

You can run unit tests on this via: 

`cargo test`

Once you are happy with the content, you can compile it to wasm via:

```
RUSTFLAGS='-C link-arg=-s' cargo wasm
cp ../../target/wasm32-unknown-unknown/release/cw721_base.wasm .
ls -l cw721_base.wasm
sha256sum cw721_base.wasm
```

Or for a production-ready (optimized) build, run a build command in the
the repository root: https://github.com/CosmWasm/cw-plus#compiling.

## Importing this contract

You can also import much of the logic of this contract to build another
CW721-compliant contract, such as tradable names, crypto kitties,
or tokenized real estate.

Basically, you just need to write your handle function and import 
`cw721_base::contract::handle_transfer`, etc and dispatch to them.
This allows you to use custom `ExecuteMsg` and `QueryMsg` with your additional
calls, but then use the underlying implementation for the standard cw721
messages you want to support. The same with `QueryMsg`. You will most
likely want to write a custom, domain-specific `instantiate`.

**TODO: add example when written**

For now, you can look at [`cw721-staking`](../cw721-staking/README.md)
for an example of how to "inherit" cw721 functionality and combine it with custom logic.
The process is similar for cw721.
