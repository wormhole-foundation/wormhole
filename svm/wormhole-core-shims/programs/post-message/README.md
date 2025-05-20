# Post Message Shim

## Objective

This program aims to reduce the cost of Wormhole Core Bridge program message
emission on SVM networks. It provides a new emission mechanism without making
changes to the existing Wormhole Core Bridge program.

## Background

The existing [post message instructions] has the following limitations:

- The post message instruction creates a new message account for every call.
  These accounts cannot be reclaimed and take up state forever, even after
  the VAA has been produced and consumed. The post message unreliable
  instruction allows reusing a message account, but according to the comment in
  the Wormhole Core Bridge program implementation, there is “no way” to recover
  the message after it has been overwritten. It also requires that the payload
  size matches the existing message account since the feature was written before
  account resizing.
- The post message instructions take mutable references to the bridge config
  and fee_collector accounts. These writable accounts limit the ability to
  parallelize calls across multiple integrators. The same could be said of the
  emitter sequence account for a single integrator, but at least that is part of
  the specification for the Wormhole Core Bridge.

The emitter address on SVM networks is a signer for the post message
instructions. Typically this address is a PDA of the integrating program.

Though it is not guaranteed to be a unique identifier, many components of the
Wormhole ecosystem use `chain/emitter/sequence` as an id. Furthermore,
the [message digest] should be unique between two different messages. As such,
the digest is recommended for replay protection. Therefore, it is important that
a new emission mechanism not reuse sequence numbers for the same emitter.

## Goals

- Provide a new mechanism for publishing messages which does not require a new
  account for every message in order to be “reliable”.
- Provide a migration recommendation to avoid emitter sequence number reuse.
  Ideally, migration should be as easy as possible for integrators.
- Detail how this could be integrated into Modular Messaging.
- Handle a message fee. If this is not required, this design could be revised to
  further reduce costs and complexity as well as increase parallelization
  potential.

## Non-Goals

- Prevent emissions on the existing Wormhole Core Bridge program. This design
  must treat the existing core bridge as non-upgradeable.
- Message verification. Integrators can use separate contracts or methods for
  emission and verification.

## Overview

1. A new SVM program which takes the same arguments as the post message
  unreliable instruction while providing the additional information necessary
  for the guardian to make them reliable. It will act as a shim, augmenting the
  existing Wormhole Core Bridge program.
2. A modification to the guardian watcher which detects calls that are made via
  this new program and observes the message solely via the instruction data
  (using Anchor event CPI).
3. New Guardian recommendations for Solana RPC instruction data availability.

## Detailed Design

### Solana Program Technical Details

For ease of integration, this program will follow the [Anchor] program
interface, which exposes one CPI instruction (post message), which:

- MUST take at least the same accounts and instruction data as the Wormhole Core
  Bridge program's post message unreliable instruction.
- MUST read the sequence number from the sequence account and emit it in an
  [Anchor CPI event], along with `Clock::unix_timestamp as u32` for convenience.
- MUST call the Wormhole Core Bridge program's post message unreliable
  instruction, replacing message's payload with an empty vector. This way,
  payloads of different sizes can be supported and the most limited data can be
  written to the account. This also keeps tracking the sequence number against
  the emitter and ensures the appropriate fee is paid.

With this mechanism, the guardian will have all of the information it needs to
re-build the message from only the instruction data.

1. Shim program’s post message instruction:
  - `nonce`
  - `payload`
  - `consistency_level`
2. Shim program’s CPI event
  - `emitter`
  - `timestamp`
  - `sequence`

### Guardian Changes

The guardian code requires two modifications to make this successful:

1. Look for the Wormhole Post Message Shim program's post message instruction
   and read the corresponding message data from the instruction instead of the
   message account. Ignore the corresponding unreliable message account from the
   Wormhole Core Bridge (or see below). The guardian also MUST NOT re-observe
   unreliable messages from these accounts as this could lead to a duplicate VAA
   for the same sequence number, but with an empty payload (since that is what
   is written to the message account).
2. Support re-observations by transaction ID. See
   https://github.com/wormhole-foundation/wormhole/pull/4101.

Optionally, ignore unreliable messages with a zero length payload (which MAY be
paired with a particular hard-coded nonce passed by the shim). Ignoring
zero-length unreliable messages avoids any mechanism of accidentally picking up
these messages via the account and avoids some added code complexity in skipping
the corresponding instruction.

### Guardian RPC Changes

The Guardian’s SVM RPCs will likely need to support fetching transaction
instruction information farther back in time than they currently support (about
30 minutes or so from initial testing). For example,
https://docs.anza.xyz/implemented-proposals/rpc-transaction-history/

For reference, guardians must already set [the following flags].

The Wormhole Foundation should work with Guardians and the community to
establish an acceptable and recommended SLA.

> 🚧 TODO: investigate the flags / process necessary to store the required data
for a longer period of time. Perhaps it is
`--enable-extended-tx-metadata-storage` or `--limit-ledger-size`?

</aside>

### Protocol Integration

The monitor will need to change to support reobservation via transaction for
these messages.

Given that the existing SVM RPC configuration for most of the guardians is
expected to be limited to approximately 30 minutes of transaction history, it
would be prudent to have a separate re-observation pipeline for SVM emissions
utilizing the shim.

Wormholescan may want to indicate the difference in emission somehow.

