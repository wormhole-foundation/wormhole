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
accepted). 

## Non-issues

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

