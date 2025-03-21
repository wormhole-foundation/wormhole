# Verify VAA Shim

## Objective

Reduce the cost of Wormhole Core Bridge program message verification on SVM
networks. Provide a new verification mechanism without making changes to the
existing Wormhole Core Bridge program.

## Background

The current way most integrators on SVM networks verify VAAs is via the
following process:

1. Call the Secp256k1 Native program and the Wormhole Core Bridge program's
   [verify signatures instruction] in order to verify and store all of the
   signatures from the VAA into a signature set account. These pair of
   instructions are called multiple times because the signature data for 13
   signatures is too long to fit into a single transaction.
   _There used to be a compute unit restriction as well, but that is no longer
   an issue as a higher limit can now be requested._
2. Call the Wormhole Core Bridge program's [post VAA instruction] to validate
   the signature set account against the VAA and post the VAA data into an
   account.
3. Call an instruction on your program which takes a posted VAA account and
   assumes its validity.

Notably, neither the signature set nor is the posted VAA accounts are closed.
And there exists no mechanism to close them either. This means that posters
permanently lose the lamports dedicated to the rent exemption for these
accounts.

However, some integrations, such as [Solana World ID] use a different approach:

1. Call a [post signatures instruction] on your program (potentially multiple
   times) to store the signature data on-chain in a guardian signatures account.
2. Call an instruction on your program which takes a Wormhole Core Bridge
   program's [guardian set] account and your program's [guardian signatures]
   account, [performing the required checks and verification]. If the
   verification is successful, it performs whatever action is needed and closes
   the signatures account, refunding the lamports.

<!-- cspell:disable -->

This approach does not require leaving any on-chain artifacts and is more cost
effective. Part of the reason this is now possible in less instructions is due
to the `secp256k1_recover` SVM syscall and ability to increase the compute unit
budget.

<!-- cspell:enable -->

> 💡 This will also provide integrators the flexibility to post the VAA payload
separately as well if needed for payloads that won’t fit into a transaction.

## Goals

- Recommend a documented mechanism for verifying VAAs which does not require
  permanently storing accounts.
- Provide a reusable approach to perform the bulk of the work.
- Provide a cost savings estimation compared to the Wormhole Core Bridge
  program's verify signatures and post VAA approach.

## Non-Goals

- Provide an on-chain program which generates an artifact of verification akin
  to the Wormhole Core Bridge program's post VAA instruction.

## Overview

Save end-user costs on SVM network VAA verification by providing a shim, which
integrators can CPI call to perform verification. This shim will allow the
client to clean up any artifacts generated during the verification process,
ultimately resulting in a lower cost verification.

## Detailed Design

The shim will have three instructions which can be leveraged over two
transactions, resulting in less transactions and rent costs than the Wormhole
Core Bridge program's approach.

This method explicitly only operates on the signatures, Wormhole Core Bridge
program's guardian set and digest, making it compatible with both v1 VAAs and
Queries while maintaining a static instruction size for verification.

As a bonus, this also allows for verification of VAA bodies that are larger what
can fit in a single instruction, which was not possible with the Wormhole Core
Bridge program. Integrators can pass the VAA body via instruction data or
account, as suits their needs. How they choose to do so is outside the scope of
this design.

### Post Signatures Technical Details

