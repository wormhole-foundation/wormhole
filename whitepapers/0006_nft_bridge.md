# NFT bridge App

[TOC]

## Objective

To use the Wormhole message passing protocol to transfer NFTs between different connected chains.

## Background

NFTs are a new asset class that has grown in popularity recently. It especially attracts new users and companies to
crypto. NFTs, just like traditional tokens, are minted on a single blockchain and cannot be transferred to other chains.
Howevre as more chains introduce NFT standards and marketplaces there is demand for ways to transfer NFTS across chains
to access these markets and collect them in a single wallet.

## Goals

We want to implement a generalized NFT bridge using the Wormhole message passing protocol that is able to bridge any
standards-compliant NFT between chains, creating unique wrapped representations on each connected chain on demand.

* Allow transfer of standards-compliant NFTs between chains.
* Use a universal NFT representation that is compatible with most VM data types.

## Non-Goals

* Support EIP1155
* Manage / Transfer chain-specific NFT metadata that isn't broadly applicable to all chains.
* Automatically relay NFT transfer messages to the target chain.

## Overview

On each chain of the NFT bridge network there will be a NFT bridge endpoint program.

These programs will manage authorization of payloads (emitter filtering), wrapped representations of foreign chain
NFTs ("Wrapped NFTs") and custody locked NFTs.

We aim to support:

- EIP721 with token_uri extension: Ethereum, BSC
- Metaplex SPL Meta: Solana
- CW721 with token_uri extension: Terra

## Detailed Design

For outbound transfers, the contracts will have a lock method that either locks up a native NFT and produces a
respective Transfer message that is posted to Wormhole, or burns a wrapped NFT and produces/posts said message.

For inbound transfers they can consume, verify and process Wormhole messages containing a NFT bridge payload.

There will be three different payloads:

* Transfer - Will trigger the release of locked NFTs or minting of wrapped NFTs.

Identical to the NFT bridge:

* RegisterChain - Register the NFT bridge contract (emitter address) for a foreign chain.
* UpgradeContract - Upgrade the contract.

In order to transfer an NFT to another chain, a user needs to call the transfer method of the bridge contract with the
recipient details. The contract will either hold the NFTs in a custody account (in case it is a native NFT) or burn
wrapped NFTs. Wrapped NFTs can be burned because they can be freely minted once they are transferred back. After the
lockup the contract will post a Transfer payload message to Wormhole. Once the message has been signed by the guardians,
it can be posted to the target chain of the transfer. The target chain will then either release the native NFT from
custody or mint a wrapped NFT depending on whether it's a native NFT there. The program will keep track of consumed
message digests for replay prevention.

Since the method for posting a VAA to the NFT bridge is authorized by the message signature itself, anyone can post any
message.

Since every NFT has unique metadata the Transfer messages contain all metadata, a transfer (even the first on per NFT)
only requires a single Wormhole message to be passed compared to the Token Bridge. On the first transfer action of an
NFT (address / symbol / name) a wrapped asset (i.e. master edition or new contract) is created. When the wrapped asset (
contract) is already initialized or was just initialized, the (new) token_id and metadata URI are registered.

### API / database schema

Proposed bridge interface:

transfer(address token, uint256 token_id, uint16 recipient_chain, bytes32 recipient) - Initiate a Transfer

completeTransfer(Message transfer) - Execute a Transfer message

registerChain(Message registerChain) - Execute a RegisterChain governance message

upgrade(Message upgrade) - Execute a UpgradeContract governance message

---
Payloads:

Transfer:

```
PayloadID uint8 = 1
// Address of the NFT. Left-zero-padded if shorter than 32 bytes
NFTAddress [32]uint8
// Chain ID of the NFT
NFTChain uint16
// Symbol of the NFT
Symbol [32]uint8
// Name of the NFT
Name [32]uint8
// ID of the token (big-endian uint256)
TokenID [32]uint8
// URI of the NFT. Valid utf8 string, maximum 200 bytes.
URILength u8
URI [n]uint8
// Address of the recipient. Left-zero-padded if shorter than 32 bytes
To [32]uint8
// Chain ID of the recipient
ToChain uint16
```

RegisterChain:

```
PayloadID uint8 = 2
// Chain ID
ChainID uint16
// Emitter address. Left-zero-padded if shorter than 32 bytes
EmitterAddress [32]uint8
```

UpgradeContract:

```
PayloadID uint8 = 3
// Address of the new contract
NewContract [32]uint8
```

## Caveats

There is no guarantee for completion of transfers. If a user initiates a transfer and doesn't call completeTransfer on
the target chain, a transfer might not be completed. In case a guardian set change happens in-between and the original
signer guardian set expires, the transfer will be stuck indefinitely.

Since there is no way for a NFT bridge endpoint to know which other chain already has wrapped assets set up for the
native asset on its chain, there may be transfers initiated for assets that don't have wrapped assets set up yet on the
target chain. However, the transfer will become executable once the wrapped asset is set up (which can be done any time)
.

The name and symbol fields of the Transfer payload are not guaranteed to be
valid UTF8 strings. Implementations might truncate longer strings at the 32 byte
mark, which may result in invalid UTF8 bytes at the end. Thus, any client
whishing to present these as strings must validate them first, potentially
dropping the garbage at the end.

Currently Solana only supports u64 token ids which is incompatible with Ethereum which specifically mentions the use of
UUIDs as token ids (utilizing all bytes of the uint256). There will either need to be a mechanism to translate ids i.e.
a map of `[32]u8 -> incrementing_u64` (in the expectation there will never be more than MaxU64 editions) or Solana needs
to change their NFT contract.

Terra CW721 contracts support arbitrary strings as token IDs. In order to fit
them into 32 bytes, we store their keccak256 hash instead. This means that when
transferring a terra-native NFT through the wormhole, the ID of the output token
will be the original token's hash. However, wrapped assets on terra will retain
their original token ids, simply stringified into a decimal number. Then,
when transferring them back through the wormhole, we can guarantee that these
ids will parse as a uint256.
