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

* Allow transfer of standards-compliant tokens between chains.
* Allow creation of wrapped assets.
* Use a universal token representation that is compatible with most VM data types.
* Allow domain-specific payload to be transferred along the token, enabling
  tight integration with smart contracts on the target chain.

## Non-Goals

* Support fee-burning / rebasing / non-standard tokens.
* Manage chain-specific token metadata that isn't broadly applicable to all chains.
* Automatically relay token transfer messages to the target chain.

## Overview

On each chain of the token bridge network there will be a token bridge endpoint program.

These programs will manage authorization of payloads (emitter filtering), wrapped representations of foreign chain
tokens ("Wrapped Assets") and custody locked tokens.

## Detailed Design

For outbound transfers, the contracts will have a lock method that either locks up a native token and produces a
respective Transfer message that is posted to Wormhole, or burns a wrapped token and produces/posts said message.

For inbound transfers they can consume, verify and process Wormhole messages containing a token bridge payload.

There will be five different payloads:

* `Transfer` - Will trigger the release of locked tokens or minting of wrapped tokens.
* `TransferWithPayload` - Will trigger the release of locked tokens or minting of wrapped tokens, with additional domain-specific payload.
* `AssetMeta` - Attests asset metadata (required before the first transfer).
* `RegisterChain` - Register the token bridge contract (emitter address) for a foreign chain.
* `UpgradeContract` - Upgrade the contract.

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
`TransferWithPayload`) message to Wormhole.  Once the message has been signed by the guardians, it can be posted to the
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

<!-- Local Variables: -->
<!-- fill-column: 120 -->
<!-- End: -->
