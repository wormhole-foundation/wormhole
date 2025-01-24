# Transfer Verifier

# Objective

Add a defense-in-depth mechanism that cross-references a message publication from a Core Bridge with a corresponding transfer into the Token Bridge.

# Background

Wormhole detects activity on a sender chain by watching for message publications on the core bridge. If an attacker
has a way to fraudulently emit message from the core bridge, the integrity of the system will be compromised. 

# Goals

- Provide another layer of defense for token transfers
- Allow for a flexible response in the Guardian when suspicious activity is detected

# Non-Goals

- Address any message publications or Token Bridge activity other than token transfers
- Check cross-chain invariants. This mechanism only checks that pre-conditions are met on the sender side.

# Overview

Users are able to send funds cross-chain by interacting with a Token Bridge contract deployed on the sending chain. When they
transfer funds into this contract, it will make a corresponding call to the Core Bridge contract on the same chain. The
Core Bridge contract will then make the details of the token transfer available via e.g. emitting a log. The Wormhole Guardians
run "watcher" software that observe this activity, parse and verify the data, and finally issue a transfer on the destination
chain to complete the cross-chain transfer.

The ability to spoof messages coming from the Core Bridge would pose a serious threat to the integrity of the system, as
it would trick the Guardians into minting or unlocking funds on the destination chain without a corresponding deposit on the source chain.

In order to mitigate this attack, the Transfer Verifier is designed to cross-reference core bridge messages against the Token Bridge's
activity. If there is no Token Bridge activity that matches the core bridge message, the Guardians will have the ability to
respond to a potentially fraudulent message, such as by dropping or delaying it.

# Detailed Design

The overview section described an abstract view of how the Token Bridge, core bridge, and Guardians interact. However,
different blockchain environments operate heterogeneously. For example, the EVM provides reliable logs in the form
of message receipts that can easily be verified. In contrast, Sui and Solana do not provide the same degree of introspection
on historical account states. As a result, the Transfer Verifier must be implemented in an ad-hoc way using the state
and tooling available to particular chains. RPC providers for different blockchains may prune state aggressively which
provides another limitation on the degree of confidence.

## Types of Implementations
Broadly, the Transfer Verifier for a chain can be thought of as "reliable" or "heuristic" based on whether or not
historical account data is easily accessible. For "reliable" implementations, the Guardian could drop
the message publication completely as it is guaranteed to be fraudulent. For "heuristic" implementations, the Guardian
could choose to delay the transfer, allowing time for the transaction to be manually triaged.

## General Process

Transfer Verifier
- Connect to the chain (using a WebSocket subscription, or else by polling)
- Monitor the Core Contract for Message Publications
- Filter the Message Publications for Token Transfers
- Examine Token Bridge activity, ensuring that at least as many funds were transferred into the Token Bridge as are encoded in the Core Bridge's message
- If the above is not true, log an error

Guardian
- If the Transfer Verifier reports an error, block the Message Publication if the implementation is "reliable", otherwise delay.

## Implementations

The initial implementations were tested for Ethereum and Sui.

### Ethereum

The EVM provides a Receipt containing detailed logs for all of the contracts that were interacted with during the transactions.
The Receipt can be parsed and filtered to isolate Token Bridge and Core Bridge activity and the details are precise. For these
reasons the Ethereum implementation can be considered "reliable". If the Transfer Verifier is enabled, the Guardians will not
publish Message Publications in violation of the invariants checked by the Transfer Verifier.

For EVM implementations, the contents of the [LogMessagePublished event](https://github.com/wormhole-foundation/wormhole/blob/ab34a049e55badc88f2fb1bd8ebd5e1043dcdb4a/ethereum/contracts/Implementation.sol#L12-L26)
can be observed and parsed.

### Sui

For Sui, [events emitting the WormholeMessage struct](https://github.com/wormhole-foundation/wormhole/blob/ab34a049e55badc88f2fb1bd8ebd5e1043dcdb4a/sui/wormhole/sources/publish_message.move#L138-L148) are analyzed.

There are a number of complications that arise when querying historical account data on Sui.

TODO: add details from Sui

# Rollout Considerations

Because the Transfer Verifier will be integrated with the Watcher code, bugs in its implementations could lead to messages
being missed. For this reason, the changes to the watcher code must be minimal, well-tested, and reversible. It should
be possible to disable the Transfer Verifier entirely or on a per-chain basis by a Guardian without the need for a 
new release.

The Transfer Verifier should be implemented in a standalone package and distributed with a CLI tool that allows users
to verify its accuracy. This would allow for isolated testing and development outside of the context of critical Guardian code.
It should be possible to run the standalone tool for long periods of time to ensure that the mechanism is reliable and does
not produce false positives.

# Security Considerations

TODO
