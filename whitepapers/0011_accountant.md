# Global Accountant

This document describes the motivation, goals, and technical design of the "Global Accountant".

The Global Accountant is a safety feature for registered token bridges.  It is implemented as an [Integrity Checker](0010_integrity_checkers.md) on Wormchain.

## Objectives

Tightly limit the impact an exploit in a connected chain can have on a registered token bridge.  Ensure the number of wrapped tokens capable of being moved out of a chain never exceeds the number sent to that chain, even if that chain was compromised.

## Background

Token Bridges built on Wormhole (e.g. [Portal](0003_token_bridge.md)) allow transferring tokens between any of the various blockchains connected to the bridge.

A core feature of the token bridge is that xAssets are fungible across all chains.  ETH sent from Ethereum to Solana to Polygon is the same as the ETH sent from Ethereum to Aptos to Polygon.  Transfers work by locking native assets (or burning wrapped assets) on the source chain and emitting a Wormhole message that allows the minting of wrapped assets (or unlocking of native assets) on the destination chain.

It is unfortunately not practical to synchronize the state of the token bridge across all chains: If a transfer of Portal-wrapped ETH happens from Solana to Polygon, the token bridge contract on Ethereum is not aware of that transfer and therefore doesn't know where the wrapped assets are currently located.

If any connected chain gets compromised, e.g. through a bug in the rpc nodes, this could cause arbitrary minting of “unbacked” wrapped assets, i.e. wrapped assets that are not backed by native assets.  Those unbacked wrapped assets could then be transferred to the native chain.  Because wrapped assets are fungible, the smart contract on the native chain cannot distinguish those unbacked wrapped assets and would proceed with unlocking the native assets, effectively draining the bridge.

The [Governor](0007_governor.md) limits the maximal impact of such an attack, but cannot fully prevent it.

The core of the issue is the differing trust assumptions and security program maturity of various blockchains.  By design, the Token Bridge is subject to the weakest link.  A holder of a wrapped asset should only be subject to security risks from the asset's native chain as well as the chain where the wrapped asset lives.  They should not be subject to security risks from other chains connected to the bridge.

## Goals

### Primary Goals

- Localize cross-chain risk by ensuring no more wrapped tokens can be moved out of a chain than have been sent to that chain, even if that chain was compromised.

### Nice-to-haves

- Support any token bridge, not just Portal.
- Provide ground-truth statistics for locked and minted assets per chain, transfer activity, etc.
- Minimize smart contract changes.  Ideally, we won't need to change the Token Bridge smart contracts at all.

### Non-Goals

- Protect against any exploit that doesn't go cross-chain e.g. a bug in a smart contract allowing minting of unbacked wrapped assets, the attacker might swap these wrapped assets into other valuable assets through other protocols or through exchanges without going through Wormhole.

## Overview

