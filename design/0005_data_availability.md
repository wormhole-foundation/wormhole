# Signed message data availability

[TOC]

## Objective

To make signed messages available to Wormhole clients without relying on a connected chain.

## Background

A Wormhole workflow typically starts by having a user submit a transaction on any of the connected chains, which results in a message being posted on-chain. This message is then picked up and confirmed by the guardian network. Once enough guardians signed the observation, the resulting message - the signed VAA - needs to be posted to the target chain to complete the operation.

With the Wormhole v1 design, we use Solana for data availability. As soon as any guardian observes a VAA with sufficient signatures, it would race to submit it to Solana, where it would be stored in an account with a deterministic address. A client - usually a web wallet - would then simulate the same deterministic calculation, using its knowledge of the VAA's digest - and retrieve the VAA from its associated Solana account. The client then posts it to the target chain, with the fee paid by the user. On Solana, no client-side submission is necessary - the VAA is executed immediately when posted for data availability.

However, while this design worked great for v1 which was bridging only two chains, it's not optimal for v2:

- It adds an unnecessary point of failure. A Solana outage would also prevent token transfers between unrelated chains (it's called mainnet-beta for a reason!). Messages would also incur unnecessary extra latency waiting for Solana to include the transaction. At the time of writing, this is exacerbated by high mainnet-beta skip rate which causes a significant percentage of transactions to fail to be included within a reasonable interval.

- It puts extra strain on Solana's network. In particular, for messages *originating* from Solana, it would cause write amplification.

- The race mechanism would be too expensive at scale and hard to incentivize, since we cannot use preflight and nodes would pay for failed messages, likely unequally. A leader selection mechanism would have to be implemented.

- Guardian nodes are required to maintain sufficient SOL balance in their wallets and need to be compensated and incentivized to actually submit transactions.

- Reproducing the deterministic Solana account address is complex to do client-side.

Our data availability requirements do not actually require messages to be posted on a finalized chain - we simply used one for convenience. Anything that reliably shuttles bytes from the guardian p2p network to the client will do the trick, and we can replace on-chain storage by a better-suited custom mechanism.

## Goals

- The mechanism must enable any client - native, app or web - to wait for and retrieve the signed VAA message for a specific message, identified by its unique (chain, emitter, sequence) identifier.

- Signed VAAs must be available to clients and guardian nodes at least until the message was posted on the target chain. Ideally, our design would enable an optional full archive of signed VAAs to be maintained.

- Signed VAAs must be available in a decentralized fashion to at least every guardian node.

- The design's performance characteristics must be sufficient to persist and retrieve all signed VAAs from all networks within at most 100-200ms per message, with all networks publishing messages concurrently at full capacity.

## Non-Goals

- The design facilitates a relayer mechanism where an incentivized third party would submit the transaction to the target chain, but it does not specify how such a mechanism would be implemented.

- Designing an incentivization scheme to compel guardians to run a public API. We assume an effective incentive is provided for guardian nodes to run high-quality API endpoints.

- Discovery of public API endpoints by clients. We assume that a well-known set of load balanced API frontends will be documented and hardcoded by client applications.

- Design of supporting infrastructure (scaling, caching, load balancing, ...)

- Resolving the fee payer issue - users need existing tokens on the target chain in order to pay fees for the client-side message submission transaction. This is a problem for users who want to use a token bridge to get such native tokens in the first place.

## Overview

Instead of submitting signed VAAs to Solana, guardians instead broadcast them on the gossip network and persist the signed VAAs locally.

Guardians that failed to observe the message (and therefore cannot reconstruct the VAA) will verify the broadcasted signed VAA and persist it as if they had observed it.

A public API endpoint is added to guardiand, exposing an API which allows clients to retrieve the signed VAA for any (chain, emitter, sequence) tuple. Guardians can use this API to serve a public, load-balanced public service for web wallets and other clients to use.

If a node has no local copy of a requested VAA, it will broadcast a message to the gossip network, requesting retransmission. Any node that stores a copy of the requested VAA will rebroadcast it, allowing the requesting node to backfill the missing data and serve the client's request.

## Detailed Design

The current guardiand implementation never broadcasts the full, signed VAA on the gossip network - only signatures. Guardians then use their own message observations and the aggregated set of signatures to assemble a valid signed VAA locally. Once more than 2/3 of signatures are present, the VAA is valid and can be submitted to the target chain. Nodes that haven't observed the message due to issues with the connected chain nodes are unable to construct a full VAA and will eventually drop the aggregated set of signatures.

Depending on the order of receipt and network topology, the aggregated set of signatures seen when the 2/3+ threshold is crossed is different from each node, but each node's VAA digest will be identical.

In v1, a node would submit the VAA directly to Solana, with complex logic for fault tolerance and retries. The first signed VAA would "win" a race and be persisted on-chain as the canonical signed VAA for this message.

Instead, each node will now locally persist the full signed VAA and broadcast it to the gossip network, where it can be received both by guardian nodes and unprivileged nodes (like future relayer services) that joined the gossip network. If a guardian receives a VAA for a tuple it has no state for, it will verify the signature and persist it using the same logic - the first valid response to be received "wins".

Locally persisted state is crucial to maintain data availability across the network - it is used to serve API queries (if enabled) and rebroadcast signed VAAs to other guardians that missed them.

We can't rely on gossip to provide atomic or reliable broadcast - messages may be lost, or nodes may be down. We need to assume that nodes can and will lose all of their local state, and be down for maintenance, including nodes used to serve a public API. We therefore need a mechanism for API nodes to backfill missing data when such data is requested.

A new gossip message type will be implemented - `RetransmissionRequest`, specifying the tuple of the missing message - which is signed using the node's guardian private key. A node that receives such a request, signed by a guardian in the current guardian set, will look up the requested the message key in its local store and rebroadcast it using the signed VAA distribution mechanism described above.

We use the (chain, emitter, sequence) tuple as global identifier. The digest is not suitable as a global identifier, since it is not known at message publication time. Instead, all contracts make the sequence number available to the caller when publishing a message, which the caller then surfaces to the client. Chain and emitter address are static.

This design deprecates existing usage of the digest as primary global identifier for messages in log messages and other user- and operator-facing interfaces, aiding troubleshooting. This a presentation layer change only - the digest continues to be used for replay protection and signature verification.

The guardiand submission state machine will be refactored to maintain aggregation state for observed VAAs in the local key-value store, rather than in-memory. This has the nice side effect of removing the timeout for observed-but-incomplete VAAs, allowing them to be completed asynchronously when missing nodes are brought back up and use chain replay to catch up on missed blocks.

### Contracts

No changes are required to smart contracts.

## Alternatives considered

### Provider-side redundancy instead of retransmissions

Instead of implementing the retransmission mechanism, we could instead make it the API provider's responsibility to maintain a complete record of VAAs by running multiple nodes listening to the gossip network.

Nodes could do idempotent writes to a single shared K/V store (like Bigtable or Redis), doing fallthrough API requests against other nodes in the cluster, or retry on the LB level.

While such features will likely be implemented in the future to improve scalability, we decided to design and implement retransmissions first:

- We want to mitigate gossip message propagation issues or operational mistakes that could affect many nodes. With retransmissions, a message can be retrieved as long as at least one node has a copy.

- For decentralization reasons, it should be possible to serve a fully-functional public API using a single node without requiring complex external dependencies or multiple nodes in separate failure domains.

### Direct P2P connectivity

libp2p supports a WebRTC transport, which would - in theory - allow web wallets to directly join the guardian gossip network. However, we decided not to pursue this route:

- libp2p is very complex and it's not clear how well such an approach would scale. Debugging any scalability (or other) issue likely requires in-depth libp2p debugging, which we have no experience with. In comparison, the challenges that come with a traditional RPC scale-out approach are much better understood.

- The only available reference implementation is written in Node.js, which is [incompatible](https://github.com/libp2p/js-libp2p/issues/287) with our libp2p QUIC transport. We would either have to join the main IPFS gossip network to take advantage of existing WebRTC-capable nodes, which would be a performance and security concern, or add support for a compatible transport to guardiand and run separate bridge nodes.

- Clients would have to publish messages to the gossip network, and a complex spam prevention mechanism would be needed.

Directly connecting to the gossip network remains a possible design for future for non-web clients like relayers that can speak the native libp2p QUIC protocol. Even a future implementation using the WebRTC transport remains a plausible avenue for future development.

## Caveats

### Invalid requests to the API

Nodes can't know whether they missed a message, or if the client requested an invalid message. Such requests for random invalid tuples can be made easily and cheaply.

Nodes have to send gossip messages in these cases, which can present a denial of service risk. Operators might choose to run an internal redundancy layer (as described above) and only do gossip requests when the client completed a proof of work or when users request support in the rare case of lost messages.

### "Leechers"

We do not specify an explicit incentive for nodes to maintain historic state. If a large fraction of the network fails to properly persist local state (like by running in ephemeral containers), we risk relying on insufficiently small number of nodes to serve retransmissions to others.

We believe that this is not an issue at this time due to the small amount of storage required (~1KiB per VAA) and data loss will occur infrequently, like when a node fails. The retransmission protocol is much slower than listening to the initial gossip transmissions, providing little incentive for API operators to misuse the mechanism.

### Gossip performance

This proposal significantly increases the amount of data broadcasted via the gossip network (signed message broadcast and retransmissions), which may affect latency and throughput of signature broadcast.

Gossip performance may be insufficient to serve retransmission requests at high rates (see also: https://github.com/certusone/wormhole/issues/40).

### Decentralization concerns

Using Solana for data availability comes with a well-established ecosystem of RPC service providers. This design instead requires facilitating a new API provider ecosystem. If too few providers exist, or the ecosystem converges on a few large monopolists, it could cause unwanted centralization in an otherwise trustless, decentralized system.

We believe that our proposal instead improves decentralization: Solana RPC nodes are expensive and complex to operate due to Solana's very high throughput and the large amount of state it needs to handle. In contrast, the Wormhole data availability problem is trivial - all we need is eventual at-least-once delivery of small key-value pairs. Serving and scaling out Wormhole RPC nodes is a much easier task than running Solana validators, which means that a larger number of parties will be able to provide them. The design makes it very easy to run a public API using autoscaling cloud services backed by any distributed key-value store.

## Security Considerations

This proposal affects only data availability of data that was already validated and signed. If the in-band data availability mechanism fails, out-of-band methods can be used to ensure data availability (like manually fetching and posting signed VAAs).

### Guardian secret key usage

We introduce a second usage of the guardian signing key for signings payloads other than VAAs. This is new, and potentially dangerous. In order to prevent potential confused deputy vulnerabilities, we append a prefix when signing and verifying non-VAA messages (i.e `retransmission|<payload>`) to distinguish non-VAA key usage.

The same approach will be used when signing heartbeats (https://github.com/certusone/wormhole/issues/267).

### Byzantine fault tolerance

Wormhole is designed to tolerate malicious behavior of up to 1/3 of guardian nodes. We allow any guardian node to request retransmissions. A byzantine node could abuse this behavior by sending a very large number of requests, overwhelming the gossip network or the guardian nodes.

Rate-limiting and blacklisting mechanism can be implemented to enable guardians to respond to such attacks.

(this assumes that libp2p itself is safe against pubsub flooding by non-guardian nodes - this an open question tracked in https://github.com/certusone/wormhole/issues/22)
