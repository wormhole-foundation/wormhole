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

**Justification:**

If an attacker can control a quorum of Guardian Keys (or other signing keys depending on the context), they
would have full control of the protocol. Achieving this would involve simultaneous compromise of heterogeneous
infrastructure operated by independent parties (i.e. the Guardians). This is invalid as an axiom
for a proof-of-concept given that it's infeasible without supporting evidence.

### The Token Bridges (Wrapped Token Transfers) do not support tokens that are malicious or attacker-controlled

**Justification:**
Registering tokens is permissionless by design. It is trivial for an attacker to create a rug-pull token
and stopping that is not in-scope for the token bridge. Such tokens might be ERC-20 tokens with methods
that always revert or lie to the caller about token invariants. As such, these tokens are worthless
in practice and the ability for an attacker to rug a contract they already controlled is not
relevant to Wormhole's security.

Findings that rely on an attacker-controlled token, and other users interacting with it are ineligible
unless they impact the bridge itself.

### Denial-of-service based on linear usage of a rate limiting mechanism

**Justification:**
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

**Justification:**
Guardian set and signature validation occur on the consuming chains. They are not expected to be
part of the VAA. If an attacker modified the Guardian Set info of an in-flight VAA to an invalid
index, the consuming chain will fail to verify it during the signature validation step. However,
it is acceptable to "repair" a VAA from an old Guardian set by updating the Guardian set index,
provided a quorum of the new set are signers. This is the responsibility of consumer contracts
on the destination chain.
VAAs are bound to a specific guardian set index; cross-set replay is prevented by on-chain validation.

### Multiple Guardian Sets may be active for a short time during a guardian set rotation / Old delegate guardian sets can influence observations for a short time period

**Justification:**
When rotating Guardian Sets, the old set stays active for a short time in order to prevent bricking
in-flight transfers.

Delegated Guardian Set updates follow the same transition model: observations already grouped under the previous delegated 
configuration may complete against that configuration after a rotation or removal, preventing in-flight messages from being stranded.

### A malicious Guardian sends a small number of p2p requests with very large size to other Guardians to try to cause DoS

**Justification:**
libp2p has a 1MB limit per message by default. It's not possible for Guardians to stuff huge amounts of data in a small 
number of messages. Large numbers of messages are considered volumetric attacks and are out of scope.

### Truncation in Governor calculations due to use of floats and rounding

**Justification:**
Governor price calculations do not need to be extremely precise. It works as
a rate limiter and some divergence in prices and calculations over time and
between Guardians is acceptable.
Even with truncation, the Governor's prices are still accurate beyond some
fractions of a cent. If an attacker tries to subvert the limits
by sending many transfers of extremely small value to abuse truncation, the gas cost 
will make this extremely unprofitable.

### Replay protection or database poisoning claims against the Governor and Notary

**Justification:**

The Governor and Notary do not implement replay protection beyond the natural lifecycle of
when observations, transfers, or related state are stored in their databases. Findings that
assume additional replay protection should exist outside of that storage lifecycle are out
of scope.

This also applies to claims that replayed or otherwise malicious messages can permanently
poison those databases. Delayed and pending messages are periodically processed and removed
when they become ready, and Governor transfer records are pruned from the database as they
fall outside the active rate-limit window, including during reload after restart.

We are also aware that a message may be delayed by the Notary and then delayed again by
the Governor after the Notary releases it. This possible "double delay" is part of the
current lifecycle when both mechanisms are enabled, not a separate replay-protection issue.

Guardians also have operational tools, including admin commands, to inspect and update
relevant local database state. If necessary, operators could also perform manual operations
on the underlying BadgerDB storage directly. Because these entries can be modified or
removed, database poisoning claims must demonstrate an impact beyond temporary local
operator intervention.

### Fee-related vulnerabilities in contracts where fees are disabled in production

**Justification:**
Some contracts (e.g., the Token Bridges) have fee mechanisms that are disabled in production,
typically with fees set to zero. Findings that assume fees are enabled or could be exploited
through fee manipulation do not apply when fees are not active. The fee logic exists for
theoretical future use cases or as a defensive measure, but is not currently enforced.
Any attempt to exploit fee logic in such contracts requires a contract upgrade to enable fees,
which is a privileged operation that would undergo the same governance and review process
as any other protocol change.

### Contract front-running risks in Solana deployments

**Justification:**
Reports suggesting that Solana program deployments could be front-run (e.g., by occupying
an expected program address) do not account for how deployments are executed in practice.
Deployment instructions are bundled atomically within a single transaction, meaning the
entire deployment succeeds or fails as a unit. An attacker cannot intervene between
deployment steps to hijack the process. This atomic execution model prevents front-running
attacks that might be possible in environments where deployment occurs across multiple
transactions or time-separated steps.

### Abuse of wormchain-related Cosmos SDK governance

**Justification:**
The wormchain network uses a custom Proof of Authority system based on a modified version
of Cosmos SDK. As a consequence: there is no circulating supply of tokens, votes cannot be proposed,
tokens cannot be staked, etc.

Any reports that suggest an issue in the Guardian network based on manipulating wormchain-related
governance must take the above into account. In practice, Cosmos's `x/gov`-related functionality
should be totally inoperative.

This can be validated by examining the genesis config for wormchain, as well as by inspecting
the mainnet state which should show that none of the validators within wormchain have any
stake.

### Out-of-order submission of governance VAAs / no invariant enforcement for monotonic increase of governance VAAs

Contracts that consume VAAs may process "new" governance VAAs _before_ processing an "old" VAA.
A common suggestion is that the contracts should strictly enforce VAA ordering such that this is
not possible.

**Justification**
We do not want to implement this mechanism because it creates a denial-of-service risk. If we do
enforce ordering, and a message fails for some reason, then a contract could get totally bricked
if it refuses to process VAAs in the future. This could happen due to a bug, or it could be
something an attacker someday figures out how to leverage.

Governance actions are infrequent and relevant contracts are expected to process them in a timely
way, so we don't need to worry about a scenario where there are so many ongoing governance actions
that something like ordering becomes very important.

### NTT attestations are evaluated against live threshold/transceiver config rather than a snapshot taken when the attestation was recorded

NTT's `ManagerBase` records each transceiver's attestation as a
persistent bit but re-evaluates approval against the *current* threshold and *current*
enabled-transceiver set at execution time, rather than against the `(threshold, enabled-set)`
that existed when the attestation was recorded. The suggested impact is that a historical
attestation which was insufficient under an original threshold of 2 or more becomes executable
once governance later reduces the threshold to 1, allowing a pre-staged fraudulent
`NativeTokenTransfer` to be executed by any permissionless caller. A similar argument is made
for transfers that are rate limited and queued for later release, during which time the threshold changes.

**Justification:**
Once a message has been approved, it is expected that it can proceed. Reducing the number of
transceivers is a non-trivial administrative action that requires the integrator to have full
confidence in the remaining transceiver(s); lowering the threshold deliberately makes the
remaining attestations sufficient, which is the intended behavior.

For the described scenario to result in theft, an attacker must first cause a fraudulent
attestation to be recorded against a transceiver. Many deployments use the Wormhole
Transceiver, where producing such an attestation requires an arbitrary VAA to be emitted.
Emitting an arbitrary VAA requires a quorum of Guardians to be malicious or compromised, which
is out of scope per "Any impact that assumes control over a quorum of signing keys as a
precondition" above. Absent a compromised attestation source, no insufficient attestation
exists to be promoted by a later threshold reduction.
