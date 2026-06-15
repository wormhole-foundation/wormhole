# Token Bridge App

[TOC]

## Objective

To use the Wormhole message passing protocol to transfer tokens between different connected chains.

## Background

The decentralized finance ecosystem is developing into a direction where different chains with different strengths
become the home for various protocols. However, a token is usually only minted on a single chain and therefore
disconnected from the ecosystem and protocols on other chains.

Each chain typically has one de-facto standard for token issuance, like ERC-20 on Ethereum and SPL on Solana. Those
standards, while not identical, all implement a similar interface with the same concepts like owning, minting,
transferring and burning tokens.

To connect chains, a token would ideally have a native mint - the original token on the chain it was originally created
on - and a wrapped version on other chains that represents ownership of the native token.

While the Wormhole messaging protocol provides a way to attest and transfer messages between chains which could
technically be used to implement bridging for individual tokens, this would require manual engineering effort for each
token and create incompatible protocols with bad UX.

## Goals

We want to implement a generalized token bridge using the Wormhole message passing protocol that is able to bridge any
standards-compliant token between chains, creating unique wrapped representations on each connected chain on demand.

- Allow transfer of standards-compliant tokens between chains.
- Allow creation of wrapped assets.
- Use a universal token representation that is compatible with most VM data types.
- Allow domain-specific payload to be transferred along the token, enabling
  tight integration with smart contracts on the target chain.
- Allow emergency pausing of token bridge operations.

## Non-Goals

- Support fee-burning / rebasing / non-standard tokens.
- Manage chain-specific token metadata that isn't broadly applicable to all chains.
- Automatically relay token transfer messages to the target chain.

## Overview

On each chain of the token bridge network there will be a token bridge endpoint program.

These programs will manage authorization of payloads (emitter filtering), wrapped representations of foreign chain
tokens ("Wrapped Assets") and custody locked tokens.

## Detailed Design

For outbound transfers, the contracts will have a lock method that either locks up a native token and produces a
respective Transfer message that is posted to Wormhole, or burns a wrapped token and produces/posts said message.

For inbound transfers they can consume, verify and process Wormhole messages containing a token bridge payload.

There will be seven different payloads:

- `Transfer` - Will trigger the release of locked tokens or minting of wrapped tokens.
- `TransferWithPayload` - Will trigger the release of locked tokens or minting of wrapped tokens, with additional domain-specific payload.
- `AssetMeta` - Attests asset metadata (required before the first transfer).
- `RegisterChain` - Register the token bridge contract (emitter address) for a foreign chain.
- `UpgradeContract` - Upgrade the contract.
- `RecoverChainId` - Recover the contract's `chainId` and `evmChainId` after a chain fork (EVM-only).
- `SetPauserAddresses` - Set the addresses authorized to pause, freeze, and unpause the token bridge.

Since anyone can use Wormhole to publish messages that match the payload format of the token bridge, an authorization
payload needs to be implemented. This is done using an `(emitter_chain, emitter_address)` tuple. Every endpoint of the
token bridge needs to know the addresses of the respective other endpoints on other chains. This registration of token
bridge endpoints is implemented via `RegisterChain` where a `(chain_id, emitter_address)` tuple can be registered. Only
one endpoint can be registered per chain. Endpoints are immutable. This payload will only be accepted if the emitter is
the hardcoded governance contract.

In order to transfer assets to another chain, a user needs to call the `transfer` (or the `transferWithPayload`) method
of the bridge contract with the recipient details and respective fee they are willing to pay. The contract will either
hold the tokens in a custody account (in case it is a native token) or burn wrapped assets. Wrapped assets can be burned
because they can be freely minted once tokens are transferred back and this way the total supply can indicate the total
amount of tokens currently held on this chain. After the lockup the contract will post a `Transfer` (or
`TransferWithPayload`) message to Wormhole. Once the message has been signed by the guardians, it can be posted to the
target chain of the transfer. Upon redeeming (see `completeTransfer` below), the target chain will either release native
tokens from custody or mint a wrapped asset depending on whether it's a native token there.
The token bridges guarantee that there will be a unique wrapped asset on each chain for each non-native token. In other
words, transferring a native token from chain A to chain C will result in the same wrapped token as transferring from A
to B first, then from B to C, and no double wrapping will happen.
The program will keep track of consumed message digests (which include a nonce) for replay prevention.

