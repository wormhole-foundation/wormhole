# Solana Wormhole Program

The `Wormhole` program acts as a bridge for Solana \<> Foreign Chain transfers using the WhP (WormHoleProtocol).

## Instructions

#### Initialize

Initializes a new Bridge at `bridge`.

| Index | Name         | Type          | signer | writeable | empty | derived |
| ----- | ------       | ------------  | ------ | --------- | ----- | ------- |
|     0 | sys          | SystemProgram |        |           |       |         |
|     1 | clock        | Sysvar        |        |           |       | ✅      |
|     2 | bridge       | BridgeConfig  |        |           | ✅    | ✅      |
|     3 | guardian_set | GuardianSet   |        | ✅        | ✅    | ✅      |
|     4 | payer        | Account       | ✅     |           |       |         |

#### PokeProposal

Pokes a `TransferOutProposal` so it is reprocessed by the guardians.

| Index | Name     | Type                | signer | writeable | empty | derived |
| ----- | ------   | ------------        | ------ | --------- | ----- | ------- |
| 0     | proposal | TransferOutProposal |        | ✅        | ️      | ✅      |

#### CreateWrappedAsset

Creates a new `WrappedAsset` to be used to create accounts and later receive transfers on chain.

| Index | Name                 | Type                | signer | writeable | empty | derived |
| ----- | --------             | ------------------- | ------ | --------- | ----- | ------- |
|     0 | sys                  | SystemProgram       |        |           |       |         |
|     1 | token_program        | SplToken            |        |           |       |         |
|     2 | rent                 | Sysvar              |        |           |       | ✅      |
|     3 | bridge               | BridgeConfig        |        |           |       |         |
|     4 | payer                | Account             | ✅     |           |       |         |
|     5 | wrapped_mint         | WrappedAsset        |        |           | ✅    | ✅      |
|     6 | wrapped_meta_account | WrappedAssetMeta    |        | ✅        | ✅    | ✅      |

#### VerifySignatures

Checks secp checks (in the previous instruction) and stores results.

| Index | Name         | Type           | signer | writeable | empty | derived |
| ----- | ------       | ------------   | ------ | --------- | ----- | ------- |
|     0 | bridge_p     | BridgeProgram  |        |           |       |         |
|     1 | sys          | SystemProgram  |        |           |       |         |
|     2 | instructions | Sysvar         |        |           |       | ✅      |
|     3 | sig_status   | SignatureState |        | ✅        |       |         |
|     4 | guardian_set | GuardianSet    |        |           |       | ✅      |
|     5 | payer        | Account        | ✅     |           |       |         |

#### TransferOut

Burns a wrapped asset `token` from `sender` on the Solana chain.

The transfer proposal will be tracked at a new account `proposal` where VAAs will be submitted by guardians.

Parameters:

| Index | Name          | Type                | signer | writeable | empty | derived |
| ----- | --------      | ------------------- | ------ | --------- | ----- | ------- |
|     0 | bridge_p      | BridgeProgram       |        |           |       |         |
|     1 | sys           | SystemProgram       |        |           |       |         |
|     2 | token_program | SplToken            |        |           |       |         |
|     3 | rent          | Sysvar              |        |           |       | ✅      |
|     4 | clock         | Sysvar              |        |           |  ✅     |         |
|     5 | token_account | TokenAccount        |        | ✅        |       |         |
|     6 | bridge        | BridgeConfig        |        |           |       |         |
|     7 | proposal      | TransferOutProposal |        | ✅        | ✅    | ✅      |
|     8 | token         | WrappedAsset        |        | ✅        |       | ✅      |
|     9 | payer         | Account             | ✅     |           |       |         |

#### TransferOutNative

Locks a Solana native token (spl-token) `token` from `sender` on the Solana chain by transferring it to the
`custody_account`.

The transfer proposal will be tracked at a new account `proposal` where a VAA will be submitted by guardians.

