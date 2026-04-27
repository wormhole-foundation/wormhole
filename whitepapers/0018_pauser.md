# WTT and NTT Pauser

## Objective

Develop a mechanism for a set of keys designated by the guardians that can be granted the authority to pause and unpause Wormhole protocols, such as WTT (Token Bridge) and NTT.

## Background

Wormhole protocols operate with a number of defense-in-depth measures, but during an active exploit each protection or action has a different scope and time to implement, including:

- Disconnecting RPCs - a super-minority of guardians can disconnect the chain from the guardian node to stop new messages from generating VAAs.
- [Governor](https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0007_governor.md) - WTT transfers may be made subject to a delay. This limit can be lowered further in response to an exploit but requires a super-minority of guardians to take effect.
- [Global Accountant](https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0011_accountant.md) - WTT transfers and some NTT deployments’ transfers are first checked against the Global Accountant which ensures that the number of wrapped tokens capable of being moved out of a chain never exceeds the number sent to that chain.
- Contract Upgrade - WTT contracts and some NTT deployments are administered by guardian governance which can perform upgrades to mitigate attacks or disable functionality. This requires code changes, review, and governance by a super-majority of guardians.
- Pauser Role - some NTT deployments have the generic guardian governance contract (e.g. [EVM implementation](https://github.com/wormhole-foundation/native-token-transfers/blob/main/evm/src/wormhole/Governance.sol)) set at the pauser. This requires a Wormhole governance message signed by a super-majority of guardians. Note that only the owner can unpause (at least in the [EVM implementation](https://github.com/wormhole-foundation/native-token-transfers/blob/a13746a430fbd4a033aef0ff8180d0d49fdbbab3/evm/src/NttManager/ManagerBase.sol#L342))

These mechanisms have two primary shortcomings when urgent action is required to mitigate an active exploit

1. They require multiple rounds of communication and coordination amongst multiple parties, which may be too slow to effectively respond.
2. Aside from a contract upgrade (WTT) or pause (NTT), which require Wormhole governance, they do not prevent the deposit of funds on the targeted chain nor the withdrawal of funds if the implementation was subject to a verification bypass.

## Goals

- Delegate authority from the Wormhole Guardians for pausing and unpausing
- Unpause MAY use a higher-trust path than pause

## Non-Goals

- Core contract pausing

## Overview

- Introduce immutable contracts which accept a new Wormhole governance payload in order to set a delegated pauser set
- Update the guardian to support the new governance

The WTT (Token Bridge) integration - a `pauser` / `unpauser` role and `pause` / `unpause` entry points - is specified in [0003_token_bridge.md](0003_token_bridge.md).

## Detailed Design

### Technical Details

#### WormholePauser

```bash
                  ┌──────────┐
         propose  │          │  threshold reached
         ────────▶│ Pending  │──────────────────▶ Executed
                  │          │
                  └────┬─────┘
                       │ block.timestamp >= expiresAt
                       └─────────────────────────────▶ Expired
```

This contract MUST be immutable and perform the following functions:

1. Verify a Pauser governance message and update the signers, threshold, expiryDuration, and config index
   1. MUST Validate that the action is the supported `SetConfig` action
   2. MUST Validate that the chain ID matches this chain
   3. MUST Check and mark replay protection for the governance message
   4. Threshold is a `u8`
      1. MUST validate `threshold > 0`
      2. MUST validate `threshold <= numSigners`
   5. Expiry duration is a `u64`
      1. MUST validate `expiryDuration > 0`
   6. Config index is a `u16`
      1. The initial on-chain index is `0`
      2. MUST validate the message index is exactly the current on-chain index + 1 (the first valid `SetConfig` carries index `1`, the next `2`, and so on). This both prevents out-of-order application and provides natural replay protection for config updates.
      3. MUST update the on-chain index to the message index
   7. Signers are defined by the built-in sender verification mechanism on the platform, e.g. `msg.sender` for EVM, `acct.is_signer` on SVM, etc.
      1. MUST validate `numSigners > 0`
      2. MUST validate that each signer's length matches this platform's native address size (e.g. 20 on EVM, 32 on Solana)
      3. MUST validate that no signer is the zero address / all-zero pubkey
      4. MUST validate that the signer list contains no duplicates
   8. MUST emit an event indicating the set has changed
2. Propose a new pause
   1. If the runtime supports it, this SHOULD be an arbitrary call / instruction so that the same contract can be used by multiple deployments
      1. The proposed call MUST carry only the target and the call payload (e.g. `(address target, bytes calldata)` on EVM); it MUST NOT include a value / lamport transfer, consistent with the NTT general purpose governance implementation
   2. MUST verify that the caller is part of the signing set
   3. MUST set the proposal expiry based on the configured expiry duration
   4. MUST mark the proposal with the current config index
   5. MUST assign a unique proposal ID from a monotonically increasing counter (the next-proposal-id counter is incremented on each successful propose). The ID is returned to the proposer and emitted with the proposed event so signers can reference it on approve.
   6. MUST store the proposal on-chain keyed by its ID
   7. MUST emit an event indicating that a proposal has been proposed
   8. MUST auto-approve by reusing the same code path as `Approve`, including the threshold-met execution step (so that a proposal with `threshold == 1` executes immediately)
3. Approve a proposal
   1. MUST verify that the caller is part of the signing set
   2. MUST require that the proposal is active - it exists, has not been executed, has not expired, and its stored config index matches the current on-chain config index. A `SetConfig` therefore implicitly invalidates all proposals from prior configs.
   3. MUST verify that the caller has not already approved
   4. MUST mark the caller as having approved and increment the approval count
   5. MUST emit an event indicating that the caller has approved this proposal
   6. If the threshold has been met:
      1. MUST mark the proposal as having been executed (effect before interaction)
      2. MUST perform the external call / instruction; if the call reverts, the entire transaction MUST revert (so no state - including the approval increment - persists on a failed execution)
      3. MUST emit an event indicating the proposal has been executed
4. Cancel an approval
   1. MUST verify that the caller is part of the signing set
   2. MUST require that the proposal is active
   3. MUST verify that the caller has already approved
   4. MUST unmark the caller as having approved and decrement the approval count
   5. MUST emit an event indicating that the caller has cancelled their approval of this proposal

### Protocol Integration

Add guardian support for the new governance message.

### API / database schema

#### Delegated Pauser Governance

```
// Wire header:
//   module:   "DelegatedPauser" left-padded to 32 bytes
//   action:   uint8 - 1 = SetConfig
//   chainId:  uint16 target chain

// Action 1: SetConfig
// Wire layout (after header):
//   index:          uint16  (must equal on-chain index + 1; first valid is 1)
//   threshold:      uint8
//   expiryDuration: uint64
//   numSigners:     uint8
//   signers:        Signer[numSigners]
//
// Each Signer is length-prefixed:
//   len:   uint8
//   bytes: [len]uint8
//
// Each signer's length must equal the target chain's native address size
// (e.g. 20 on EVM, 32 on Solana). The receiving contract rejects any
// signer whose length does not match its native format.
```

The WTT (Token Bridge) `SetPauserAddresses` governance payload is specified in [0003_token_bridge.md](0003_token_bridge.md).

## Caveats

### Runtime Support

This has proven support with the NTT Wormhole Governance for EVM and SVM, but some chains may not support dynamic dispatch, such as Sui. Those chains may require dedicated contracts per pausing contract.

### Increased Cost

This change may increase cost on some platforms, such as EVM, by introducing a new state read and check on every call.

### Backwards Compatibility

This change should be fully backwards compatible on EVM and SVM for both WTT and NTT. NTT already incorporates a `pauser` role.

### Unpauser Contract

This design does not introduce an Unpauser delegation at this time, as the Wormhole Governance can serve that higher threshold.

Should this be insufficient, another contract and governance message can be implemented specifically for the unpauser functionality. It would likely be equivalent with the exception of the `module` field. No changes to WTT or NTT would need to be made, only a governance to switch the Unpauser role from the generic Wormhole Governance contract to the new Unpauser contract.

## Alternatives Considered

### Proposal Cleanup

Proposal cleanup could be implemented, if so desired. This would remove the permanent on-chain record of the final vote but also save cost and chain state so may be worthwhile. The events and transaction history would likely still be available off-chain.

### Content-Derived Proposal ID

Proposal IDs could be derived deterministically from the proposed call rather than assigned from a counter - for example `keccak256(abi.encode(configIndex, expiresAt, target, value, calldata))` on EVM. The appeal is that any signer could compute the ID from the call data alone, without being told which counter slot it landed in.

In practice this introduces collision and freshness edge cases (two signers proposing the same call in the same block hash to the same ID; re-proposing the same call after an executed/expired entry needs an in-hash nonce like `expiresAt` purely for uniqueness) and costs a keccak over a dynamic payload on every propose. Because the delegated pauser set is a pre-coordinated group with off-chain communication, the proposer can simply broadcast the assigned counter ID, and the content-derived hash buys little. The counter design avoids these edge cases entirely and keeps `configIndex` / `expiresAt` checks where they already live - on the stored proposal.

The counter approach is well-established in on-chain multisig and governance designs:

- Compound GovernorBravo uses a `proposalCount` counter, incremented in `propose()` ([GovernorBravoDelegate.sol#L93-L94](https://github.com/compound-finance/compound-protocol/blob/a3214f67b73310d547e00fc578e8355911c9d376/contracts/Governance/GovernorBravoDelegate.sol#L93-L94)).
- Aave Governance V2 uses a `_proposalsCount` counter ([AaveGovernanceV2.sol#L127](https://github.com/aave/governance-v2/blob/d4e5ae0c31824b9ec9aec4e06a3ee2c3c34ae431/contracts/governance/AaveGovernanceV2.sol#L127)).
- The legacy Gnosis MultiSigWallet uses a `transactionCount` counter for submitted transactions ([MultiSigWallet.sol#L294-L301](https://github.com/gnosis/MultiSigWallet/blob/90639984c960d281bed3e0a5d56dd4adcb9407c4/contracts/MultiSigWallet.sol#L294-L301)).

### Separate Execute Step

The threshold-reaching approval could be decoupled from the external call (e.g. `approve` only marks the threshold as reached, and a subsequent `execute` performs the call). This would let the proposal survive a failed execution attempt and avoid forcing the last approver to pay the gas of the external call. However, it introduces another required transaction in the response path. Given the goal of minimizing time-to-action during an active exploit, the design folds execution into the threshold-reaching approval; if that call reverts, the entire transaction reverts - including the approval increment - so the signer whose tx reverted (or any signer who has not yet approved) may attempt again. Signers whose approvals are already persisted on-chain cannot retry without first cancelling their approval.

### Symmetric Pauser

NTT already set the precedent for unpausing to be a more privileged action than pausing. This design maintains that approach. Besides, if symmetric pauser/unpauser is desired, they can simply be set to the same address.

### Off-Chain Signatures

- Pros - it isn’t publicly revealed that the pause is coming, doesn’t require each signer to have funds on every supported chain
- Cons - requires off-chain coordination / aggregation of signatures

The existing governance mechanism is already public and reducing time-to-action and friction seemed more important. Getting enough funds for a proposal / approval is likely a one-time cost.

### Separate Contracts per Pauser Set

This would require updating all the pauser roles on an update instead of a single governance per chain.

### Core Bridge Pause

This proposal explicitly does not introduce pausing of the core bridge as that would impact all integrating protocols’ ability to issue or execute cross-chain governance.

## Security Considerations

Governance must still be functional during a pause and still has the ultimate authority over the contracts. In that way, pausers only introduce a temporary denial of service protection for users and integrators.