To redeem the transaction on the target chain, the VAA must be posted to the target token bridge. Since the VAA includes
a signature from the guardians, it does not matter in general who submits it to the target chain, as VAAs cannot be
spoofed, and the VAA includes the target address that the tokens will be sent to upon completion.
An exception to this is `TransferWithPayload`, which must be redeemed by the target address, because it contains
additional payload that must be handled by the recipient (such as token-swap instructions).
The `completeTransfer` method will accept a fee recipient. In case that field is set, the fee amount
specified will be sent to the fee recipient and the remainder of the amount to the intended receiver of the transfer.
This allows transfers to be completed by independent relayers to improve UX for users that will only need to send a
single transaction for as long as the fee is sufficient and the token accepted by anyone acting as a relayer.

In order to keep `Transfer` messages small, they don't carry all metadata of the token. However, this means that before
a token can be transferred to a new chain for the first time, the metadata needs to be bridged and the wrapped asset
created. Metadata in this case includes the amount of decimals which is a core requirement for instantiating a token.

The metadata of a token can be attested by calling `attestToken` on its respective native chain which will produce a
`AssetMeta` wormhole message. This message can be used to attest state and initialize a WrappedAsset on any chain in the
wormhole network using the details. A token is identified by the tuple `(chain_id, chain_address)` and metadata should
be mapped to this identifier. A wrapped asset may only ever be created once for a given identifier and not updated.

### Pausing

The token bridge supports an emergency pause for use during an active exploit. While paused, every entry point reverts except for governance handlers and the pause-management entry points (`pause`, `freeze`, `unpause`, and the permissionless `unpauseExpired`). The pause state is a boolean `paused`; a companion `pauseExpiry` timestamp records the point at which an active pause becomes eligible to be lifted permissionlessly.

Three roles control the pause state, each configured per chain via a `SetPauserAddresses` governance message:

- A `pauser` may call `pause` to set `paused` to `true` and set `pauseExpiry` to `block.timestamp + PAUSE_DURATION`, where `PAUSE_DURATION` is initially a hard-coded constant of 5 days. `pause` may be called repeatedly; each call pushes `pauseExpiry` to 5 days from the current time, so the bridge stays paused for as long as the `pauser` keeps re-pausing. A `pause` call MUST NOT reduce a `pauseExpiry` already further in the future (e.g. one set by `freeze`) - a lower-trust `pauser` cannot curtail a `freeze`. This requirement is why unpausing MUST set `pauseExpiry` to the current time: a stale expiry left over from a prior `freeze` would otherwise block a later `pause`.
- A `freezer` may call `freeze` to set `paused` to `true` and set `pauseExpiry` to the maximum representable timestamp, pausing the bridge for the maximum amount of time. A frozen bridge will not become permissionlessly unpausable in practice and can only be lifted by the `unpauser`. `freeze` is the higher-trust counterpart to the temporary, self-expiring `pauser`. A `freeze` call from the `freeze` role always succeeds: if the bridge is unpaused, it causes a pause; if already paused, it extends the existing pause to the maximum duration. Successive `freeze` calls are a no-op.
- An `unpauser` may call `unpause` to set `paused` back to `false` and `pauseExpiry` to `block.timestamp` at any time, regardless of `pauseExpiry`. Recording the current time (rather than `0`) leaves on-chain evidence of the last unpause while still bringing any stale `freeze` expiry down to the present so it cannot block a later `pause`. This is the privileged path to lift a pause (including a `freeze`) before it would otherwise expire.

Additionally, a permissionless `unpauseExpired` entry point allows anyone to set `paused` to `false` once `block.timestamp >= pauseExpiry`. This bounds a `pauser`-initiated pause to `PAUSE_DURATION` without requiring the `unpauser` to act, while ensuring the pause never lapses prematurely: the boolean `paused` remains authoritative, so a pause is only lifted by an explicit `unpause` or `unpauseExpired` call - never silently by the passage of time.

The roles are kept separate to allow for asymmetric authority between the assigned addresses - for example, a 2/3 multisig empowered to `pause` for short windows, a higher-threshold key empowered to `freeze`, and Wormhole governance retained for `unpause`. These SHOULD be effectively different roles, but this is intentionally not enforced on-chain or in the Guardian VAA generation process.

