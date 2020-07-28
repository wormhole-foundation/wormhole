## Solana Wormhole Program

The `Wormhole` program acts as a bridge for Solana \<> Foreign Chain transfers using the WhP (WormHoleProtocol).

### Instructions

##### Initialize

Initializes a new Bridge at `bridge`.

| Index | Name   | Type         | signer | writeable | empty | derived |
| ----- | ------ | ------------ | ------ | --------- | ----- | ------- |
| 0     | owner  | Account      | ✅️    |           |       |         |
| 0     | bridge | BridgeConfig |        |           | ✅️   | ✅️     |

##### Lock

Burns a wrapped asset `token` from `sender` on the Solana chain.

The transfer proposal will be tracked at a new account `proposal` where VPAs will be submitted by guardians.

Parameters:

| Index | Name     | Type         | signer | writeable | empty | derived |
| ----- | -------- | ------------ | ------ | --------- | ----- | ------- |
| 0     | sender   | TokenAccount |        | ✅        |       |         |
| 1     | bridge   | BridgeConfig |        |           |       |         |
| 2     | proposal | LockProposal |        | ✅        | ✅    | ✅      |
| 3     | token    | WrappedAsset |        | ✅        |       | ✅      |

##### LockNative

Locks a Solana native token (spl-token) `token` from `sender` on the Solana chain by transferring it to the 
`custody_account`.

The transfer proposal will be tracked at a new account `proposal` where VPAs will be submitted by guardians.

| Index | Name            | Type         | signer | writeable | empty | derived |
| ----- | --------------- | ------------ | ------ | --------- | ----- | ------- |
| 0     | sender          | TokenAccount |        | ✅        |       |         |
| 1     | bridge          | BridgeConfig |        |           |       |         |
| 2     | proposal        | LockProposal |        | ✅        | ✅    | ✅      |
| 3     | token           | Mint         |        | ✅        |       |         |
| 4     | custody_account | Mint         |        | ✅        | opt   | ✅      |

##### PostVPA

Submits a VPA signed by `guardian` on a valid `proposal`.

| Index | Name            | Type         | signer | writeable | empty | derived |
| ----- | --------------- | ------------ | ------ | --------- | ----- | ------- |
| 0     | guardian          | Account |    ✅    |         |       |         |
| 1     | bridge          | BridgeConfig |        |           |       |         |
| 2     | proposal        | LockProposal |        | ✅        |     | ✅      |

##### Reclaim

Reclaim tokens that did not receive enough VPAs on the `proposal` within the `SIGN_PERIOD` to finish the transfer.
`claimant` will get back the `locked_token` previously locked via `ILock`. 

| Index | Name         | Type         | signer | writeable | empty | derived |
| ----- | ------------ | ------------ | ------ | --------- | ----- | ------- |
| 0     | claimant     | TokenAccount |        | ✅        |       |         |
| 1     | bridge       | BridgeConfig |        |           |       |         |
| 2     | proposal     | LockProposal |        | ✅        |       | ✅      |
| 3     | locked_token | WrappedAsset |        |           |       | ✅      |

##### ReclaimNative

Reclaim tokens that did not receive enough VPAs on the `proposal` within the `SIGN_PERIOD` to finish the transfer.
`claimant` will get back the `locked_token` previously locked via `ILockNative` from the `custody_account`. 

| Index | Name            | Type         | signer | writeable | empty | derived |
| ----- | --------------- | ------------ | ------ | --------- | ----- | ------- |
| 0     | claimant        | TokenAccount |        | ✅        |       |         |
| 1     | bridge          | BridgeConfig |        |           |       |         |
| 2     | proposal        | LockProposal |        | ✅        |       | ✅      |
| 3     | locked_token    | Mint         |        |           |       |         |
| 4     | custody_account | Mint         |        | ✅        |       | ✅      |

##### EvictLock

Deletes a `proposal` after the `BRIDGE_WAIT_PERIOD` to free up space on chain. This returns the rent to `guardian`.

| Index | Name     | Type         | signer | writeable | empty | derived |
| ----- | -------- | ------------ | ------ | --------- | ----- | ------- |
| 0     | guardian | Account      | ✅     |           |       |         |
| 1     | bridge   | BridgeConfig |        |           |       |         |
| 2     | proposal | LockProposal |        | ✅        |       | ✅      |

##### ConfirmForeignLockup

The `guardian` confirms that a user locked up a foreign asset on a foreign chain.
This creates or updates a `proposal` to mint the wrapped asset `token` to `destination`.
If enough confirmations have been submitted, this instruction mints the token.

| Index | Name        | Type                   | signer | writeable | empty | derived |
| ----- | ----------- | ---------------------- | ------ | --------- | ----- | ------- |
| 0     | guardian    | Account                | ✅     |           |       |         |
| 1     | bridge      | BridgeConfig           |        |           |       |         |
| 2     | proposal    | ReleaseWrappedProposal | opt    | ✅        |       | ✅      |
| 3     | token       | WrappedAsset           |        |           |   opt    | ✅      |
| 4     | destination | TokenAccount           |        | ✅        | opt?  |         |

