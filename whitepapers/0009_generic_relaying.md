# Generic Message Relaying

[TOC]

## Objective

Designing a generalized protocol to allow a message sender to specify a destination for a Wormhole message and attach a
reward such that there is an incentive for a 3rd party to deliver it.

## Background

Wormhole messages are currently only proving someone has published a specific piece of information, a message, on a
chain. This message can be verified in all places that run a Wormhole verifier that is tracking the state of the
network (i.e. guardians). That makes messages "anycast" in the sense that there is no intended recipient specified in
the Wormhole message frame. Protocols like the token bridge encode the intended recipient in the message payload however
in order to implement an off-chain piece of software that would automatically deliver the message to the target would
need to be built to specifically parse a particular protocol's payloads. Also incentivizing the "relayer" would need to
be done at the application level which increases the implementation and operations effort for people building on top of
Wormhole.

## Goals

There needs to be a higher level protocol on top of wormhole that encapsulates the application payload and adds a
delivery-specific protocol frame including intended recipient and incentives.

* Provide a simple interface to use a universal delivery layer (at least once)
* Allow smart contracts to implement the `WormholeReceiver` interface to start receiving messages
* Gas-effective incentivization mechanism for message delivery

## Non-Goals

* Support delivery on Solana and chains that can't use a generic receiver interface
* Delivery-confirmation
* Exactly-once delivery
* Support fee coins other than the native currency of the chain

## Overview

On each chain of the delivery network there will be a delivery bridge endpoint program.

These programs will manage authorization of payloads (emitter filtering) and publishing of messages.

The protocol wraps the emitted message in a frame encoding the intended recipient of the protocol and delivery fee
attached. Receivers of Wormhole messages are expected to implement a receiver interface which is universal and the
off-chain relayers know how to interact with. This interface will interact with the delivery bridge endpoint for
verification of the VAA and fee accounting.

Fees are collected on the sending side and collected fees are tracked on the receiving side (per relayer). The relayer
can then claim fees in batch, requesting a payout on the receiving side and claiming them using the produced VAA on the
receiving side.

## Detailed Design

A proxy contract on each supported chain allows programs to emit wormhole messages with delivery information,
essentially wrapping it in a relayer protocol frame.

Relayers (running a piece of software supporting this protocol) will watch for messages emitted by the proxy contracts
and parse the frame. Then they will try to submit it to the target chain using the receiver interface. They will make
sure that the attached delivery reward/fee is sufficient to cover their cost.

Ideally senders will have an API to estimate gas costs of message delivery using a universal API that simulates the
delivery of the produced VAA on the target chain (estimating gas). This will require a more complex simulation logic
that can skip signature verification in the contract (replacing it with a stub; here we need to make sure to still
account for verification gas which is almost constant) as otherwise, due to the lack of a valid VAA already existing,
the message would not execute.

When a message is published the sender can attach a fee in a coin. This coin is going to be held in escrow by the proxy
contract on the sending side. The receiving side will do replay protection on wrapped message delivery and track the
accrued "virtual" rewards per relayer. So whenever a relayer delivers a message, the contract tracks the fee and adds it
to their account.

Whenever the relayer wants to claim the rewards, they can request a payout on the receiver side which will emit a
Wormhole message that can be used to claim the accrued rewards out of escrow on the sending side. This allows gas
efficient fees as no coins are transferred on delivery. Ideally the native coin is used instead of ERC20s on EVM chains
to further reduce the cost of sending the message.

Since anyone can use Wormhole to publish messages that match the payload format of the relayer protocol, an
authorization payload needs to be implemented. This is done using an `(emitter_chain, emitter_address)` tuple. Every
endpoint of the bridge needs to know the addresses of the respective other endpoints on other chains. This registration
of bridge endpoints is implemented via `RegisterChain` where a `(chain_id, emitter_address)` tuple can be registered.
Only one endpoint can be registered per chain. Endpoints are immutable. This payload will only be accepted if the
emitter is the hardcoded governance contract.

### API / database schema

Proposed proxy interface:

`postMessage(bytes payload, u8 confirmations, u8 target_chain, [32]u8 target_address, Coins delivery_reward)` - Publish
a message
`verifyMessage(bytes vaa) -> ([32]u8 emitter, bytes payload)` - Verify a wrapped message and account fees

---

Receiver interface:

`receiveWormholeMessage(bytes vaa)` - Receive a wormhole message

---
Payloads:

WrappedMessage:

```
PayloadID uint8 = 1
// Address of the fee coin. Left-zero-padded if shorter than 32 bytes
FeeAddress uint256
// Amount of fee coin the relayer will receive
FeeAmount uint256
// Address of the emitter of the actual message. Left-zero-padded if shorter than 32 bytes
EmitterAddress [32]uint8
// Payload of the application
Payload []uint8
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

ClaimRewards:

```
PayloadID uint8 = 4
// Address of the reward coin. Left-zero-padded if shorter than 32 bytes
RewardAddress uint256
// Amount of reward coin the relayer will claim
RewardAmount uint256
```

## Caveats
