# Solana Wormhole Program

The `Wormhole` program acts as a bridge for Solana \<> Foreign Chain transfers using the WhP (WormHoleProtocol).

## Instructions

#### Initialize

Initializes a new Bridge at `bridge`.

| Index | Name   | Type         | signer | writeable | empty | derived |
| ----- | ------ | ------------ | ------ | --------- | ----- | ------- |
| 0     | owner  | Account      | ✅️    |           |       |         |
| 0     | bridge | BridgeConfig |        |           | ✅️   | ✅️     |

#### TransferOut

Burns a wrapped asset `token` from `sender` on the Solana chain.

The transfer proposal will be tracked at a new account `proposal` where VAAs will be submitted by guardians.

Parameters:

| Index | Name     | Type                | signer | writeable | empty | derived |
| ----- | -------- | ------------------- | ------ | --------- | ----- | ------- |
| 0     | sender   | TokenAccount        |        | ✅        |       |         |
| 1     | bridge   | BridgeConfig        |        |           |       |         |
| 2     | proposal | TransferOutProposal |        | ✅        | ✅    | ✅      |
| 3     | token    | WrappedAsset        |        | ✅        |       | ✅      |

#### TransferOutNative

Locks a Solana native token (spl-token) `token` from `sender` on the Solana chain by transferring it to the
`custody_account`.

The transfer proposal will be tracked at a new account `proposal` where a VAA will be submitted by guardians.

| Index | Name            | Type                | signer | writeable | empty | derived |
| ----- | --------------- | ------------------- | ------ | --------- | ----- | ------- |
| 0     | sender          | TokenAccount        |        | ✅        |       |         |
| 1     | bridge          | BridgeConfig        |        |           |       |         |
| 2     | proposal        | TransferOutProposal |        | ✅        | ✅    | ✅      |
| 3     | token           | Mint                |        | ✅        |       |         |
| 4     | custody_account | Mint                |        | ✅        | opt   | ✅      |

#### EvictTransferOut

Deletes a `proposal` after the `BRIDGE_WAIT_PERIOD` to free up space on chain. This returns the rent to `guardian`.

| Index | Name     | Type                | signer | writeable | empty | derived |
| ----- | -------- | ------------------- | ------ | --------- | ----- | ------- |
| 0     | guardian | Account             | ✅     |           |       |         |
| 1     | bridge   | BridgeConfig        |        |           |       |         |
| 2     | proposal | TransferOutProposal |        | ✅        |       | ✅      |

#### PostVAA

Submits a VAA signed by the guardians to perform an action.

The required accounts depend on the `action` of the VAA:

##### Guardian set update

| Index | Name         | Type                | signer | writeable | empty | derived |
| ----- | ------------ | ------------------- | ------ | --------- | ----- | ------- |
| 0     | bridge       | BridgeConfig        |        | ✅        |       |         |
| 1     | guardian_set | GuardianSet         |        |   ✅       | ✅    | ✅      |
| 2     | proposal     | TransferOutProposal |        | ✅        |       | ✅      |

##### Ethereum (native) -> Solana (wrapped)

| Index | Name         | Type         | signer | writeable | empty | derived |
| ----- | ------------ | ------------ | ------ | --------- | ----- | ------- |
| 0     | bridge       | BridgeConfig |        |           |       |         |
| 1     | guardian_set | GuardianSet  |        |           |       |         |
| 2     | token        | WrappedAsset |        |           | opt   | ✅      |
| 3     | destination  | TokenAccount |        | ✅        | opt   |         |

##### Ethereum (wrapped) -> Solana (native)

| Index | Name         | Type         | signer | writeable | empty | derived |
| ----- | ------------ | ------------ | ------ | --------- | ----- | ------- |
| 0     | bridge       | BridgeConfig |        |           |       |         |
| 1     | guardian_set | GuardianSet  |        |           |       |         |
| 2     | token        | Mint         |        |           |       | ✅      |
| 3     | custody_src  | TokenAccount |        | ✅        |       | ✅      |
| 4     | destination  | TokenAccount |        | ✅        | opt   |         |

##### Solana (any) -> Ethereum (any)