##### ConfirmForeignLockupOfNative

The `guardian` confirms that a user locked up a native asset on a foreign chain.
This creates or updates a `proposal` to release the `token` to `destination` from `custody_src`.
If enough confirmations have been submitted, this instruction releases the token.

| Index | Name        | Type                   | signer | writeable | empty | derived |
| ----- | ----------- | ---------------------- | ------ | --------- | ----- | ------- |
| 0     | guardian    | Account                | ✅     |           |       |         |
| 1     | bridge      | BridgeConfig           |        |           |       |         |
| 2     | proposal    | ReleaseWrappedProposal | opt    | ✅        |   opt    | ✅      |
| 3     | token       | WrappedAsset           |        |           |       | ✅      |
| 4     | custody_src | TokenAccount           |        | ✅        |       | ✅      |
| 5     | destination | TokenAccount           |        | ✅        | opt?  |         |

##### EvictRelease

Deletes a `proposal` after the `RELEASE_WRAPPED_TIMEOUT_PERIOD` to free up space on chain. This returns the rent to `guardian`.

| Index | Name     | Type                            | signer | writeable | empty | derived |
| ----- | -------- | ------------------------------- | ------ | --------- | ----- | ------- |
| 0     | guardian | Account                         | ✅     |           |       |         |
| 1     | bridge   | BridgeConfig                    |        |           |       |         |
| 2     | proposal | ReleaseWrappedProposal |        | ✅        |       | ✅      |✅ |

##### ChangeGuardianAdmin

This instruction is used to change the admin account of a guardian i.e. the account that manages rewards and the
signer account.

| Index | Name         | Type         | signer | writeable | empty | derived |
| ----- | ------------ | ------------ | ------ | --------- | ----- | ------- |
| 0     | guardian     | Account      | ✅     |           |       |         |
| 1     | bridge       | BridgeConfig |        | ✅        |       |         |
| 2     | new_guardian | Account      | ✅     |           |       |         |

##### ChangeGuardianSigner

This instruction is used to change the signer account of a guardian.

| Index | Name       | Type         | signer | writeable | empty | derived |
| ----- | ---------- | ------------ | ------ | --------- | ----- | ------- |
| 0     | guardian   | Account      | ✅     |           |       |         |
| 1     | bridge     | BridgeConfig |        | ✅        |       |         |
| 2     | new_signer | Account      | ✅     |           |       |         |

### Accounts

The following types of accounts are owned by creators of bridges:

##### _BridgeConfig_ Account

This account tracks the configuration of the transfer bridge.

| Parameter           | Description                                                                                                                                                   |
| ------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| SIGN_PERIOD         | The period in which enough foreign chain signatures need to be aggregated before tokens are freed up again                                                    |
| BRIDGE_WAIT_PERIOD  | The period after enough signatures have been published to a _lock account_ after which the account can be evicted. This exists to guarantee data availability |
| RELEASE_WRAPPED_TIMEOUT_PERIOD | The period in which enough votes need to be cast for an asset to be minted.                                                                                   |

### Program Accounts

The program own the following types of accounts:

##### _LockProposal_ Account

> Seed derivation: `lock_<chain>_<asset>_<lock_hash>`
>
> **chain**: CHAIN_ID of the native chain of this asset
>
> **asset**: address of the asset
>
> **lock_hash**: Random ID of the lock

This account is created when a user wants to lock tokens to transfer them to a foreign chain using the `ILock` instruction.

It tracks the progress of validator signatures. If not enough valid signatures are submitted within `SIGN_PERIOD`,
the tokens can be claimed by the user using the `IReclaim` instruction.

If enough signatures have been submitted, the account can be deleted using `IEvictLock` after `BRIDGE_WAIT_PERIOD`,
freeing up the rent.

##### _ReleaseWrappedProposal_ Account

> Seed derivation: `release_<chain>_<asset>_<foreign_lock_hash>`
>
> **chain**: CHAIN_ID of the native chain of this asset
>
> **asset**: address of the asset
>
> **foreign_lock_hash**: Hash of the foreign chain lock transaction

This account is created when the first validator sees a _fully confirmed_ **Lockup** of an asset on a foreign chain.

It tracks the confirmations of validators that have also seen the Lockup using `IConfirmForeignLockup`.

Once enough votes have been cast within the `RELEASE_WRAPPED_TIMEOUT_PERIOD`, this account is evicted and wrapped tokens are minted
or native tokens released.

If not enough votes are cast within the `RELEASE_WRAPPED_TIMEOUT_PERIOD`, the account can be evicted and the release aborted using
`IEvictRelease`.

##### _WrappedAsset_ Mint

> Seed derivation: `wrapped_<chain>_<asset>`
>
> **chain**: CHAIN_ID of the native chain of this asset
>
> **asset**: address of the asset on the foreign chain

This account is an instance of `spl-token/Mint` tracks a wrapped asset on the Solana chain.

##### _NativeAsset_ TokenAccount

> Seed derivation: `custody_<asset>`
>
> **asset**: address of the asset on the native chain

This account is an instance of `spl-token/TokenAccount` and holds spl tokens in custody that have been transferred to a
foreign chain.
