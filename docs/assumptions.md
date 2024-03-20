# Security Assumptions

This page details various assumptions that Wormhole relies on for security and availability. Many of these are
universal assumptions that apply to various decentralized protocols.

This document assumes familiarity with Wormhole concepts like VAAs.

## Gossip network availability

Wormhole's peer-to-peer gossip network relies on the [go-libp2p](https://github.com/libp2p/go-libp2p) and
[go-libp2p-pubsub](https://github.com/libp2p/go-libp2p-pubsub) libraries. libp2p is a very popular library used by many
major decentralized networks like IPFS and Ethereum 2.0. Nevertheless, like any distributed protocol, it may be
susceptible to various denial-of-service attacks that may cause message loss or overwhelm individual nodes.

We do _not_ rely on libp2p for security, only for availability. libp2p's channels are encrypted and authenticated by
default, but we do not rely on that property. A compromise of libp2p transport security could, at worst, result in
denial of service attacks on the gossip network or individual nodes.

Gossip network unavailability can result in missing events, but never permanently. Nodes will periodically
attempt to retransmit signatures for VAAs which failed to reach consensus in order to mitigate short-term
network outages. Longer network outages, leading to timeouts, and correlated crashes of a superminority of
nodes may result in observations being dropped.

The mitigation for this is a polling control loop in the case of Solana or chain replay for other chains. On Solana, the
node will consistently poll for unprocessed observations, resulting in re-observation by nodes and another round of
consensus. During chain replay, nodes will re-process events from connected chains up from a given block height, check
whether a VAA has already been submitted to Solana, and initiate a round of consensus for missed observations.

This carries no risk and can be done any number of times. VAAs are fully deterministic and idempotent - any
given observation will always result in the same VAA body hash. All connected chains keep a permanent record
of whether a given VAA body - identified by its hash - has already been executed, therefore, VAAs can safely
undergo multiple rounds of consensus until they are executed on all chains.

The bridge does not yet implement chain replay (see https://github.com/wormhole-foundation/wormhole/issues/123). Network outages
can therefore result in missed observations from chains other than Solana in the case of a prolonged network outage. It
will be possible to retroactively replay blocks after chain replay has been implemented to catch up on missed events.

## Chain consistency and finality

The Wormhole network always observes _external events_ and never initiates them on its own. It relies on the connected
chain's consensus, security and finality properties. In the case of guardian set updates, it relies on off-chain
operator consensus in the same way.

A non-exhaustive list of external chain properties Wormhole relies on:

- It can be assumed that at some point, transactions are final and cannot be rolled back.
- A given transaction is only included/executed once in a single block, resulting in a deterministic VAA body.
- Account data and state is permanent, by default or through a mechanism like Solana's rent exemptions.
- No equivocation - there is only one valid block at a given height.

## On-chain spam prevention

We assume that all connected chains use a fee or similar mechanism to prevent an attacker from overwhelming the network,
and that Wormhole's processing capacity is greater than the sum of the capacity of all connected chains.

Solana has ridiculous processing capacity and can process transactions at a greater rate than what its websocket
subscription interface, the agent, or the Wormhole itself could handle. This is partially mitigated by the fee that the
Wormhole contracts charge in excess of the (very cheap) transaction fee, but a sufficiently incentivized attacker could
still execute a sustained attack by simply paying said fee.

A possible future improvement would be dynamic fees on the Solana side, but this is currently blocked by runtime
limitations (see https://github.com/wormhole-foundation/wormhole/issues/125). Even with dynamic fees, raising the fees beyond the
amount that a reasonable user would pay may already constitute a successful attack against the protocol.

DDoS attacks on decentralized protocols are a tricky thing in general, and mostly a matter of game theory/incentives.
Defense strategies are dynamic and evolve as the ecosystem grows. We therefore exclude such attacks from the current
Wormhole threat model. The assumption is that the incentive to execute such an attack is less than the cost in fees and
the legal/liability risks an attacker would incur, and that the costs to sustain the attack would be greater than simply
attacking the connected chains directly.

## Guardian incentive alignment

Wormhole is a decentralized PoA bridge. Its game-theoretical security relies on hand-picked operators whose incentives
strongly align with the Solana ecosystem - large token holders, ecosystem projects, top validators and similar, who
would risk damage to their reputation, token values, and ecosystem growth by attacking the network or neglecting their
duties.

We assume that, at the present time, such incentive alignment is easier to bootstrap and get right than a separate chain,
which requires carefully-designed token economy and slashing criteria. In particular, it attracts operators who care
about the ecosystem beyond short-term validation rewards, resulting in a high-quality, resilient guardian set.

As the project grows, there's a number of potential improvements to consider other than a staking token, including
the [Balsa](https://docs.google.com/document/d/1sCgxHIOrVHAqrt4NWkUJXxQvpSxq6DyZrkf4IR-R-YM/edit) insurance pool
proposal, and a DAO that offsets operational costs and rewards operators.

## Uncompromised hosts

This should go without saying - in the context of a single node, we assume that an adversary cannot read or write host
memory, execute code, or otherwise compromise the running host operating system or platform while or after the node is
running. If a supermajority of nodes is compromised, an attacker can produce arbitrary VAAs. If a superminority of nodes
is compromised, the network may temporarily lose consensus (there's no way to intentionally void a guardian key or
prevent it from being replaced by the supermajority).

Contrary to popular belief, hardware security modules do _not_ significantly change the risks associated with host
compromise when dealing with cryptocurrency keys. A compromised host could easily abuse the HSM as a signing oracle,
causing irreversible damage with a single signature. It merely complicates the attack, but not in a major way.

For some use cases, like PoS validation, the risk of host compromise can be fully mitigated by running a smart HSM like
[SignOS](https://certus.one/sign-os). In these cases, the smart HSM can parse the signature payload and apply
constraints like "a given block height may only be signed once", which can be independently verified in a secure
enclave.

In the case of an oracle like Wormhole, this constraint is "only finalized events may be certified", which is impossible
to verify without verifying merkle proofs and syncing at least a sparse header chain. Therefore, in the case of
Wormhole, the entire Wormhole instance would have to run inside a smart HSM/SignOS, including light clients for the
chains it supports.

## Third-party libraries

Like any modern software project, we rely on a number of external libraries. We applied best practices in dealing with
such third party dependencies, including minimizing their number, avoiding binary dependencies, and using lockfiles to
pin dependencies to exact versions and hashes to avoid distribution-level compromises. We assume that the third-party
libraries we use are safe and do not contain backdoors.

Go's supply chain is particularly hardened against such compromises thanks to the [public go.sum
database](https://go.googlesource.com/proposal/+/master/design/25530-sumdb.md).

For cryptography in the node software, we exclusively rely on high-level interfaces in Go's standard library - which is
known for its robustness - and go-ethereum, both of which have been exhaustively audited.

## Safe handling of crashes in the Solana eBPF VM

Due to the instruction count limitations in the Solana runtime, the Solana contracts make liberal use of unsafe blocks
to serialize and deserialize data without incurring the overhead of a memory-safe approach.

This follows current best practices for Solana contract development. It assumes that invalid operations or out-of-bounds
accesses will always cause a crash and be caught by the bytecode interpreter, and safely halt contract execution like
any other error during contract execution would.
