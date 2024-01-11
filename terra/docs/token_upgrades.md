# Upgrading the CW20 wrapped token contract

## Background: CosmWasm upgrades

CosmWasm contracts are deployed by first uploading the WASM bytecode, then instantiating a contract from that bytecode. The bytecode itself gets assigned a "code ID", which is an incrementally allocated identifier to uploaded WASM bytecodes. A given code ID can be instantiated multiple times, each time running the initialiser (the `instantiate` entrypoint).

Upgrades to a contract can then be performed by the contract owner sending a `migrate` message to the contract with the new code ID. The contract's storage will remain intact, but the underlying code id (and thus the bytecode) is replaced with the new ID. The runtime also executes the `migrate` entrypoint of the *new* bytecode within the upgrade transaction atomically.

## Background: Token bridge CW20 wrapped tokens

When the token bridge is instantiated, the wrapped asset code ID is passed to the instantiation handler:

```rust
pub struct InstantiateMsg {
    // governance contract details
    pub gov_chain: u16,
    pub gov_address: Binary,

    pub wormhole_contract: HumanAddr,
    pub wrapped_asset_code_id: u64, // <---- code id
}
```
[wormhole/terra/contracts/token-bridge/src/msg.rs#L9-L16](https://github.com/wormhole-foundation/wormhole/blob/dee0d1532b4a4ab6657dbdd1f0b8d19eadd90ec9/terra/contracts/token-bridge/src/msg.rs#L9-L16)

then during wrapped asset creation, the token bridge contract instantiates new instances of this contract by sending the `instantiate` message with the appropriate code id:

```rust
        CosmosMsg::Wasm(WasmMsg::Instantiate {
            admin: Some(env.contract.address.clone().into_string()),
            code_id: cfg.wrapped_asset_code_id,
            msg: to_binary(&WrappedInit {
              ...
            })?,
            ...
        })
```
[wormhole/terra/contracts/token-bridge/src/contract.rs#L458-L477](https://github.com/wormhole-foundation/wormhole/blob/dee0d1532b4a4ab6657dbdd1f0b8d19eadd90ec9/terra/contracts/token-bridge/src/contract.rs#L458-L477)


## Upgrading the wrapped CW20 contract

When upgrading the token contract, two steps need to be taken:

1. Update the code ID in the state so future wrapped assets are created from the new ID:

```rust
    let mut c = config(deps.storage).load()?;
    c.wrapped_asset_code_id = new_code_id;
    config(deps.storage).save(&c)?;
```
[wormhole/terra/contracts/token-bridge/src/contract.rs#L79-L81](https://github.com/wormhole-foundation/wormhole/blob/dee0d1532b4a4ab6657dbdd1f0b8d19eadd90ec9/terra/contracts/token-bridge/src/contract.rs#L79-L81)

2. Migrate all existing contracts to the new code ID. A simple implementation of this function can be seen [here](https://github.com/wormhole-foundation/wormhole/blob/dee0d1532b4a4ab6657dbdd1f0b8d19eadd90ec9/terra/contracts/token-bridge/src/contract.rs#L123-L147). Unfortunately, while this simple approach works in local testing, it does not scale to mainnet, the contract sends too many migrate messages (one to each wrapped asset), and exceeds the gas counter.

Currently there is no implementation of a token contract migration function that works on mainnet. If the need arises, one would have to be designed. If the upgrade does not require the migration to happen atomically, then it could be handled with a new permissionless entrypoint that can handle a range of contracts, and it could be done in multiple transactions. If for some reason atomicity is required, then extra care must be taken to pause all other bridge operation and perform the upgrade over multiple transactions.

## Historical note:

In Aug 2022, an attempt to upgrade the the token contracts was made, but failed due to the gas error:
https://finder.terra.money/classic/tx/FE39E9549770F59E2AAA1C6B0B86DDF36A4C56CED0CFB0CA4C9D4CC9FBE1E5BA. A subsequent upgrade changed the code id of *new* wrapped contracts to 767, but did not perform the migration for old contracts. This means that currently (as of Dec 2023), some wrapped tokens are still on the old code id, and some (the ones deployed after Aug 2022) are on the new code. This discrepancy is fine in the current case, because it only affects the rendering of the token name (https://github.com/wormhole-foundation/wormhole/commit/c832b123fcfb017d55086cb4d71241370ed270c6).
