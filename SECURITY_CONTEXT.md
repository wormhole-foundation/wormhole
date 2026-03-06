# Security Context

> **For security researchers and automated tools:** This document describes
> design decisions, accepted risks, and commonly misreported patterns in the
> Wormhole codebase. Findings that match the patterns below will be closed
> as informational. If your finding is a novel variant not covered by the
> explanation, please submit it.
>
> **Bug bounty program:** https://immunefi.com/bug-bounty/wormhole/information/
>
> **Full security policy:** [SECURITY.md](./SECURITY.md)

## How to Use This File

Before submitting a finding, check whether it matches a pattern below.
Each entry describes a bug class that researchers commonly report, and
explains why it does not apply to this codebase (or why the risk is
accepted). Bounty submissions that fit the below description will
be automatically closed.

## Non-issues

### Any impact that assumes control over a quorum of signing keys as a precondition

**Why this is accepted:**

If an attacker can control a quorum of Guardian Keys (or other signing keys depending on the context), they
would have full control of the protocol. Achieving this would involve simultaneous compromise of heterogeneous
infrastructure operated by independent parties (i.e. the Guardians). This is invalid as an axiom
for a proof-of-concept given that it's infeasible without supporting evidence.

### The Token Bridges (Wrapped Token Transfers) do not support tokens that are malicious or attacker-controlled

**Why this is accepted:**
Registering tokens is permissionless by design. It is trivial for an attacker to create a rug-pull token
and stopping that is not in-scope for the token bridge. Such tokens might be ERC-20 tokens with methods
that always revert or lie to the caller about token invariants. As such, these tokens are worthless
in practice and the ability for an attacker to rug a contract they already controlled is not
relevant to Wormhole's security.

Findings that rely on an attacker-controlled token, and other users interacting with it are ineligible
unless they impact the bridge itself.

### Denial-of-service based on linear usage of a rate limiting mechanism

**Why this is accepted:**
The token bridges have the Governor enabled. NTT implementations may use their own rate limiter.
If an attacker sends a large amount of funds through a rate-limited protocol, it is normal
and expected that their transfers, as well as those of others, are delayed for some amount of time.
This is the purpose of these security mechanisms. Guardians or NTT administrators can choose to
scale up their rate limits in case of heavy flows of assets, or else enable flow-cancelling
as appropriate. An attacker sending a ton of tokens through a protocol is the same
case as many users sending a large number of smaller transfers. It is safe and expected.

The term 'linear' here means that an attacker sends an amount of tokens through in order to
consume an equivalent amount of capacity in a rate limiter. If they can inflate the
consumed capacity out of proportion to the funds they send, this could be a bug
that we want to know about.

### Guardian Sets or indices not being signed or included directly within a VAAs hashed contents

**Why this is accepted:**
Guardian set and signature validation occur on the consuming chains. They are not expected to be
part of the VAA. If an attacker modified the Guardian Set info of an in-flight VAA, the consuming 
chain will fail to verify it during the signature validation step. This is the responsibility
of consumer contracts on the destination chain.
VAAs are bound to a specific guardian set index; cross-set replay is prevented by on-chain validation.

### Multiple Guardian Sets may be active for a short time during a guardian set rotation

**Why this is accepted:**
When rotating Guardian Sets, the old set stays active for a short time in order to prevent bricking
in-flight transfers.

### A malicious Guardian sends a small number of p2p requests with very large size to other Guardians to try to cause DoS

**Why this is accepted:**
libp2p has a 1MB limit per message by default. It's not possible for Guardians to stuff huge amounts of data in a small number of messages. Large numbers of messages are considered volumetric attacks and are out of scope.

### Truncation in Governor calculations due to use of floats and rounding

**Why this is accepted:**
Governor price calculations do not need to be extremely precise. It works as
a rate limiter and some divergence in prices and calculations over time and
between Guardians is acceptable.
Even with truncation, the Governor's prices are still accurate beyond some
fractions of a cent. If an attacker tries to subvert the limits
by sending many transfers of extremely small value to abuse truncation, the gas cost 
will make this extremely unprofitable.

