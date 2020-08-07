# Solana Wormhole Program

The `Wormhole` program acts as a bridge for Solana \<> Foreign Chain transfers using the WhP (WormHoleProtocol).

## Instructions

#### Initialize

Initializes a new Bridge at `bridge`.

| Index | Name   | Type         | signer | writeable | empty | derived |
| ----- | ------ | ------------ | ------ | --------- | ----- | ------- |
| 0     | sys | SystemProgram |        |           | ️   |      |
| 1     | clock | Sysvar |        |           | ️   | ✅     |
| 2     | bridge | BridgeConfig |        |           | ✅️   | ✅️     |
| 3     | guardian_set | GuardianSet         |        |   ✅       | ✅    | ✅      |
| 4     | payer | Account         |    ✅     |          |     |       |

#### TransferOut

Burns a wrapped asset `token` from `sender` on the Solana chain.

The transfer proposal will be tracked at a new account `proposal` where VAAs will be submitted by guardians.

Parameters:

| Index | Name     | Type                | signer | writeable | empty | derived |
| ----- | -------- | ------------------- | ------ | --------- | ----- | ------- |
| 0     | sys | SystemProgram |        |           | ️   |      |
| 1     | token_program | SplToken |        |           | ️   |      |
| 2     | token_account   | TokenAccount        |        | ✅        |       |         |
| 3     | bridge   | BridgeConfig        |        |           |       |         |
| 4     | proposal | TransferOutProposal |        | ✅        | ✅    | ✅      |
| 5     | token    | WrappedAsset        |        | ✅        |       | ✅      |
| 6     | payer    | Account        |    ✅    |         |       |       |

#### TransferOutNative

Locks a Solana native token (spl-token) `token` from `sender` on the Solana chain by transferring it to the
`custody_account`.

The transfer proposal will be tracked at a new account `proposal` where a VAA will be submitted by guardians.

| Index | Name            | Type                | signer | writeable | empty | derived |
| ----- | --------------- | ------------------- | ------ | --------- | ----- | ------- |
| 0     | sys | SystemProgram |        |           | ️   |      |
| 1     | token_program | SplToken |        |           | ️   |      |
| 2     | token_account          | TokenAccount        |        | ✅        |       |         |
| 3     | bridge          | BridgeConfig        |        |           |       |         |
| 4     | proposal        | TransferOutProposal |        | ✅        | ✅    | ✅      |
| 5     | token           | Mint                |        | ✅        |       |         |
| 6     | payer    | Account        |    ✅    |         |       |       |
| 7     | custody_account | TokenAccount                |        | ✅        | opt   | ✅      |

#### EvictTransferOut

Deletes a `proposal` after the `VAA_EXPIRATION_TIME` to free up space on chain. This returns the rent to `guardian`.

| Index | Name     | Type                | signer | writeable | empty | derived |
| ----- | -------- | ------------------- | ------ | --------- | ----- | ------- |
| 0     | guardian | Account             | ✅     |           |       |         |
| 1     | clock | Sysvar |        |           | ️   | ✅     |
| 2     | bridge   | BridgeConfig        |        |           |       |         |
| 3     | proposal | TransferOutProposal |        | ✅        |       | ✅      |

#### EvictExecutedVAA

Deletes a `ExecutedVAA` after the `VAA_EXPIRATION_TIME` to free up space on chain. This returns the rent to `guardian`.

| Index | Name     | Type                | signer | writeable | empty | derived |
| ----- | -------- | ------------------- | ------ | --------- | ----- | ------- |
| 0     | guardian | Account             | ✅     |           |       |         |
| 1     | clock | Sysvar |        |           | ️   | ✅     |
| 2     | bridge   | BridgeConfig        |        |           |       |         |
| 3     | proposal | ExecutedVAA |        | ✅        |       | ✅      |

#### PostVAA

Submits a VAA signed by the guardians to perform an action.

The required accounts depend on the `action` of the VAA:

All require:

| Index | Name         | Type         | signer | writeable | empty | derived |
| ----- | ------------ | ------------ | ------ | --------- | ----- | ------- |
| 0     | sys | SystemProgram |        |           | ️   |      |
| 1     | clock | Sysvar |        |           | ️   | ✅     |
| 2     | bridge       | BridgeConfig |        |           |       |         |  
| 3     | guardian_set | GuardianSet  |        |           |       |         |
| 4     | claim     | ExecutedVAA |        | ✅        |   ✅    | ✅      |
| 5     | payer  | Account |  ✅      |         |    |         |

followed by:

##### Guardian set update

| Index | Name         | Type                | signer | writeable | empty | derived |
| ----- | ------------ | ------------------- | ------ | --------- | ----- | ------- |
| 6     | guardian_set_new | GuardianSet         |        |   ✅       |  ✅   | ✅      |

##### Transfer: Ethereum (native) -> Solana (wrapped)

| Index | Name         | Type         | signer | writeable | empty | derived |
| ----- | ------------ | ------------ | ------ | --------- | ----- | ------- |
| 6     | token_program | SplToken |        |           | ️   |      |
| 7     | token        | WrappedAsset |        |           | opt   | ✅      |
| 8     | destination  | TokenAccount |        | ✅        | opt   |         |
| 9     | wrapped_meta_account | WrappedAssetMeta  |        | ✅        | opt   | ✅      |

##### Transfer: Ethereum (wrapped) -> Solana (native)

| Index | Name         | Type         | signer | writeable | empty | derived |
| ----- | ------------ | ------------ | ------ | --------- | ----- | ------- |
| 6     | token_program | SplToken |        |           | ️   |      |
| 7     | token        | Mint         |        |           |       | ✅      |
| 8     | destination  | TokenAccount |        | ✅        |    opt   |         |
| 9     | custody_src  | TokenAccount |        | ✅        |       | ✅      |

##### Transfer: Solana (any) -> Ethereum (any)

| Index | Name         | Type                | signer | writeable | empty | derived |
| ----- | ------------ | ------------------- | ------ | --------- | ----- | ------- |
| 6     | out_proposal | TransferOutProposal |        | ✅        |       | ✅      |

## Accounts

The following types of accounts are owned by creators of bridges:

#### _BridgeConfig_ Account

This account tracks the configuration of the transfer bridge.

| Parameter          | Description                                                                                                                                                     |
| ------------------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| VAA_EXPIRATION_TIME | Period for how long a VAA is valid. This exists to guarantee data availability and prevent replays|
| GUARDIAN_SET_INDEX | Index of the current active guardian set //TODO do we need to track this if the VAA contains the index?                                                         |

## Program Accounts

The program own the following types of accounts:

#### _ExecutedVAA_ Account

> Seed derivation: `executedvaa_<vaa_hash>`
>
> **vaa_hash**: Hash of the VAA

This account is created when a VAA is executed/consumed on Solana (i.e. not when a TransferOutProposal is approved).
It tracks a used VAA to protect from replay attacks where a VAA is executed multiple times. This account stays active
until the `VAA_EXPIRATION_TIME` has passed and can then be evicted using `IEvictExecutedVAA`.

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
after `VAA_EXPIRATION_TIME` has passed.

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
