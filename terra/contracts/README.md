# Terra Wormhole Contracts

The Wormhole Terra integration is developed and maintained by Everstake / @ysavchenko.

## Summary

To facilitate token exchange via the Wormhole bridge blockchains must provide a set of smart contracts to process bridge commands on chain. Here are such contracts for the Terra blockchain.

The first contract, `cw20-wrapped` is basically a `cw20-base` contract ([see here](https://github.com/CosmWasm/cosmwasm-plus/tree/master/contracts/cw20-base)) instantiated for every new token type issued on the blockchain by the bridge. And the second one, `wormhole` provides the bridge functionality itself:

- It locks tokens on the Terra blockchain when they are sent out to other blockchains
- It sends out wrapped or original tokens (depending on the blockchain origin of the token) to recipients when receiving tokens from the other blockchains

## Details

### `cw20-wrapped`

This contract mostly wraps functionality of the `cw20-base` contract with the following differences:

- It stores `WrappedAssetInfo` state with information about the source blockchain, asset address on this blockchain and the `wormhole` contract address
- Once initialized it calls the hook action specified in the initialization params (`init_hook` field). It is used to record newly instantiated contract's address in the `wormhole` contract
- Full mint authority is provided to the `wormhole` contract

### `wormhole`

This contract controls token transfers, minting and burning as well as maintaining the list of guardians: off-chain
entities identified by their public key hashes, majority of whom can issue commands to the contract.

`wormhole` bridge processes the following instructions.

#### `SubmitVAA`

Receives VAAs from the guardians (read about VAAs [here](../../docs/protocol.md)), verifies and processes them. In the current bridge implementation VAAs can trigger the following actions:

- Send token to the Terra recipient
- Update the list of guardians

Sending tokens to the Terra recipient is handled by the `vaa_transfer` method. For the native Terra tokens it simply transfers the corresponding amount from its balance. For the non-native tokens `wormhole` either mints the corresponding amount from the already deployed `cw20-wrapped` contract or deploys a new one with the mint amount in the initialization message.

#### `RegisterAssetHook`

Gets called from the `cw20-wrapped` constructor to record its address in the contract's directory of wrapped assets. It is used later to check whether the wrapped contract for the asset is already deployed on Terra blockchain or not.

#### `LockAssets`

Called to initiate token transfer from the Terra blockchain to other blockchains. Caller must provide allowance to the `wormhole` contract to spend tokens, then the contract either transfers (if it is a native token) or burns it (if it is a wrapped token from the different blockchain). Then the information is logged to be read by the guardians operating the bridge, which triggers sending VAAs to the destination blockchain.

#### `SetActive`

Safety feature to turn off the `wormhole` contract in the case of any issues found in production. Only the owner can send this message, once the contract is inactive it stops processing token transfer commands.