Any role may be left unset. A zero-length value or an all-zero address (of the target chain's native address size) is treated as the role being unassigned. When a role is unassigned, the corresponding entry point MUST revert before comparing the caller against the configured role. Implementations MUST NOT treat an all-zero address as an authorized caller. This allows governance to disable `pause`, `freeze`, or `unpause` without removing the entry point - for example, leaving `pauser` unassigned on chains where pause authority is not yet desired, or zeroing a key suspected of compromise without first provisioning its replacement. The permissionless `unpauseExpired` entry point has no associated role and is always callable. If `unpauser` is unassigned while the bridge is paused, recovery before `pauseExpiry` requires Wormhole governance to first assign a non-zero `unpauser` via `SetPauserAddresses`; otherwise the pause can be permissionlessly lifted once `pauseExpiry` has passed.

### Handling of token amounts and decimals

Due to constraints on some supported chains, all token amounts passed through the token bridge are truncated to a maximum of 8 decimals.

Any chains implementation must make sure that of any token only ever MaxUint64 units (post-shifting) are bridged into the wormhole network at any given time (all target chains combined), even though the slot is 32 bytes long (theoretically fitting uint256).

Token "dust" that can not be transferred due to truncation during a deposit needs to be refunded back to the user.

**Examples:**

- The amount "1" of a 18 decimal Ethereum token is originally represented as: `1000000000000000000`, over the wormhole it is passed as: `100000000`.
- The amount "2" of a 4 decimal token is represented as `20000` and is passed over the wormhole without a decimal shift.

**Handling on the target Chains:**

Implementations on target chains can handle the decimal shift in one of the following ways:

- In case the chain supports the original decimal amount (known from the `AssetMeta`) it can do a decimal shift back to the original decimal amount. This allows for out-of-the-box interoperability of DeFi protocols across for example different EVM environments.
- Otherwise the wrapped token should stick to the 8 decimals that the protocol uses.

### API / database schema

Proposed bridge interface:

`attestToken(address token)` - Produce a `AssetMeta` message for a given token

`transfer(address token, uint64-uint256 amount (size depending on chains standards), uint16 recipient_chain, bytes32 recipient, uint256 fee)` - Initiate
a `Transfer`. Amount in the tokens native decimals.

`transferWithPayload(address token, uint64-uint256 amount (size depending on chains standards), uint16 recipient_chain, bytes32 recipient, bytes payload)` - Initiate
a `TransferWithPayload`. Amount in the tokens native decimals. `payload` is an arbitrary binary blob.

`createWrapped(Message assetMeta)` - Creates a wrapped asset using `AssetMeta`

`completeTransfer(Message transfer)` - Execute a `Transfer` message

`completeTransferWithPayload(Message transfer)` - Execute a `TransferWithPayload` message

`registerChain(Message registerChain)` - Execute a `RegisterChain` governance message

`upgrade(Message upgrade)` - Execute a `UpgradeContract` governance message

`submitRecoverChainId(Message recoverChainId)` - Execute a `RecoverChainId` governance message. Only callable on a forked chain (EVM-only).

`pause()` - Set `paused` to `true` and `pauseExpiry` to `block.timestamp + PAUSE_DURATION` (5 days), never reducing a `pauseExpiry` already further in the future. Callable only by the `pauser`; reverts when `pauser` is unassigned. Not idempotent: each call pushes `pauseExpiry` forward.

`freeze()` - Set `paused` to `true` and `pauseExpiry` to the maximum representable timestamp. Callable only by the `freezer`; reverts when `freezer` is unassigned. Idempotent.

`unpause()` - Set `paused` to `false` and `pauseExpiry` to `block.timestamp`. Callable only by the `unpauser`; reverts when `unpauser` is unassigned.

`unpauseExpired()` - Set `paused` to `false` and `pauseExpiry` to `block.timestamp`. Permissionless; reverts unless `block.timestamp >= pauseExpiry`.

`setPauserAddresses(Message setPauserAddresses)` - Execute a `SetPauserAddresses` governance message

---

**Payloads**:

Transfer:

```
PayloadID uint8 = 1
// Amount being transferred (big-endian uint256)
Amount uint256
// Address of the token. Left-zero-padded if shorter than 32 bytes
TokenAddress bytes32
// Chain ID of the token
TokenChain uint16
// Address of the recipient. Left-zero-padded if shorter than 32 bytes
To bytes32
// Chain ID of the recipient
ToChain uint16
// Amount of tokens (big-endian uint256) that the user is willing to pay as relayer fee. Must be <= Amount.
Fee uint256
```

TransferWithPayload:

```
PayloadID uint8 = 3
// Amount being transferred (big-endian uint256)
Amount uint256
// Address of the token. Left-zero-padded if shorter than 32 bytes
TokenAddress bytes32
// Chain ID of the token
TokenChain uint16
// Address of the recipient. Left-zero-padded if shorter than 32 bytes
To bytes32
// Chain ID of the recipient
ToChain uint16
// The address of the message sender on the source chain
FromAddress bytes32
// Arbitrary payload
Payload bytes
```

AssetMeta:

```
PayloadID uint8 = 2
// Address of the token. Left-zero-padded if shorter than 32 bytes
TokenAddress [32]uint8
// Chain ID of the token
TokenChain uint16
// Number of decimals of the token
// (the native decimals, not truncated to 8)
Decimals uint8
// Symbol of the token (UTF-8)
Symbol [32]uint8
// Name of the token (UTF-8)
Name [32]uint8
```

RegisterChain:

```
// Gov Header
// Module Identifier  ("TokenBridge" left-padded)
Module [32]byte
// Governance Action ID (1 for RegisterChain)
Action uint8 = 1
// Target Chain (Where the governance action should be applied)
// (0 is a valid value for all chains)
ChainId uint16

// Packet
// Emitter Chain ID
EmitterChainID uint16
// Emitter address. Left-zero-padded if shorter than 32 bytes
EmitterAddress [32]uint8
```

UpgradeContract:

```
// Header
// Module Identifier  ("TokenBridge" left-padded)
Module [32]byte
// Governance Action ID (2 for UpgradeContract)
Action uint8 = 2
// Target Chain  (Where the governance action should be applied)
ChainId uint16

// Packet
// Address of the new contract
NewContract [32]uint8
```

RecoverChainId:

```
// Header
// Module Identifier  ("TokenBridge" left-padded)
Module [32]byte
// Governance Action ID (3 for RecoverChainId)
Action uint8 = 3

// Packet
// EVM chain ID of the forked chain. The contract MUST verify this
// matches its current `block.chainid` before applying the update.
EvmChainId uint256
// New Wormhole chain ID to set on the contract
NewChainId uint16
```

This action is only valid on a forked EVM chain. Unlike other governance messages, the payload is not targeted by Wormhole `ChainId` (since that is the value being recovered); instead, the contract requires `EvmChainId` to equal `block.chainid` so that the message can only be executed on the intended fork. On execution, the contract updates both its stored `evmChainId` and `chainId`.

SetPauserAddresses:

```
// Header
// Module Identifier  ("TokenBridge" left-padded)
Module [32]byte
// Governance Action ID (4 for SetPauserAddresses)
Action uint8 = 4
// Target Chain (Where the governance action should be applied)
ChainId uint16

// Packet
// Length of the pauser address. Must equal the target chain's native
// address size (e.g. 20 on EVM, 32 on Solana), or 0 to leave the role
// unassigned. The receiver rejects any other length. An all-zero
// address of the native size is equivalent to a zero length and is
// also treated as unassigned.
PauserLen uint8
// Address authorized to temporarily pause the bridge (for PAUSE_DURATION)
Pauser [PauserLen]uint8
// Length of the freezer address. Must equal the target chain's native
// address size, or 0 to leave the role unassigned. An all-zero address
// of the native size is also treated as unassigned.
FreezerLen uint8
// Address authorized to pause the bridge for the maximum duration
Freezer [FreezerLen]uint8
// Length of the unpauser address. Must equal the target chain's
// native address size, or 0 to leave the role unassigned. An all-zero
// address of the native size is also treated as unassigned.
UnpauserLen uint8
// Address authorized to unpause the bridge
Unpauser [UnpauserLen]uint8
```

Implementations MUST perform length-based validation for `SetPauserAddresses` on the target runtime, ensuring each address is an expected length (e.g. 20 bytes for EVM, 32 bytes for SVM) and that there are no remaining bytes after parsing the three addresses.

Existing deployments that are upgrading to include pausing functionality MUST ensure the new pause state initializes to `false`, `pauseExpiry` initializes to zero, and neither aliases non-zero pre-existing storage/account bytes.

## Caveats

### Transfer completion

A user who initiated a transfer should call `completeTransfer` within 24 hours. Guardian Sets are guaranteed to be valid for at least 24 hours. If the user waits longer, it could be that the Guardian Set has changed between the time where the transfer was initiated and the the time the user attempts to redeem the VAA. Let's call the Guardian Set at the time of signing `setA` and the Guardian Set at the time of redeeming on the target chain `setB`.

If `setA != setB` and more than 24 hours have passed, there are multiple options for still redeeming the VAA on the target chain:

1. The quorum of Guardians that signed the VAA may still be part of `setB`. In this case, the VAA merely needs to be modified to have the new Guardian Set Index along with any `setA` only guardian signatures removed to make a valid VAA. The updated VAA can then be be redeemed. The typescript sdk includes a [`repairVaa()`](../sdk/js/src/utils/repairVaa.ts) function to perform this automatically.
2. The intersection between `setA` and `setB` is greater than 2/3 of `setB`, but not all signatures of the VAA are from Guardians in `setB`. Then it may be possible to gather signatures from the other Guardians from other sources. E.g. Wormholescan provides an API under (/api/v1/observations/:chain/:emitter/:sequence)[https://docs.wormholescan.io/#/Wormscan/find-observations-by-sequence].
3. A Guardian may send a signed re-observation request to the network using the `send-observation-request` admin command. A new valid VAA with an updated Guardian Set Index is generated once enough Guardians have re-observed the old message. Note that this is only possible if a quorum of Guardians is running archive nodes that still include this transaction.

### Setup of wrapped assets

Since there is no way for a token bridge endpoint to know which other chain already has wrapped assets set up for the
native asset on its chain, there may be transfers initiated for assets that don't have wrapped assets set up yet on the
target chain. However, the transfer will become executable once the wrapped asset is set up (which can be done any time).

The name and symbol fields of the Transfer payload are not guaranteed to be valid UTF8 strings.
Implementations might truncate longer strings at the 32 byte mark, which may result in invalid UTF8 bytes at the end.
Thus, any client wishing to present these as strings must validate them first, potentially dropping the garbage at the end.

### Backwards compatibility of the pause check

Adding the `paused` check is interface backwards compatible on EVM and Solana. On Solana, the bridge already passes the `config` account on all relevant instructions, so no client-side changes are required to begin enforcing the new check.

However, deploying pause support updates existing contract state. For example, the SVM implementation resizes account data and there is no built-in rollback path to the exact pre-upgrade state. This should not break existing deployments, but integrators, SDKs, and off-chain indexers that assume the previous storage layout or account size may need to account for the new pause fields.

## Alternatives Considered

### Granular pause

Rather than a single boolean `paused` state, the bridge could expose finer-grained pauses (e.g. inbound vs. outbound) so that, for example, redemptions can continue while new deposits are blocked. We are not taking this approach now in favor of a simple boolean; granular pause can be added later without breaking the pause/unpause governance defined here.

### Pause state as a pure timestamp

The temporary pause could be represented solely by a timestamp - with no boolean - where every entry point treats the contract as paused while `block.timestamp < pauseExpiry`. This was rejected in favor of the boolean-plus-expiry design for two reasons:

1. It would make the contract become unpaused silently the instant the timestamp passes, creating a risk that the bridge resumes prematurely - for example mid-incident - without any explicit action or event. Keeping `paused` boolean and authoritative means a pause is only lifted by an explicit `unpause` or `unpauseExpired` call, each of which emits an event.
2. It would add a timestamp comparison to the hot path of every entry point. Retaining a boolean preserves the cheap single-boolean check on every call, while `pauseExpiry` is only consulted by the permissionless `unpauseExpired` entry point.

### Per-runtime `SetPauserAddresses` actions

`SetPauserAddresses` could use a separate governance action per runtime (e.g. action 4 for 20-byte EVM addresses, action 5 for 32-byte Solana pubkeys) instead of a single length-prefixed encoding. This was rejected for two reasons:

1. The fixed 32-byte left-padded encoding used by `RegisterChain.EmitterAddress` does not generalize to runtimes whose native addresses exceed 32 bytes (e.g. NEAR account IDs, Stacks contract principals, Cosmos bech32 strings). Length-prefix supports any address size up to 255 bytes without requiring a new action ID per runtime.
2. The safety property of "the receiver only accepts addresses well-formed for its runtime" is preserved either way - with per-runtime actions, the receiver checks the action ID; with length-prefix, the receiver checks the length against its native address size. In both cases the off-chain encoder must know the target chain's address format, so the wire format is the only thing that differs.

<!-- Local Variables: -->
<!-- fill-column: 120 -->
<!-- End: -->
