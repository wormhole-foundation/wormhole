# Token ID encoding in the Token bridge

## Background

Terra classic (being a Cosmos chain with a CosmWasm runtime) supports two types of tokens: "native Bank tokens" and "CW20 tokens".
The Bank module is a native Cosmos SDK module that supports the transfer of tokens between accounts. These tokens are identified by their denomination (e.g. `uluna` or `uusd`), and are analogous to the native tokens on other chains, such as eth on Ethereum or sol on Solana.
CW20 tokens on the other hand are smart contracts that implement the CW20 interface, which is analogous to the ERC20 interface on EVM chains. These tokens are identified by their contract address.
The Terra token bridge supports both types of tokens directly (i.e. without wrapping them in a synthetic token like the EVM token bridge wraps eth into the canonical Wrapped ETH ERC20 contract). This means that the token bridge needs to be able to distinguish between the two types of tokens.

Addresses (both account and CosmWasm contract) on Terra used to fit into 20 bytes.
This changed when the chain underwent a hard fork to upgrade the runtime to CosmWasm 1.1.0 in June 2023. New contract addresses and account addresses are now 32 bytes long.

The initial design was made with the assumption that addresses are 20 bytes long. We first discuss that original version, and the adjustments that were made to support 32 byte addresses.

## Token ID encoding before CosmWasm 1.1.0

In the [Wormhole Token Bridge wire format](../../whitepapers/0003_token_bridge.md), token addresses are encoded as 32 bytes. Since CW20 addresses were 20 bytes long, the first 12 bytes were set to zero. The decision was also made to limit the length of native denom strings to 20 bytes also. This meant that the first 12 bytes of both CW20 addresses and native denoms were always zero.
The way the token bridge would then distinguish between the two is by writing a `0x01` byte in the first byte position of native denoms. Then, if the first byte of the token address is `0x01`, the token is a native denom, and if it is `0x00`, the token is a CW20 token. If it is anything else, the token is invalid.

```rust
    let marker_byte = transfer_info.token_address.as_slice()[0];
    if transfer_info.token_chain == CHAIN_ID {
        match marker_byte {
            1 => handle_complete_transfer_token_native(...),
            0 => handle_complete_transfer_token(...),
            b => Err(StdError::generic_err(format!("Unknown marker byte: {b}"))),
        }
    } else {
        handle_complete_transfer_token(...)
    }
```
[wormhole/terra/contracts/token-bridge/src/contract.rs#L734-L770](https://github.com/wormhole-foundation/wormhole/blob/dee0d1532b4a4ab6657dbdd1f0b8d19eadd90ec9/terra/contracts/token-bridge/src/contract.rs#L734-L770)

## Token ID encoding after CosmWasm 1.1.0

After the hard fork, addresses can now be 32 bytes long. Theoretically this would mean that new (32 byte addressed) tokens bridged out then back could collide with that check above. However, on the way out the token's address was checked to fit into 20 bytes, so no 32 byte addressed CW20 could be bridged out. That is, prior to upgrading the contracts to CW 1.1.0, the token bridge would not allow bridging out 32 byte addressed CW20 tokens.

In order to support 32 byte addresses, we simply change the above check so instead of just checking that the first byte is `0x01` for native denoms, we check that the first byte is `0x01` and the next 11 bytes are `0x00`:
```rust
fn is_native_id(address: &[u8]) -> bool {
    address[0] == 1 && address[1..12].iter().all(|&x| x == 0)
}
```
[wormhole/terra/contracts/token-bridge/src/contract.rs#L1434-L1436](https://github.com/wormhole-foundation/wormhole/blob/6e9127bd2a0a3d7f71ac6709a2893f6132bfe3ae/terra/contracts/token-bridge/src/contract.rs#L1434-L1436)

Now the check becomes:

```rustic
    if transfer_info.token_chain == CHAIN_ID && is_native_id(transfer_info.token_address.as_slice())
    {
        handle_complete_transfer_token_native(...)
    } else {
        handle_complete_transfer_token(...)
    }
```
[wormhole/terra/contracts/token-bridge/src/contract.rs#L684-L707](https://github.com/wormhole-foundation/wormhole/blob/6e9127bd2a0a3d7f71ac6709a2893f6132bfe3ae/terra/contracts/token-bridge/src/contract.rs#L684-L707)

This is backwards compatible with the old encoding, but also allows for 32 byte addressed CW20 tokens. There is a theoretical possibility that the CW20 address happens have the first 12 bytes in the form `0x01 0x00 0x00 ... 0x00`, but this is extremely unlikely (1 in 2^96, assuming that the bits of the address are uniformly distributed).
