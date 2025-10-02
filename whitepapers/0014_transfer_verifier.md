# Transfer Verifier

# Objective

Add a defense-in-depth mechanism that cross-references a message publication from a Core Bridge with a corresponding transfer into the Token Bridge.

# Background

Wormhole works by watching for message publications on Core Bridge contracts for supported chains. Message Publications
are created when a contract calls into the Core Bridge to trigger a state change in the Wormhole network.

Generally speaking, if an attacker has a way to fraudulently emit message from the Core Bridge, the integrity of the system will be compromised. 

In order to mitigate this risk for Token Transfers, it's possible to cross-reference the Core Bridge message publications with the
calling contract i.e. the corresponding Token Bridge for that chain.

# Goals

- Detect and respond to a compromise of the Core Bridge pertaining to publishing arbitrary token transfer messages.
- Provide another layer of defense for token transfers.
- Allow for a flexible response in the Guardian when suspicious activity is detected.

# Non-Goals

- Block, delay, or otherwise interfere with message processing. The Transfer Verifier reports whether a message looks
  safe or not, but it does not take any other action.
- Address any message publications or Token Bridge activity other than token transfers.
- Check cross-chain invariants. This mechanism only checks that pre-conditions are met on the sender side.
- Protect against RPC compromises. (The Transfer Verifier relies on reliable RPC information.)
- Protect against a compromise of the Token Bridge. (The Transfer Verifier checks for a potential _Core Bridge_ issue by querying the Token Bridge.)

# Overview

Users are able to send funds cross-chain by interacting with a Token Bridge contract deployed on the sending chain. When they
transfer funds into this contract, it will make a corresponding call to the Core Bridge contract on the same chain. The
Core Bridge contract will then make the details of the token transfer available via a message publishing process 
(e.g. emitting an event on EVM based chains). The Wormhole Guardians run "watcher" software that observe this activity, 
parse and verify the data, and finally issue a transfer on the destination chain to complete the cross-chain transfer.

The ability to spoof messages coming from the Core Bridge would pose a serious threat to the integrity of the system, as
it would trick the Guardians into minting or unlocking funds on the destination chain without a corresponding deposit on the source chain.

In order to mitigate this attack, a Transfer Verifier is designed to cross-reference Core Bridge messages against the Token Bridge's
activity. If there is no Token Bridge activity that matches the Core Bridge message, the Guardians will have the ability to
respond to a potentially fraudulent message, such as by dropping or delaying it.

The Transfer Verifier will not drop or delay messages itself. Another part of the node can respond as it sees fit.
For example, the processor could be extended to process messages differently based on the status a Transfer Verifier
gives them.

# Detailed Design

The overview section described an abstract view of how the Token Bridge, Core Bridge, and Guardians interact. However,
different blockchain environments operate heterogeneously. For example, the EVM provides strict logs in the form
of message receipts that can easily be verified. In contrast, Sui and Solana do not provide the same degree of introspection
on historical account states. As a result, a Transfer Verifier must be implemented in an ad-hoc way using the state
and tooling available to particular chains. RPC providers for different blockchains may prune state aggressively which
provides another limitation on the degree of confidence.

## Types of Implementations
Broadly, a Transfer Verifier for a chain can be thought of as "strict" or "heuristic" based on whether or not
historical account data is easily accessible. For "strict" implementations, the Guardian could drop
the message publication completely as it is guaranteed to be fraudulent. For "heuristic" implementations, the Guardian
could choose to delay the transfer, allowing time for the transaction to be manually triaged.

## General Process

- Connect to the chain (using a WebSocket subscription, or else by polling)
- Monitor the Core Contract for Message Publications
- Filter the Message Publications for Token Transfers
- Examine Token Bridge activity, ensuring that at least as many funds were
transferred into the Token Bridge as are encoded in the Core Bridge's message
- If the above is not true, report the Message Publication as having failed verification.

### Responding to suspicious messages
If  a Transfer Verifier reports an error the Guardian should block the Message
Publication if the implementation is "strict". If the implementation is
"heuristic", the message should be delayed. This will allow some time for core
contributors to analyze the message and associated logs for potential bugs or
malicious activity.

