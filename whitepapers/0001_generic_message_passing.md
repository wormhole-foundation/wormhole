# Generic Message Passing

## Objective

To refactor Wormhole into a fully generic cross chain messaging protocol and remove any application-specific
functionality from the core protocol.

## Background

Wormhole was originally designed to support a very specific kind of cross-chain message passing - token wrapping/swaps
between Solana and Ethereum. Read more about the original design and its goals in
the [announcement blog post](https://medium.com/certus-one/introducing-the-wormhole-bridge-24911b7335f7) and
the [protocol documentation](https://github.com/wormhole-foundation/wormhole/blob/48b3c0a3f8b35818952f61c38d89850eb8924b55/docs/protocol.md)

Since then, it has become clear that there is strong demand for using Wormhole's simple cross-chain state attestation
model for applications beyond its original design. This includes third-party projects wanting to transfer tokens other
than ERC20 (like NFTs), transfers guaranteed by insurance pools, "slow path/fast path" designs, as well as entirely
different use cases like oracles broadcasting arbitrary data to multiple chains.

Enabling these use cases requires extending Wormhole to provide a generic set of APIs and design patterns, decoupling it
from the application logic.

The core problem that both the current and future Wormhole design is solving is that of **enabling contracts on one
chain to verify messages from a different chain**. Smart contract engines on chains are often insufficiently powerful to
independently verify expensive state proofs from other chains due to the amount of storage and compute required. They
therefore need to rely on off-chain oracles to observe and verify messages and then re-sign them such that they _can_ be
verified on any of the connected chains, by trusting the oracle network as an intermediary rather than trusting the
remote chain.

We previously designed a similar protocol extension for the current Wormhole design, called EE-VAAs, which is the
precursor to this fully generic design:

- [External Entity VAAs](https://github.com/wormhole-foundation/wormhole/issues/147)
- [External Entity: Account State Attestation](https://github.com/wormhole-foundation/wormhole/issues/149)
- [External Entity: Relayer mode](https://github.com/wormhole-foundation/wormhole/issues/150)

This design doc assumes basic familiarity with the current design of Wormhole.

## Goals

We want to enable a wider range of both 1:1 and 1:n messaging applications to be built on Wormhole without requiring
changes to the core protocol for each new use case. Some examples of such applications that third parties could build:

- Unicast messaging between two specific contracts on different chains (example: token or NFT swaps).

- Multicast from a single chain to a specific set of connected chains (example: relaying data published by price oracles
  like Pyth or Chainlink).

The goal is to **redesign the protocol such that it is fully decoupled from the application logic**. This means that
Wormhole will no longer hold assets in custody or interact with any tokens other than providing the low-level protocol
which protocols interacting with tokens could be built on top of. This includes message delivery - Wormhole's current
design directly delivered messages to a target contract for some chains. With a generic protocol, the delivery mechanism
can wildly differ between different use cases.

## Non-Goals

This design document focuses only on the mechanics of the message passing protocol and does not attempt to solve the
following problems, leaving them for future design iterations:

- The specifics of implementing applications, other than ensuring we provide the right APIs.

- Data availability/persistence. Delivering the signed message to the target chain is up to the
  individual application. Possible implementations include client-side message retrieval and submission, like the
  current Wormhole implementation does for delivering transfer messages on Ethereum, or message relays.

- The mechanics of economically incentivizing nodes to maintain uptime and not to censor or forge messages.

- Governance and criteria for inclusion in the guardian set. We only specify the governance API without defining its
  implementation, which could be a smart contract on one of the connected chains.

## Overview

We simplify the design of Wormhole to only provide generic **signed attestations of chain state**.
Attestations can be requested by any contract by publishing a message, which is then picked up and signed by the
Wormhole guardian set. The signed attestation will be published on the Wormhole P2P network.

Delivering the message to a contract on the target chain is shifted to the higher-layer protocol.

## Detailed Design

The following defines the generic VAA version 1 struct:

```go
// VAA is a verifiable action approval of the Wormhole protocol.
// It represents a message observation made by the Wormhole network.
VAA struct {
	// --------------------------------------------------------------------
	// HEADER - these values are not part of the observation and instead
	// carry metadata used to interpret the observation. It is not signed.

	// Protocol version of the entire VAA.
	Version uint8 = 1

	// GuardianSetIndex is the index of the guardian set that signed this VAA.
	// Signatures are verified against the public keys in the guardian set.
	GuardianSetIndex uint32

	// Number of signatures included in this VAA
	LenSignatures uint8

	// Signatures contain a list of signatures made by the guardian set.
	Signatures []*Signature

	// --------------------------------------------------------------------
	// BODY - these fields are *deterministically* set by the
	// Guardian nodes when making an observation. They uniquely identify
	// a message and are used for replay protection.
	//
	// Any given message MUST NEVER result in two different VAAs.
	//
	// These fields are part of the signed digest.

	// Timestamp, in seconds, of the observed message.
	// This timestamp is derived from the block, rather than the
	// time the block was seen by the guardians.
	Timestamp time.Time // uint32

	// Nonce (provided by the on-chain integrator).
	Nonce uint32 // <-- NEW

	// EmitterChain the VAA was emitted on. Set by the guardian node
	// according to which chain it received the message from.
	EmitterChain ChainID // <-- NEW

	// EmitterAddress of the contract that emitted the message. Set by
	// the core contract and read by guardian node according to protocol
	// metadata.
	EmitterAddress Address // <-- NEW

	// Sequence number of the message. Automatically set and
	// and incremented by the core contract when called by
	// an emitter contract.
	//
	// Tracked per (EmitterChain, EmitterAddress) tuple.
	Sequence uint64 // <-- NEW

	// Level of consistency requested by the emitter.
	//
	// The semantic meaning of this field is specific to the emitter
	// chain. See Consistency Levels below.
	ConsistencyLevel uint8 // <-- NEW

	// Payload of the message (provided by the on-chain integrator).
	Payload []byte // <-- NEW
}

// ChainID of a Wormhole chain. These are defined in the guardian node
// for each chain it talks to.
ChainID uint16

// Address is a Wormhole protocol address. It contains the native chain's address.
// If the address data type of a chain is < 32 bytes, the value is zero-padded on the left.
Address [32]byte

// Signature of a single guardian.
Signature struct {
	// Index of the validator in the guardian set.
	Index uint8
	// Signature bytes.
	Signature [65]byte
}
```

The previous `Payload` method and `BodyTransfer`/`BodyGuardianSetUpdate`/`BodyContractUpgrade` structs with fields
like `TargetChain`, `TargetAddress`, `Asset`and `Amount` will be removed and replaced by top-level `EmitterChain`
and `EmitterAddress` fields and an unstructured `Payload` blob. To allow for ordering on the receiving end, `Sequence`
was added which is a message counter tracked per emitter.

Notably, we remove target chain semantics, leaving it as an implementation detail for a higher-level relayer protocol.

Guardian set updates and contract upgrades will still be handled and special-cased at the Wormhole contract layer.
Instead of specifying a VAA payload type like we previously did, Wormhole contracts will instead be initialized with a
specific well-known `EmitterChain` and `EmitterAddress` tuple which is authorized to execute governance operations.
Governance operations are executed by calling a dedicated governance method on the contracts.

All contracts will be expected to support online upgrades. This implies changes to the Ethereum and Terra contracts to
make them upgradeable.

### Consistency Levels

The consistency level represents the integrator's request to withhold from signing a message until a specified
commitment level is reached on a given chain, or alternatively to leverage faster-than-finality messaging.
This differentiation is critically important on chains which do not have instant finality, such as Ethereum,
to ensure that the transaction which resulted in a Wormhole message was not 'rolled back' due to a chain
reorganization. Each [guardian watcher](../node/pkg/watchers/README.md) is responsible for defining the consistency
level meanings and enforcing them.

#### EVM

- `200` - publish immediately
- `201` - `safe`, if available, otherwise falls back to `finalized`
- `202` - `finalized`
- `203` - custom handling
- anything else is treated as `finalized`

Historically, the EVM watcher specified the consistency level as the block depth (from `latest`) the transaction
should reach before publishing. However, since [The Merge](https://ethereum.org/en/roadmap/merge/), adoption of
`safe` and `finalized` block tags have become widespread and offer a more exact measure of commitment.

### Custom Handling

The consistency level of 203 indicates that the integrator desires special handling. When the watcher sees this value,
it reads the `CustomConsistencyLevel` on-chain contract, using the emitter address as the key to determine the
special handling to be performed. If the emitter address is not configured in the contract, then the observation
is treated as though it had consistency level 202 (finalized).

#### Additional Blocks handling

Currently, the only supported custom handling is to wait the configured number of additional blocks after the specified
consistency level. For instance, if the integrator has configured their emitter address in the `CustomConsistencyLevel`
contract as consistencyLevel == `201` and additional blocks == 5, then the watcher will not approve the observation until
five safe blocks after when the observation block is marked safe.

##### Usage on chains that publish bursts

Since safe and finalized blocks on Ethereum advance in bursts, rather than one-by-one, it probably does not make sense to use
this feature for safe or finalized. Either "safe plus two" will happen at the same time as the observed block is marked safe,
or it will wait until the next burst, which could be an arbitrarily long time.

However, this feature might be useful on Ethereum to wait X blocks after the block is published as latest. For instance,
if an integrator has configured their emitter address in the `CustomConsistencyLevel` contract as consistencyLevel == `200`
and additional blocks == 2, the observation would get published two blocks after the observed block.

##### Usage on chains that publish incrementally

On chains like Hyper EVM, where safe and finalized are published regularly, this feature might be useful to wait X blocks
after the block is published as safe or finalized, since that would happen in a predictable fashion. For instance,
if an integrator has configured their emitter address in the `CustomConsistencyLevel` contract as consistencyLevel == `201`
and additional blocks == 2, the observation would get published two safe blocks after the observed block is marked safe.

#### Solana

The Solana core contract provides an enum for `ConsistencyLevel` used by the instruction data:

- `0` - Confirmed
- `1` - Finalized

However, the resulting account and subsequent VAA will have:

- `1` - Confirmed
- `32` - Finalized

#### Others

All other chains do not offer configurable consistency levels and this field will be `0`.

## Caveats

While the `<chain>/<emitter>/<sequence>` is commonly used to identify VAAs, the hash of the observation body is used
to uniquely identify a VAA for replay protection.

## Related Technologies

In this section, Wormhole is compared to related technologies on the market. We have carefully evaluated all existing
solutions to ensure that we have selected a unique set of trade-offs and are not reinventing any wheels.

### Cosmos Hub and IBC

The [IBC protocol](https://ibcprotocol.org/documentation), famously implemented by the Cosmos SDK, occupies a similar
problem space as Wormhole - cross-chain message passing. It is orthogonal to Wormhole and solves a larger and
differently shaped problem, leading to a different design.

IBC specifies a cross-chain communication protocol with high-level semantics like channels, ports, acknowledgments,
ordering and timeouts. It is a stream abstraction on top of a packet/datagram transport, vaguely similar to the TCP/IP
protocol. IBC is part of the Cosmos Internet of Blockchain scalability vision, with hundreds or even thousands of
sovereign IBC-compatible chains (called "zones") communicating via IBC using a hub-and-spoke topology. Data availability
is provided by permissionless relayers.

With IBC, for two chains to communicate directly with each other, they would have to be able to prove state mutually.
This usually means implementing light clients for the other chain. In modern pBFT chains like those based
on [Tendermint](https://v1.cosmos.network/resources/whitepaper) consensus, verifying light client proofs
is [very cheap](https://blog.cosmos.network/light-clients-in-tendermint-consensus-1237cfbda104) - all that is needed is
to follow validator set changes, instead of a full header chain. However, chains talking to each other directly would
get unmanageable with many chains - and this is where central hubs like Cosmos Hub come in. Instead of every individual
chain discovering and validating proofs of every other chain, instead, it can choose trust a single chain - the Hub -
which then runs light clients for every chain it is connected to. This requires the hub to have a very high degree of
security, which is why the Cosmos Hub has its own token - $ATOM - which now has a billion-dollar market cap.

IBC works best when connecting modern pBFT chains that implement the IBC protocol and whose light client proofs are
cheap to verify.

This is not the case for chains like Ethereum or Solana. Ethereum requires a lot of state - the full header chain - to
verify inclusion proofs. This is too expensive to do on the Hub, or any individual Cosmos chain, so a proxy chain (
called a "peg zone") instead verifies the proofs, similarly to Wormhole. The peg zone would have its own security and
validator set just like any other zone, and vouches for the Ethereum state.

See [Gravity](https://github.com/cosmos/gravity-bridge) for how an Ethereum peg zone would look like. It's possible to
verify Cosmos light client proofs on Ethereum, but not vice versa - the peg zone validators are trusted just like
Wormhole nodes, and use a multisig mechanism similar to Wormhole for messages sent to Ethereum.

Solana does not currently provide a light client implementation, but like Ethereum, any Solana light client would also
need a [large amount of state](https://docs.solana.com/proposals/simple-payment-and-state-verification) to verify
inclusion proofs due to the complexity of the Solana consensus.

Instead of connecting hundreds of IBC-compatible chains with a few non-IBC outliers with peg zones, Wormhole is designed
to **connect a low number of high-value DeFi chains**, most of which do not support IBC, which results in a different
design.

A peg zone is the closest analogy to Wormhole in the IBC model, with some important differences:

- Wormhole is a lower-level building block than IBC and specifies no high-level semantics like connections or target
  chains, leaving this to higher-layer protocols (think "Ethernet", not "TCP/IP"). This is more flexible and less
  complex to implement and audit, and moves the complexity to the upper layer and libraries only where it is needed.

- Instead of operating our own Layer 1 proof-of-stake chain, we rely on finality of the connected chains. A staking
  mechanism for Wormhole guardian nodes would be managed by a smart contract on one of those chains and inherit its
  security properties. Nodes cannot initiate consensus on their own.

- By only reacting to finalized state on chains, each with strong finality guarantees, the Wormhole protocol does not
  need complex consensus, finality or leader election. It signs _observations_ of finalized state, which all nodes do
  synchronously, and broadcasts them to a peer-to-peer network. There's no possibility of equivocation or eclipse
  attacks leading to disagreements.

- Long-range attacks and other PoS attacks are prevented by guardian set update finality on the connected chains. After
  a brief convergence window, the old guardian set becomes invalid and no alternative histories can be built.

- Instead of relying on inclusion proofs, we use a multisig scheme which is easier to understand and audit and cheaper
  to verify on all connected chains. The extra guarantees offered by an inclusion proof are not needed in the Wormhole
  network, since it merely shuttles data between chains, each of which have provable and immutable history.

## Security Considerations

When integrating with Wormhole, it is important to understand the trust stack inherited based on each field of the VAA.

```go
byte        version                  // VAA Version
u32         guardian_set_index       // Indicates which guardian set is signing
u8          len_signatures           // Number of signatures stored
[]signature signatures               // Collection of guardian signatures
```

These fields are **not** part of the signed observation but are used by the verifying core bridge to determine if the
designated guardian set is active and if the corresponding guardian signatures are valid. You always additionally
inherit the trust assumptions of any verification mechanism you use, such as the core bridge on a given chain and its
runtime.

The following fields all inherit the [trust assumptions](../SECURITY.md#trust-assumptions) of the **Guardians**.

```go
u16         emitter_chain     // The id of the chain that emitted the message
```

The emitter chain is solely determined by the guardian. Only the chain RPC nodes which are connected by a quorum of
guardians for a given chain ID can emit verifiable messages with this chain ID.

Therefore, the following fields inherit the trust assumptions of the **Guardians and the emitter chain**'s RPC node.

```go
u32         timestamp         // The timestamp of the block this message was published in
```

The timestamp is provided directly by the RPC node.

Based on the particular chain implementation, the core bridge emits a message with the following properties, which
are interpreted by the guardian and inherit the trust assumptions of the **Guardians, emitter chain, and core bridge**
implementation.

```go
[32]byte    emitter_address   // The contract address (wormhole formatted) that called the core contract
u64         sequence          // The auto incrementing integer that represents the number of messages published by this emitter
```

The core bridge is responsible for identifying the calling contract address and incrementing its sequence number.

The remaining fields are controlled by the calling contract, and therefore inherit **all of the above trust assumptions
in addition to those of the emitter contract**.

```go
u32         nonce             //
u8          consistency_level // The consistency level (finality) required by this emitter
[]byte      payload           // arbitrary bytes containing the data to be acted on
```

These five fields are interpreted by the guardian which must not proceed with the signing process until the specified
`consistency_level` has been reached, as applicable.