| Index | Name            | Type                | signer | writeable | empty | derived |
| ----- | --------------- | ------------------- | ------ | --------- | ----- | ------- |
|     0 | bridge_p        | BridgeProgram       |        |           |       |         |
|     1 | sys             | SystemProgram       |        |           |       |         |
|     2 | token_program   | SplToken            |        |           |       |         |
|     3 | rent            | Sysvar              |        |           |       | ✅      |
|     4 | clock           | Sysvar              |        |           |       | ✅      |
|     5 | token_account   | TokenAccount        |        | ✅        |       |         |
|     6 | bridge          | BridgeConfig        |        |           |       |         |
|     7 | proposal        | TransferOutProposal |        | ✅        | ✅    | ✅      |
|     8 | token           | Mint                |        | ✅        |       |         |
|     9 | payer           | Account             | ✅     |           |       |         |
|    10 | custody_account | TokenAccount        |        | ✅        | opt   | ✅      |

#### EvictTransferOut

Deletes a `proposal` after the `VAA_EXPIRATION_TIME` to free up space on chain. This returns the rent to `guardian`.

| Index | Name     | Type                | signer | writeable | empty | derived |
| ----- | -------- | ------------------- | ------ | --------- | ----- | ------- |
|     0 | bridge_p | BridgeProgram       |        |           |       |         |
|     1 | guardian | Account             | ✅     |           |       |         |
|     2 | clock    | Sysvar              |        |           |       | ✅      |
|     3 | bridge   | BridgeConfig        |        |           |       |         |
|     4 | proposal | TransferOutProposal |        | ✅        |       | ✅      |

#### EvictClaimedVAA

Deletes a `ClaimedVAA` after the `VAA_EXPIRATION_TIME` to free up space on chain. This returns the rent to `guardian`.

| Index | Name     | Type                | signer | writeable | empty | derived |
| ----- | -------- | ------------------- | ------ | --------- | ----- | ------- |
|     0 | bridge_p | BridgeProgram       |        |           |       |         |
|     1 | guardian | Account             | ✅     |           |       |         |
|     2 | clock    | Sysvar              |        |           |       | ✅      |
|     3 | bridge   | BridgeConfig        |        |           |       |         |
|     4 | claim    | ClaimedVAA          |        | ✅        |       | ✅      |

#### SubmitVAA

Submits a VAA signed by the guardians to perform an action.

The required accounts depend on the `action` of the VAA:

All require:

| Index | Name         | Type          | signer | writeable | empty | derived |
| ----- | ------------ | ------------  | ------ | --------- | ----- | ------- |
|     0 | bridge_p     | BridgeProgram |        |           |       |         |
|     1 | sys          | SystemProgram |        |           |       |         |
|     2 | rent         | Sysvar        |        |           |       | ✅      |
|     3 | clock        | Sysvar        |        |           |       | ✅      |
|     4 | bridge       | BridgeConfig  |        |           |       |         |
|     5 | guardian_set | GuardianSet   |        |           |       |         |
|     6 | claim        | ExecutedVAA   |        | ✅        | ✅    | ✅      |
|     7 | sig_info     | SigState      |        |           | ✅    |         |
|     8 | payer        | Account       | ✅     |           |       |         |

followed by:

##### Guardian set update

| Index | Name             | Type                | signer | writeable | empty | derived |
| ----- | ------------     | ------------------- | ------ | --------- | ----- | ------- |
| 9     | guardian_set_new | GuardianSet         |        | ✅        | ✅    | ✅      |

##### Transfer: Ethereum (native) -> Solana (wrapped)

| Index | Name          | Type         | signer | writeable | empty | derived |
| ----- | ------------  | ------------ | ------ | --------- | ----- | ------- |
|     9 | token_program | SplToken     |        |           |       |         |
|    10 | token         | WrappedAsset |        |           |       | ✅      |
|    11 | destination   | TokenAccount |        | ✅        |       |         |
|    12 | wrapped_meta  | WrappedMeta  |        | ✅        | opt   | ✅      |

##### Transfer: Ethereum (wrapped) -> Solana (native)

