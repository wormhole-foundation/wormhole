# Message Publishing

[TOC]

## Objective

To specify the mechanics and interfaces for publishing messages over Wormhole.

## Background

The original Generic Message Passing design doc describes the format of Wormhole messages, however, the way messages are
published needs to be defined clearly for chain endpoints to be built.

## Goals

* Specify an interface for posting messages.
* Specify a fee model for message posting.

## Non-Goals

* Dynamic spam protection.
* Distribution of fees.

## Overview

The core Wormhole contracts on every chain will have a method for posting messages to Wormhole, which will emit an event
that needs to be observable by guardians running a full node on a given chain.

The fees will be payable in the respective chain's native currency. Fees can be
claimed by the protocol and collected in a fee pool on Solana where they can be distributed according to protocol rules.

## Detailed Design

Wormhole core contracts have a `postMessage` method which can be used by EOAs (externally owned accounts) or SCs (smart
contracts)
to publish a message via Wormhole.

This method has to perform verification on the payload for the maximum size limitation of **750 bytes**. The message
should be emitted such that it can be picked up by guardians in a way that allows offline nodes to replay missed blocks.

The Wormhole contract will also need to make the emitter of the published message available to the guardians. The
emitter is either a parameter to the postMessage method if the chain allows proving that the caller controls or is
authorized by said address (i.e. Solana PDAs), or it is the sender of the transaction.

Additionally, the Wormhole contract will keep track of a sequence number per emitter that is incremented for each
message submitted.

The timestamp is derived by the guardian software using the finalized timestamp of the block the message was published
in.

When a message is posted, the emitter can specify for how many confirmations the guardians should wait before an
attestation is produced. This allows latency sensitive applications to make sacrifices on safety while critical
applications can sacrifice latency over safety. Chains with instant finality can omit the argument.

**Fees:**

In order to incentivize guardians and prevent spamming of the Wormhole network, publishing a message will require a fee
payment.

This fee is supposed to be paid in any of the chain's native fee currencies when publishing a message. This assumes that
anyone sending a transaction is already required to hold such assets in order to make the transaction publishing the
message, and that the fee will therefore not negatively affect usability of the bridge.

The fee is defined by governance using the `SetMessageFee` VAA. The fees set are denominated in the respective chains
native currency. Each chain's Wormhole program is supposed to use an on-chain price oracle (e.g. a uniswap pool TWAP or
Pyth price feed)
Fees are set per chain to allow the protocol to take into consideration the effort required to keep the chain's nodes
online and account for spam attacks.

Fees will be collected in a wallet that is controlled by the Wormhole contract, governance or a more automated mechanism
to be implemented in a later design doc will be able to produce a `TransferFees` which will allow to move the collected
fee tokens to a specified address. In case there is a widely accepted token bridge, this mechanism might be extended to
bridge tokens back to the chain where the governance and staking contracts are located for them to be distributed there.

### API / database schema

Proposed bridge interface:

`postMessage(bytes payload, u8 confirmations)` - Publish a message to be attested by Wormhole.

`setFees(VAA fee_payload)` - Update the fees using a `SetMessageFee` VAA

`transferFees(VAA transfer_payload)` - Transfer fees using a `TransferFees` VAA

---

**Payloads**:

The payloads follow the governance message format.

SetMessageFee:

```
// Core Wormhole Module
Module [32]byte = "Core"
// Action index (3 for Fee Update)
Action uint16 = 3
Chain uint16
// Message fee in the native token
Fee uint256
```

TransferFees:

```
// Core Wormhole Module
Module [32]byte = "Core"
// Action index (4 for Fee Transfer)
Action uint16 = 4
Chain uint16
// Amount being transferred (big-endian uint256)
Amount uint256
// Address of the recipient. Left-zero-padded if shorter than 32 bytes
To [32]uint8
```

## Caveats

A governance decision is required for the collection of fees. This means a lot of manual intervention in the
distribution of fees to stakers and guardians. The lack of a token bridge makes it hard to automate this in the early
days of the protocol. Also, a transfer primitive is unlikely to support token bridges (which may require smart contract
calls), so a contract will be required.

## Security Considerations