Modular Messaging - an adapter can use this program for message emission and
its own mechanism (via loading the Wormhole Core Bridge program accounts) for
verification.

### API and Database Schema

N/A

## Caveats

This approach would not increase the potential parallelization of Wormhole Core
Bridge program message emission because it still relies on the shared writable
fee collector account.

For Solana specifically, it is critically important to note that this network
has a [maximum call depth of 4] and the very first call with a given emitter
will result in 1 additional CPI call stack depth than the equivalent
Wormhole Core Bridge program's post message instruction call. However,
subsequent calls will be equivalent. **For this reason, if an integrating program
is _only_ called via CPI from another program, it is recommended that they emit
an empty message upon initialization / migration so that they will not encounter
an edge case here.**

```jsx
// Direct core integration, `post_message`
Integrator Program -> Core Bridge -> System Program (init account)
// Initial shim call, per emitter (+1)
Integrator Program -> Shim -> Core Bridge -> System Program (init account)
                           -> Shim
// Subsequent shim calls, per emitter (+0)
Integrator Program -> Shim -> Core Bridge
                           -> Shim
```

# Alternatives Considered

- Make account reuse for the Wormhole Core Bridge program's post message
unreliable instruction reliable via guardian changes. Unfortunately, the only
artifact of the sequence number at the time of the emission is the log message,
which can be truncated. And the content of the message account can be
overridden. There is currently a [guardian optimization] which relies on the log
messages, but they should not be relied upon for critical information.
- Avoid calling the existing Wormhole Core Bridge program and track sequence
numbers independently. This path would be more reasonable if the message fee was
not required, but would require special handling for migrations to avoid
potentially duplicate message digests. Some possible solutions include:
  - A new, unique emitter could be derived by this contract, which would avoid
    conflicts but would make it more difficult to migrate. Existing protocols
    would need to update their emitter and potential support two emitters
    simultaneously during the migration.
  - This contract could reuse the emitter but use new enum values for the
    consistency levels. Like other alternatives (such as setting the first bit
    of the sequence number to 1), this solution is not very intuitive, scalable
    or repeatable.
  - Document this issue and warn integrators against re-using an emitter.

## Security Considerations

As mentioned at various points in this document, it is critically important
that a given message MUST NEVER result in two different VAAs. Incidents which
may result in duplicate message emission for unique messages (such as reusing
sequence numbers) should also be avoided, as it could lead to an unredeemable
VAA for an integrator (even if they were using the recommended message digest
approach).

After initial testing is complete and proper configuration and functionality of
the Wormhole Post Message Shim program is confirmed, the upgrade authority
should be discarded in order to improve trust assumptions and avoid possible
compromise of the upgrade authority key.

## Test Plan

The Wormhole Post Message Shim program may be developed in a standalone repo and
can leverage Tilt akin to NTT. That way it can test emission e2e with a running
guardian. Alternatively, if desired, it could exist in a new directory in the
monorepo (the advantage of which would be to ensure that co-developed guardian
code continues to work).

Gas cost analysis should be performed for initial and subsequent message
emissions with and without the shim and of various lengths.

## Performance Impact

This should greatly reduce the cost of emitting messages on Solana for
integrators who are willing to accept the risk of not having a permanent account
for each message, as it will save the current rent exemption cost per message.

It does, however, increase the compute units required by introducing another
program and two instructions (one via event CPI) into the mix. Some of this may
be slightly reduced by meticulously optimizing the Wormhole Post Message Shim
program instruction. Initial testing results below compare the existing
Wormhole Core Bridge program's post message instruction call with one from this
Shim program, with and without extra Anchor address enforcement that is not
necessary but helps IDL generation.

NOTE: The compute unit calculations below are dependent on the number of bump
iterations using a specific emitter key. Because the Wormhole Post Message Shim
program's message PDA and Wormhole Core Bridge program's emitter sequence
accounts are derived using the emitter key, CU usage can vary. This CU report
can may also change in the future with any optimizations introduced to the
program.

```jsx
core: lamports 1638720 SOL 0.00163872, $0.3564871488, CU 26827
shim: lamports 5120    SOL 0.00000512, $0.0011138048, CU 23057
```

## Rollout

This will require a guardian release and deployment of the Wormhole Post Message
Shim program. If desired, upgrade authority of the shim can be maintained for a
limited time while testing.

### Acceptance Criteria

Test that emitting a message via Wormhole Core Bridge program's post message
instructions along with the Wormhole Post Message Shim program's post message
instruction all result in the appropriate VAAs.

### Rollback

Disable the guardian detection of shim instructions. Enabling could be gated
behind a feature flag if desired.

[Anchor]: https://www.anchor-lang.com/
[Anchor CPI event]: https://www.anchor-lang.com/docs/features/events#emit_cpi
[guardian optimization]: https://github.com/wormhole-foundation/wormhole/blob/1dbe8459b96e182932d0dd5ae4b6bbce6f48cb09/node/pkg/watchers/solana/client.go#L563
[maximum call depth of 4]: https://solana.com/docs/core/cpi#key-points
[message digest]: https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0001_generic_message_passing.md#detailed-design
[post message instructions]: https://github.com/wormhole-foundation/wormhole/blob/main/solana/bridge/program/src/api/post_message.rs
[the following flags]: https://github.com/wormhole-foundation/wormhole/blob/main/docs/operations.md#solana-node-requirements