| Index | Name          | Type         | signer | writeable | empty | derived |
| ----- | ------------  | ------------ | ------ | --------- | ----- | ------- |
|     9 | token_program | SplToken     |        |           |       |         |
|    10 | token         | Mint         |        |           |       | ✅      |
|    11 | destination   | TokenAccount |        | ✅        | opt   |         |
|    12 | custody_src   | TokenAccount |        | ✅        |       | ✅      |

##### Transfer: Solana (any) -> Ethereum (any)

| Index | Name         | Type                | signer | writeable | empty | derived |
| ----- | ------------ | ------------------- | ------ | --------- | ----- | ------- |
| 9     | out_proposal | TransferOutProposal |        | ✅        |       | ✅      |

## Accounts

The following types of accounts are owned by creators of bridges:

#### _BridgeConfig_ Account

This account tracks the configuration of the transfer bridge.

| Parameter           | Description                                                                                              |
| ------------------  | -------------------------------------------------------------------------------------------------------- |
| VAA_EXPIRATION_TIME | Period for how long a VAA is valid. This exists to guarantee data availability and prevent replays       |
| GUARDIAN_SET_INDEX  | Index of the current active guardian set //TODO do we need to track this if the VAA contains the index?  |

## Program Accounts

The program own the following types of accounts:

#### _ClaimedVAA_ Account

> Seed derivation: `claim || <bridge> || <hash>`
>
> **bridge**: Pubkey of the bridge
>
> **hash**: signing hash of the VAA

This account is created when a VAA is executed/consumed on Solana (i.e. not when a TransferOutProposal is approved).
It tracks a used VAA to protect from replay attacks where a VAA is executed multiple times. This account stays active
until the `VAA_EXPIRATION_TIME` has passed and can then be evicted using `IEvictClaimedVAA`.

#### _GuardianSet_ Account

> Seed derivation: `guardian || <bridge> || <index>`
>
> **bridge**: Pubkey of the bridge
>
> **index**: Index of the guardian set

This account is created when a new guardian set is set. It tracks the public key, creation time and expiration time of
this set.
The expiration time is set when this guardian set is abandoned. When a switchover happens, the guardian-issued VAAs will
still be valid until the expiration time.

#### _TransferOutProposal_ Account

> Seed derivation: `transfer || <bridge> || <asset_chain> || <asset> || <target_chain> || <target_address> || <sender> || <nonce>`
>
> **bridge**: Pubkey of the bridge
>
> **asset_chain**: CHAIN_ID of the native chain of this asset
>
> **asset**: address of the asset
>
> **target_chain**: ChainID of the recipient
>
> **target_address**: address of the recipient
>
> **sender**: pubkey of the sender
>
> **nonce**: nonce of the transfer

This account is created when a user wants to lock tokens to transfer them to a foreign chain using the `ITransferOut`
instruction.

It is used to signal a pending transfer to a foreign chain and will also store the respective VAA provided using
`ISubmitVAA`.

Once the VAA has been published this TransferOut is considered completed and can be evicted using `EvictTransferOut`
after `VAA_EXPIRATION_TIME` has passed.

#### _WrappedAsset_ Mint

> Seed derivation: `wrapped || <bridge> || <chain> || <asset>`
>
> **bridge**: Pubkey of the bridge
>
> **chain**: CHAIN_ID of the native chain of this asset
>
> **asset**: address of the asset on the foreign chain

This account is an instance of `spl-token/Mint` tracks a wrapped asset on the Solana chain.

#### _WrappedAssetMeta_ Mint

> Seed derivation: `meta || <bridge> || <wrapped>`
>
> **bridge**: Pubkey of the bridge
>
> **wrapped**: address of the wrapped asset

This account tracks the metadata about a wrapped asset to allow reverse lookups.

#### _Custody_ TokenAccount

> Seed derivation: `custody || <bridge> || <asset>`
>
> **bridge**: Pubkey of the bridge
>
> **asset**: address of the asset mint on the native chain

This account is an instance of `spl-token/TokenAccount` and holds spl tokens in custody that have been transferred to a
foreign chain.