This instruction creates or appends to a guardian signatures account account for
subsequent use by the [verify hash instruction](#verify-hash-technical-details).
This step is necessary because the Wormhole VAA body, which has an arbitrary
size, and 13 guardian signatures (a quorum of the current 19 mainnet guardians,
66 bytes each) alongside the required accounts is likely larger than the
transaction size limit on Solana (1232 bytes). This will also allow for the
verification of other messages which guardians sign, such as query results.

This instruction allows for the initial payer to append additional signatures to
the account by calling the instruction again. Subsequent calls may be necessary
if a quorum of signatures from the current guardian set grows larger than can
fit into a single transaction.

The guardian signatures account can be closed by the initial payer via
the [close signatures instruction](#close-signatures-technical-details), which
will refund the initial payer.

### Verify Hash Technical Details

<!-- cspell:disable -->

This instruction is intended to be invoked via CPI call. It verifies a digest
against a GuardianSignatures account and a Wormhole Core Bridge program's
guardian set. Prior to this call, and likely in a separate transaction,
the post signatures instruction must be called to create the account.
Immediately after calling the verify hash instruction, the close signatures
instruction should be called to reclaim the lamports.


A v1 VAA digest can be computed as follows:

```rust
let message_hash = &solana_program::keccak::hashv(&[&vaa_body]).to_bytes();
let digest = keccak::hash(message_hash.as_slice()).to_bytes();
```

A QueryResponse digest can be computed as follows:

```rust
use wormhole_query_sdk::MESSAGE_PREFIX;
let message_hash = [
  MESSAGE_PREFIX,
  &solana_program::keccak::hashv(&[&bytes]).to_bytes(),
]
.concat();
let digest = keccak::hash(message_hash.as_slice()).to_bytes();
```

<!-- cspell:enable -->

### Close Signatures Technical Details

Allows the initial payer to close the guardian signature account, reclaiming the
rent taken by the post signatures instruction to create this account.

### Protocol Integration

There are no changes required to the protocol.

### API / Database Schema

N/A

## Caveats

This shim will need to include the following patch made to the Wormhole Core
Bridge program when calculating the expiry for guardian sets.

```rs
pub fn is_active(&self, timestamp: u32) -> bool {
    // Note: This is a fix for Wormhole on mainnet. The initial guardian set was
    // never expired so we block it here.
    if self.index == 0 && self.creation_time == 1_628_099_186 {
        false
    } else {
        self.expiration_time == 0 || self.expiration_time >= timestamp
    }
}
```

Unlike the Wormhole Core Bridge program, this shim program will not need to
perform signature set deny-listing, as it does the verification directly through
`secp256k1_recover`.

Since it is planned to be non-upgradeable, any similar mitigation strategies
will not be possible. For example, the only way to expire a guardian set will be
for the Wormhole Core Bridge program to properly expire it.

## Alternatives Considered

- Integrators perform verification logic themselves. This design does not
  prohibit integrators from doing this. Having each integrator implement
  verification in their own program is a significant amount of code duplication
  to recommend. The purpose here is to provide an easy mechanism for integrators
  to verify VAAs.
- Post VAA data to an account. For some integrators, the VAA body may fit into
  their instruction data, so avoiding the account would save cost and added
  complexity. For integrators with larger payloads, they can first post the body
  into an account and process it in a subsequent instruction. They may have
  other efficiencies to gain in how they do that, such as automatically closing
  the account after verifying the VAA and performing the desired action. It
  seems best to leave this up to the integrator.
- Provide an artifact of verification, akin to the Wormhole Core Bridge
  program's post VAA instruction. While this has the advantage of using only 1
  account instead of 3, it has the disadvantage of being another temporary
  account to clean up _and_ does not enforce that the VAA is _currently_ valid.
  Rather that it was at some time in the past. Since the advent of
  [Address Lookup Tables], using more accounts is less of an issue than it once
  was, and ensuring that the VAA is actually valid when you consume it is a
  security gain.

## Security Considerations

See [Caveats](#caveats).

After initial testing is complete and proper configuration and functionality of
the shim is confirmed, the upgrade authority should be discarded in order to
improve trust assumptions and avoid possible compromise of the key.

## Test Plan

Gas cost analysis should be performed for verifying VAAs with and without the shim.

## Performance Impact

This should greatly reduce the cost of verifying messages on Solana, as it will
save the current rent exemption cost per signature set and posted VAA account.

It does, however, increase the compute units required by using
`secp256k1_recover` instead of the native instruction.

```jsx
core (1/4): lamports  1346520, SOL  0.00134652,  $ 0.2929219608,  CU 37058
core (2/4): lamports    40000, SOL  0.00004,     $ 0.0087016,     CU 25016
core (3/4): lamports  2482760, SOL  0.00248276,  $ 0.5400996104,  CU 82333
core (4/4): lamports     4992, SOL  0.000004992, $ 0.00108595968, CU  2302
total:                3874272,      0.003874272, $ 0.84280913088,   146709

shim (1/2): lamports  7206592, SOL  0.007206592, $1.56772202368,  CU   3037
shim (2/2): lamports -7191616  SOL -0.007191616, $-1.56446414464, CU 334846
total:                  15040,      0.000015040, $ 0.00325787904,    337883

savings:              3859232,      0.003859232, $ 0.83955125184,   -191174
```

That is ~190k CU for ~.3% of the cost.

## Rollout

This only requires deployment of the shim program. If desired, upgrade authority
of the shim can be maintained for a limited time while testing.

### Acceptance Criteria

Test that verifying a valid VAA digest and signatures via the shim’s verify hash
instruction succeeds. Test that an expired guardian set, incorrect digest, bad
signatures, etc, fail.

## Rollback

N/A

[Address Lookup Tables]: https://solana.com/docs/advanced/lookup-tables
[guardian set]: https://github.com/wormholelabs-xyz/solana-world-id-program/blob/68f1740b2b9bad9d86bd933001a3716a2a993930/programs/solana-world-id-program/src/instructions/update_root_with_query.rs#L70
[guardian signatures]: https://github.com/wormholelabs-xyz/solana-world-id-program/blob/68f1740b2b9bad9d86bd933001a3716a2a993930/programs/solana-world-id-program/src/instructions/update_root_with_query.rs#L74
[performing the required checks and verification]: https://github.com/wormholelabs-xyz/solana-world-id-program/blob/68f1740b2b9bad9d86bd933001a3716a2a993930/programs/solana-world-id-program/src/instructions/update_root_with_query.rs#L113
[post signatures instruction]: https://github.com/wormholelabs-xyz/solana-world-id-program/blob/68f1740b2b9bad9d86bd933001a3716a2a993930/programs/solana-world-id-program/src/instructions/post_signatures.rs#L46
[post VAA instruction]: https://github.com/wormhole-foundation/wormhole/blob/db1ee86bfbaf16383c46150a44b1437a522514da/solana/bridge/program/src/api/post_vaa.rs#L102
[Solana World ID]: https://github.com/wormholelabs-xyz/solana-world-id-program/tree/main
[verify signatures instruction]: https://github.com/wormhole-foundation/wormhole/blob/db1ee86bfbaf16383c46150a44b1437a522514da/solana/bridge/program/src/api/verify_signature.rs#L67
