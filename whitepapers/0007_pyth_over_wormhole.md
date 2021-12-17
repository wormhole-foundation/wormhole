# Pyth over Wormhole Design

Date Created: June 23, 2021

# Bridging Pyth data over Wormhole

## Objective

The implementation of Pyth data attestation over the Wormhole bridge.

## Background

### Solana

Solana is a general purpose proof-of-stake cryptocurrency system focused around scalable performance at a low transaction cost. It boasts a unique on-chain storage system that enables performant, parallel smart contract execution and disposal of unneeded stored content.

[Scalable Blockchain Infrastructure: Billions of transactions & counting](https://solana.com/)

### Pyth

The Pyth Network is a Solana-based market data oracle aimed at reliably providing prices of a collection of supported assets.

[Home](https://pyth.network/)

### Wormhole

Wormhole is an inter-blockchain communication system aiming to provide tamper-proof messaging functionality to a group of supported blockchains. With Wormhole, you can send data from chain A to chain B, where B is potentially any blockchain capable of basic smart contract functionality.

## Goals

- Outline of a process for reliable, secure transfers of Pyth asset price data from Solana to other blockchains supported by Wormhole.

## Non-Goals

- Wormhole design - while this effort will deliver many insights about Wormhole's usability, the correct place to address bridge-specific design challenges is Wormhole's own design documents.

## Overview

 TBD

## Detailed Design

### Existing Software intended for implementation
| Name                      | Description/Purpose                                                                                                                                                                                | Webpage                                           |
|---------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------|
| Rust                      | Safe, fast programming language supported for writing Solana contracts                                                                                                                             | https://www.rust-lang.org/                        |
| pyth-client-rs            | Rust library for reaching Pyth on Solana                                                                                                                                                           | https://github.com/pyth-network/pyth-client-rs    |
| Solana                    | The main codebase for Solana client libraries                                                                                                                                                      | https://github.com/solana-labs/solana             |
| Solitaire                 | A convenient Rust framework for contract development that greatly simplifies Solana's unique execution model; A by-product of Wormhole's Solana efforts, built on top of the official Solana APIs. | https://forge.certus.one/plugins/gitiles/wormhole |
| Wormhole's Solana program | The Solana end of Wormhole, built with Solitaire.                                                                                                                                                  | https://forge.certus.one/plugins/gitiles/wormhole |

### New Software components required for implementation
| Name                              | Type                                                  | Description                                                                                                                                                                                                                                                                                                                                        |
|-----------------------------------|-------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| pyth2wormhole                     | on-chain Solana program in Rust                       | When called, retrieves Pyth data for the specified asset and posts it to Wormhole's solana program.                                                                                                                                                                                                                                                |
| pyth2wormhole-client              | off-chain Rust program                                | When run, this off-chain program calls pyth2wormhole to forward the current price data of an asset to Wormhole.                                                                                                                                                                                                                                    |
| Pyth relayer software (name TBD?) | off-chain program                                     | When active, this program acts as a Wormhole relayer. A relayer listens to peer-to-peer Wormhole guardian messaging for data coming from the supported chains. When our relayer detects relevant Pyth price data, it will complete the bridging effort by relaying the price data to a specific smart contract on the requested target blockchain. |
| Target chain smart contracts      | on-chain smart contract for each desired target chain | When called, each target chain smart contract will verify the provided Pyth data and make it available to target chain users.                                                                                                                                                                                                                      |

### API / database schema

### A note on Pyth data
Pyth's storage model maintains a linked-list-style collection of
accounts that describe multiple assets. Each asset contains its own
list of price data of different types, which themselves contain prices
from various quoter accounts that name concrete numbers as current
price value. The individual source values are also available as a
single, aggregated number from the upper-level asset price
description.

In the current system metadata from arbitrary Pyth feeds can not be
trusted and anyone can create new feeds for popular symbols. Projects
that build products on top of Pyth need to maintain a list of the
product_ids (Solana account keys) that they trust.  As long as this is
the case we can also rely on projects to hardcode the relevant
metadata that they require in their business logic and focus this
initial release on just attesting pricefeeds.

### Message serialization conventions between pyth2wormhole and target chains
The structure proposed by Pyth employs deeply-nested storage
strategies specific to Solana and Pyth itself. In order to enable
interoperability with other chains, we propose a more general format.

#### Byte-level format
For byte-for-byte serialization we commit to tightly packed bytes in
big-endian byte order with two's complement representation for signed
integers.

#### Header
We use a "P2WH" as a magic byte string and a 2-byte unsigned version
field on all top-level messages. We expect to increase the version
value with subsequent versions of this format.

| Field Name          | Type    | Length in bytes | Description                                                                                           |
|---------------------|---------|-----------------|-------------------------------------------------------------------------------------------------------|
| magic               | bytes   | 4               | Constant 4 ASCII bytes "P2WH", not terminated; sanity check that payload is not bogus                 |
| version             | uint16  | 2               | Constant value: 1, version number of the format; bigger number means newer version                                       |

#### Nesting
Nested non-primitive data structures may choose not to include
magic/version/payload_id if the context of surrounding data structure
is deemed sufficient. Unless otherwise stated the header/no-header
choice is assumed to be consistent.

#### Variable-length data
In the event of variable-length data becoming part of the protocol in
the future, we defer to future message format designs for appropriate
length tracking. 

### Verifiable Messages
In the following subsections we outline typed schemas for the messages
forwarded from pyth2wormhole to target chain contracts. Each message,
after conversion to a stream of bytes (serialized as described above)
will cross the bridge and then be captured by the Pyth relayer for use
in target chain contracts.

#### PriceAttestation
Used for direct communication of tamper-proof product price updates.

| Field Name          | Type    | Length in bytes | Description                                                                                           |
|---------------------|---------|-----------------|-------------------------------------------------------------------------------------------------------|
| header              | Header  | 6               | See [Header](#Header)                                                                                 |
| payload_id          | bytes   | 1               | constant value: 1; Distinguishes price attestation from other messages                                |
| product_id          | bytes32 | 32              | solana account key of the product                                                                     |
| price_id            | bytes32 | 32              | solana account key of the price; used for disambiguation between different prices on a single product |
| price_type          | bytes   | 1               | Price `ptype` field in u8 representation                                                              |
| price               | int64   | 8               | PriceInfo `price` field                                                                               |
| expo                | int32   | 4               | Price `expo` field; denotes price value exponent                                                      |
| twap                | Ema     | 24              | Price `twap` field, see [Ema](#Ema)                                                                   |
| twac                | Ema     | 24              | Price `twac` field, see [Ema](#Ema)                                                                   |
| confidence_interval | uint64  | 8               | PriceInfo `conf` field.                                                                               |
| status              | bytes   | 1               | PriceInfo `status` field in u8 representation                                                         |
| corp_act            | bytes   | 1               | PriceInfo `corp_act` field in u8 representation                                                       |
| timestamp           | int64   | 8               | Unix timestamp from this price attestation; Based on Solana contract call time                        |

#### Ema
Nested field in [`PriceAttestation`](#PriceAttestation). Does not use
a header or payload ID. Modeled after the `Ema` structure
[upstream](https://github.com/pyth-network/pyth-client-rs/commit/05954af231f01d77e4bb2e153dd6a7829ae1ee57).

| Field Name | Type   | Length in bytes | Description                       |
|------------|--------|-----------------|-----------------------------------|
| val        | int64, | 8               | current value of ema              |
| numer      | int64, | 8               | numerator state for next update   |
| denom      | int64  | 8               | denominator state for next update |

### Solana pyth2wormhole Contract and client API
Pyth2wormhole's main goal is emission of **aggregate** price
attestations for consumption on each supported target chain. Component
prices are out of scope at the time of this writing.

#### Solana storage/ACL in a nutshell
We observe that expressing an API precisely in terms of Solana's
and Wormhole's idiomatic primitives may obstruct the view
of the intended scope of pyth2wormhole. Below we give a brief
explanation of the complexity. Please refer to the [code-level
docs](#detailed-code-level-documentation) for concrete implementation details.

Under Solana's current programming model all authorities, actors,
required privileged operations, transitive cross-call
actors/privileges etc. must be expressed at RPC call time using
[*accounts*](https://docs.solana.com/developing/programming-model/accounts).
The requirement enables strict access control and performance
optimizations of the chain.

Because of this constraint, numerous accounts necessary for a
transaction on Solana would not find a matching counterpart on other
chains. In a similar vein, it is common for a significant portion of
the account addresses to be *deduced* using Solana-specific
techniques, such as:
* well-known system addresses a.k.a. sysvars (privileges, chain
  metadata, cross-calls, rent metadata)
* deterministic public key derivation schemes (notably program-owned
  contract state and spontaneous storage in the spirit of events on
  Ethereum)

#### High level user API
* `attest(product_id, price_id)` - Attest the metadata of the
  specified price account.
* `last_attestation(product_id, price_id)` - Look up the last attested
  price data for a price account (if present); Free of charge
  off-chain.

#### Detailed Code level documentation
The Rust code documentation for pyth2wormhole can be found
[here (TBD, ideally rust's cargo-doc UI for pyth2wormhole crates)](https://example.com).

### Target chain contract APIs

Mandatory target chain interface:

`attestPrice(VM attestation)` returns `PriceAttestation` - Caches aggregated price attestations

`upgrade(VM upgrade)` - Execute a `UpgradeContract` governance message

**Getters:**

`latestAggregatePrice(bytes32 productId)` returns `PriceAttestation` - Returns latest cached aggregate price

### Payloads

### UpgradeContract
| Field Name  | Type    | Description                                     |
|-------------|---------|-------------------------------------------------|
| Module      | bytes32 | "Pyth" - left-padded                            |
| Action      | uint8   | 1 = contract upgrade                            |
| ChainId     | uint16  | chain id                                        |
| NewContract | bytes32 | address of the new implementation (left-padded) |