| Index | Name         | Type                | signer | writeable | empty | derived |
| ----- | ------------ | ------------------- | ------ | --------- | ----- | ------- |
| 0     | bridge       | BridgeConfig        |        |           |       |         |
| 1     | guardian_set | GuardianSet         |        |           |       |         |
| 2     | out_proposal | TransferOutProposal |        | ✅        |       | ✅      |

## Accounts

The following types of accounts are owned by creators of bridges:

#### _BridgeConfig_ Account

This account tracks the configuration of the transfer bridge.

| Parameter          | Description                                                                                                                                                     |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| BRIDGE_WAIT_PERIOD | The period after a valid VAA has been published to a `transfer out proposal` after which the account can be evicted. This exists to guarantee data availability |
| GUARDIAN_SET_INDEX | Index of the current active guardian set //TODO do we need to track this if the VAA contains the index?                                                         |

## Program Accounts

The program own the following types of accounts:

#### _GuardianSet_ Account

> Seed derivation: `guardians_<index>`
>
> **index**: Index of the guardian set

This account is created when a new guardian set is set. It tracks the public key, creation time and expiration time of
this set.
The expiration time is set when this guardian set is abandoned. When a switchover happens, the guardian-issued VAAs will
still be valid until the expiration time.

#### _TransferOutProposal_ Account

> Seed derivation: `out_<chain>_<asset>_<transfer_hash>`
>
> **chain**: CHAIN_ID of the native chain of this asset
>
> **asset**: address of the asset
>
> **transfer_hash**: Random ID of the transfer

This account is created when a user wants to lock tokens to transfer them to a foreign chain using the `ITransferOut`
instruction.

It is used to signal a pending transfer to a foreign chain and will also store the respective VAA provided using
`IPostVAA`.

Once the VAA has been published this TransferOut is considered completed and can be evicted using `EvictTransferOut`
after `BRIDGE_WAIT_PERIOD`

#### _WrappedAsset_ Mint

> Seed derivation: `wrapped_<chain>_<asset>`
>
> **chain**: CHAIN_ID of the native chain of this asset
>
> **asset**: address of the asset on the foreign chain

This account is an instance of `spl-token/Mint` tracks a wrapped asset on the Solana chain.

#### _NativeAsset_ TokenAccount

> Seed derivation: `custody_<asset>`
>
> **asset**: address of the asset on the native chain

This account is an instance of `spl-token/TokenAccount` and holds spl tokens in custody that have been transferred to a
foreign chain.

## Archive

### Reclaim mechanism

**Options:**

| Parameter                      | Description                                                                 |
| ------------------------------ | --------------------------------------------------------------------------- |
| RELEASE_WRAPPED_TIMEOUT_PERIOD | The period in which enough votes need to be cast for an asset to be minted. |

Reclaim calls were intended to allow users to reclaim tokens if no VAA was provided in time. This would protect a user
against censorship attacks from guardians.

However this opens a window for race conditions where a VAA would be delayed and the user would frontrun that VAA with
a Reclaim.

#### Reclaim

Reclaim tokens that did not receive enough VAAs on the `proposal` within the `SIGN_PERIOD` to finish the transfer.
`claimant` will get back the `locked_token` previously locked via `ITransferOut`.

| Index | Name         | Type                | signer | writeable | empty | derived |
| ----- | ------------ | ------------------- | ------ | --------- | ----- | ------- |
| 0     | claimant     | TokenAccount        |        | ✅        |       |         |
| 1     | bridge       | BridgeConfig        |        |           |       |         |
| 2     | proposal     | TransferOutProposal |        | ✅        |       | ✅      |
| 3     | locked_token | WrappedAsset        |        |           |       | ✅      |

#### ReclaimNative

Reclaim tokens that did not receive enough VAAs on the `proposal` within the `SIGN_PERIOD` to finish the transfer.
`claimant` will get back the `locked_token` previously locked via `ITransferOutNative` from the `custody_account`.

| Index | Name            | Type                | signer | writeable | empty | derived |
| ----- | --------------- | ------------------- | ------ | --------- | ----- | ------- |
| 0     | claimant        | TokenAccount        |        | ✅        |       |         |
| 1     | bridge          | BridgeConfig        |        |           |       |         |
| 2     | proposal        | TransferOutProposal |        | ✅        |       | ✅      |
| 3     | locked_token    | Mint                |        |           |       |         |
| 4     | custody_account | Mint                |        | ✅        |       | ✅      |