The `global-accountant` contract on wormchain acts as an [Integrity Checker](0010_integrity_checkers.md).  Guardians submit [pre-observations](0010_integrity_checker.md#pre-observations) to it and only finalize their observations if the `global-accountant` gives the go-ahead.

`global-accountant`keeps track of the tokens locked and minted on each connected blockchain.

Once a Token Bridge message has reached a quorum of pre-observations (i.e. 13 guardians have submitted a pre-observation), it attempts to apply the transfer to the account balances for each chain.  If the transfer doesn't result in a negative account balance, it is allowed to proceed.

## Detailed Design

### Accountant Smart Contract

#### Query methods for signature verification

The global-accountant is a cosmwasm-based smart contract rather than a built-in native cosmos-sdk module.  It needs to expose query methods from the native wormhole module to cosmwasm (like getting the members of a guardian set).  Luckily, extending the contract interface is [fairly straightforward](https://docs.cosmwasm.com/docs/1.0/integration#extending-the-contract-interface).  The `x/wormhole` native module will expose the following query methods:

```rust
#[cw_serde]
#[derive(QueryResponses)]
pub enum WormholeQuery {
    /// Verifies that `vaa` has been signed by a quorum of guardians from valid guardian set.
    #[returns(Empty)]
    VerifyVaa {
        vaa: Binary,
    },

    /// Verifies that `data` has been signed by a guardian from `guardian_set_index` using a correct `prefix`.
    #[returns(Empty)]
    VerifyMessageSignature {
        prefix: Binary,
        data: Binary,
        guardian_set_index: u32,
        signature: Signature,
    },

    /// Returns the number of signatures necessary for quorum for the given guardian set index.
    #[returns(u32)]
    CalculateQuorum { guardian_set_index: u32 },
}
```

#### Submitting observations to wormchain

When a guardian observes a token transfer message from the registered tokenbridge on a particular chain, it submits the observation directly to the accountant.  The contract keeps track of pending token transfers and emits an event once it has received a quorum of observations.  If a quorum of guardians submit their observations within a single tendermint block then the token transfer only needs to wait for one round of tendermint consensus and one round of guardian consensus.

When a guardian submits a transfer (observation) to the contract, the contract will perform several checks:

- Check that the signature is valid and from a guardian in the current guardian set.
- If the transfer does not yet have a quorum of signatures, return a `Pending` status and mark the transfer as having a signature from the guardian.
- If a transfer with the same `(emitter_chain, emitter_address, sequence)` tuple has already been committed, then check that the digest of the transfer matches the digest of the committed transfer.  If they match return a `Committed` status; otherwise return an `Error` status and emit an `Error` event.
- If the transfer has a quorum of signatures, parse the tokenbridge payload and commit the transfer.
- If committing the transfer fails (for example, if one of the accounts does not have a sufficient balance), return an `Error` status and emit an `Error` event.  Otherwise return a `Committed` status and emit a `Committed` event.

If the guardian receives a `Committed` status for a transfer then it can immediately proceed to signing the VAA.  If it receives an `Error` status, then it should halt further processing of the transfer (see the [Handling Rejections](#handling-rejections) section for more details).  If it receives a `Pending` status, then it should add the observation to a local database of pending transfers and wait for an event from the accountant.

#### Observing Events from Wormchain

Each guardian sets up a watcher for the accountant watching for events emitted signalling a determination on a transfer.  An event for a transfer is emitted when there is a quorum of pre-observations.

There are 2 types of events that the guardian must handle:

- Transfer committed - This indicates that an observation has reached quorum and the transfer has been committed.  The guardian must verify the contents of the event and then sign the VAA if they match.  See [Threat Model](#threat-model) for more details.
- Error - This indicates that the accountant ran into an error while processing an observation.  The guardian should halt further processing of the transfer.  See the [Handling Rejections](#handling-rejections) section for more details

#### Handling stuck transfers

It may be possible that some transfers don't receive enough observations to reach quorum.  This can happen if a number of guardians miss the transaction on the origin chain.  Since the accountant runs before the peer to peer gossip layer, it cannot take advantage of the existing automatic retry mechanisms that the guardian network uses.  To deal with this the accountant contract provides a `MissingObservations` query that returns a list of pending transfers for which the accountant does not have an observation from a particular guardian.

Guardians are expected to periodically run this query and re-observe transactions for which the accountant is missing observations from them.

#### Contract Upgrades

Upgrading the contract is performed with a migrate contract governance action from the guardian network. Once quorum+1 guardians (13) have signed the migrate contract governance VAA, the [wormchain client migrate command](https://github.com/wormhole-foundation/wormhole/blob/a846036b6ebff3af6f12ff375f5c3801ada20291/wormchain/x/wormhole/client/cli/tx_wasmd.go#L148) can be ran by anyone with an authorized wallet to submit the valid governance vaa and updated wasm contract to wormchain.

### Account Management

The accountant keeps track of token balances for each blockchain ensuring they never become negative.  When committing a transfer, the accountant parses the tokenbridge payload to get the emitter chain, recipient chain, token chain, and token address.  These values are used to look up the relevant accounts and adjust the balances as needed.

It will have the following accounts:

- Accounts for native tokens ($SOL on solana, $ETH or $wETH on ethereum).  These are classified as assets because they represent actual tokens held by smart contracts on the various chains.  All tokens not issued by the token bridge are considered native tokens, even if they are wrapped tokens issued by another bridge.  If the token is held by the bridge contract rather than burned, it is an asset.  We'll call these "custody accounts".
- Accounts for wormhole wrapped tokens (wrapped SOL on ethereum, wrapped ETH on solana).  These are classified as liabilities.  These tokens are not held by the smart contract and instead represent a liability for the bridge: if a user returns a wrapped token to the bridge, the bridge **must** be able to transfer the native token to that user on its native chain.  We'll call these “wrapped accounts”.

Transferring a native token across the tokenbridge (locking) increases the balance on both the custody account and wrapped account on the destination chain.  Transferring a wrapped token back across the tokenbridge (burning) and subsequently unlocking its native asset decreases the balance on both the wrapped and custody accounts.  Any transfer causing the balance of an account to become negative is invalid and gets rejected.

Each account is identified via a `(chain_id, token_chain, token_address)` tuple.  Custody accounts are those where `chain_id == token_chain`, while the rest are wrapped accounts. See the [governance](#governance) section below for further clarification.

#### Governance

Account balances recorded by the accountant may end up inaccurate due to bugs in the program or exploits of the token bridge.  In these cases, the guardians may issue governance actions to manually modify the balances of the accounts.  These governance actions must be signed by a quorum of guardians.

#### Examples

Example 1: A user sends SOL from solana to ethereum.  In this transaction the  `(Solana, Solana, SOL)` account is debited and the `(Ethereum, Solana, SOL)` account is credited, increasing the balances of both accounts.

Example 2: A user sends wrapped SOL from ethereum to solana.  In this transaction the `(Ethereum, Solana, SOL)` account is debited and the `(Solana, Solana, SOL)` account is credited, decreasing the balances of both accounts.

Example 3: An attacker exploits a bug in the solana contract to print fake wrapped ETH and transfers it over the bridge. This could drain real ETH from the ethereum contract. Thanks to the accounting contract, the total amount drained is limited by the total number of wrapped ETH issued by the solana token bridge contract.  Since the token bridge is still liable for all the legitimately issued wrapped ETH the account balances no longer reflect reality.

Wormhole decides to cover the cost of the hack and transfers the ETH equivalent to the stolen amount to the token bridge contract on ethereum.  To update the accounts, the guardians must issue 2 governance actions: one to increase the balance of the `(Ethereum, Ethereum, ETH)` custody account and one to increase the balance of the `(Solana, Ethereum, ETH)` wrapped account.

Example 4: A user sends wrapped SOL from ethereum to polygon.  Even though SOL is not native to either chain, the transaction is still recorded the same way: the `(Ethereum, Solana, SOL)` wrapped account is debited and the `(Polygon, Solana, SOL)` wrapped account is credited.  The total liability of the token bridge has not changed and instead we have just moved some liability from ethereum to polygon.

#### Creating accounts

Since users can send any arbitrary token over the token bridge, the accountant should automatically create accounts for any new token.  To determine whether the newly created account should be a custody account or a wrapped account, these steps are followed:

- If `emitter_chain == token_chain` and a custody account doesn't already exist, create one.
- If `emitter_chain != token_chain` and a wrapped account doesn't already exist, reject the transaction.  The wrapped account should have been created when the tokens were originally transferred to `emitter_chain`.
- If `token_chain != recipient_chain` and a wrapped account doesn't already exist, create one.
- If `token_chain == recipient_chain` and a custody account doesn't already exist, reject the transaction.  The custody account should have been created when the tokens were first transferred from `recipient_chain`.

### Bootstrapping / Backfills

In order for the accountant to have accurate values for the balances of the various accounts it needs to replay all transactions since the initial launch of the token bridge.  Similarly, it may be necessary to backfill missing transfers if the accountant is disabled for any period of time.

To handle both issues, the accountant has the ability to directly process tokenbridge transfer VAAs.  As long as the VAAs are signed by a quorum of guardians from the latest guardian set, the accountant uses them to update its on-chain state of token balances.

Submitting VAAs does not disable any core checks: a VAA causing the balance of an account to become negative will still be rejected even if it has a quorum of signatures.  This can be used to catch problematic transfers after the fact.

### Handling Rejections

Legitimate transactions should never be rejected so a rejection implies either a bug in the accountant or an actual attack on the bridge.  In both cases manual intervention is required.

#### Alerting

Whenever a guardian observes an error (either via an observation response or via an error event), it will increase a prometheus error counter.  Guardians will be encouraged to set up alerts whenever these metrics change so that they can be notified when an error occurs.

## Deployment Plan

- Deploy wormchain and have a quorum of guardians bring their nodes online.
- Deploy the accountant contract to wormchain and backfill it with all the tokenbridge transfer VAAs that have been emitted since inception.
- Enable the accountant in log-only mode.  In this mode, each guardian will submit its observations to the accountant but will not wait for the transfer to commit before signing the VAA.  This allows time to shake out any remaining implementation issues without impacting the network.
- After confidence is achieved the accountant is running correctly in log-only mode, switch to enforcing mode.  In this mode, guardians will not sign VAAs for token transfers until the transfer is committed by the accountant.  As long as at least a quorum of guardians submit their observations to the accountant, only a super-minority (1/3) is needed to turn on enforcing-mode for it to be effective.

## Security Considerations

### Threat Model

Since the Global Accountant is implemented as an Integrity Checker, it inherits the threat model of Integrity Checkers. Therefore, it is only able to block messages and not create or modify messages.

### Susceptibility to wormchain downtime

Once all token transfers are gated on approval from the accountant, the uptime of the guardian network for those transfers becomes dependent on the uptime of wormchain itself.  Any downtime for wormchain would halt all token transfers until the network is back online (or 2/3+ of the guardians choose to disable the accountant).

In practice, cosmos chains are stable and wormchain is not particularly complex.  However, once the accountant is live, attacking wormchain could shut down all token transfers for the bridge.

### Token Bridge messages cannot reach quorum if 7 or more, but less than 13 Guardians decide to disable Accountant

Due to the nature of the accountant, a guardian that chooses to enable it in enforcing-mode will refuse to sign a token transfer until it sees that the transfer is committed by the accountant.  If a super-minority of guardians choose to not enable the accountant at all (i.e. stop sending their observations to the contract) then we will end up with a stalemate: the guardians with the feature disabled will be unable to reach quorum because they don't have enough signatures while the guardians in enforcing-mode will never observe the transfer being committed by the accountant (and so will never sign the observation).  This is very different from [the governor feature](0007_governor.md), where only a super-minority of guardians have to enable it to be effective.