In order to handle messages flexibly, they can be tagged with a "verification
state". This state will be embedded in to the message itself to allow for
different parts of the message publication process within the node to handle
messages in specific ways. For example, an `Anomalous` message originating from
Sui could be treated in a different way than one coming from Solana. As chains
are developed, they may introduce other tools or logging artifacts that result
in a Transfer Verifier implementation to become more or less strict.

Some example "verification states" are:
- Not verified
- Not applicable
- Verified
- Anomalous
- Rejected
- Could not verify

Messages can have an initial state of `Not verified` which should eventually be changed to one of the other states.

Messages marked as `Not applicable` should be processed normally. This state
will be used for messages that are not Token Transfers, as only Token Transfers
are process by  a Transfer Verifier.

A `Verified` message is in a "known good" state and should be processed normally.

Both `Anomalous` and `Rejected` represent potentially dangerous states. `Anomalous` should be used for "heuristic"
Transfer Verifier implementations and `Rejected` should be used for "strict" implementations. `Rejected` messages
must be dropped and must not result in a VAA creation for the message. `Anomalous` messages can be delayed, though
other potential responses may be appropriate depending on the reliability of  a Transfer Verifier implementation
and how likely `Anomalous` messages are to occur in practice.

`Could not verify` refers to transactions that could be correctly parsed for whatever reason. This state signals
that an attempt was made to verify a transfer but that it could not be completed successfully.

### Sequencing with the Governor and the Accountant

Assuming both are enabled, the processor first sends a message to the Governor and then to the Accountant once the
message is released by the Governor.

Checks relating to  a Transfer Verifier should occur before the checks to the Governor or the Accountant. If a message
is `Rejected` or `Anomalous`, it should be dropped or delayed before it is included in the Governor or Accountant.
This prevents a scenario where a `Rejected` message might still consume outgoing Governor capacity, for example.

## Implementations

### Ethereum

The EVM provides a Receipt containing detailed logs for all of the contracts that were interacted with during the transactions.
The Receipt can be parsed and filtered to isolate Token Bridge and Core Bridge activity and the details are precise. For these
reasons the Ethereum implementation can be considered "strict". If  a Transfer Verifier is enabled, the Guardians will not
publish Message Publications in violation of the invariants checked by  a Transfer Verifier.

For EVM implementations, the contents of the [LogMessagePublished event](https://github.com/wormhole-foundation/wormhole/blob/ab34a049e55badc88f2fb1bd8ebd5e1043dcdb4a/ethereum/contracts/Implementation.sol#L12-L26)
can be observed and parsed.

# Deployment Considerations

## Modularity
Because  a Transfer Verifier will be integrated with the Watcher code, bugs in its implementations could lead to messages
being missed. For this reason, the changes to the watcher code must be minimal, well-tested, and reversible. It should
be possible to disable a Transfer Verifier entirely or on a per-chain basis by a Guardian without the need for a 
new release.

## Testing
The Transfer Verifier should be implemented in a standalone package and distributed with a CLI tool that allows users
to verify its accuracy. This would allow for isolated testing and development outside of the context of critical Guardian code.
It should be possible to run the standalone tool for long periods of time to ensure that the mechanism is strict and does
not produce false positives.

# Security Considerations

## Interactions with the Governor

The idea of delaying unusual transfers overlaps to some extent with the Governor's delay mechanism. It is tempting to
leverage this existing mechanism, e.g. by adding a suspicious transfer to the Governor's pending list. 

There are some complexities to consider if this route is chosen, outlined below.

### Multiple Delays
If a Transfer Verifier result causes a message to be delayed, this should be considered against other delays that may
be introduced. For example, a suspicious message on a governed chain with a notional value beyond the Governor's "big
transaction limit" may result in a message being delayed twice.

### Flow Cancel can release transfers early

The Flow Cancel mechanism can result in the "daily limit" of the Governor being reduced under specific conditions.
If suspicious transfers are queued alongside regular transfers, and Flow
Cancel is enabled, the suspicious transfers may become released early.

It should be noted that this only affects small transfers, as Flow Cancel does not affect transfers over the big
transfer limit.

More details about the Flow Cancel feature can be found in the [Governor whitepaper](./0007_governor.md).
