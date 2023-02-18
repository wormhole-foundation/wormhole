# Global Accountant

This document describes the motivation, goals, and technical design of “Global Accountant”.

The Global Accountant is a safety feature for registered token bridges. It is implemented as an [Integrity Checker](0010_integrity_checkers.md) on Wormchain.

## Objectives

Tightly limit the impact that an exploit in a connected chain can have on a registered token bridge. Ensure that no more wrapped tokens can be moved out of a chain than have been sent to that chain, even if that chain was completely compromised.

## Background

Token Bridges built on Wormhole (e.g. [Portal](0003_token_bridge.md)) allow anyone to transfer tokens between any of the various L1 blockchains connected to the bridge.

A core feature of the token bridge is that xAssets are fungible across all chains. ETH sent from Ethereum to Solana to Polygon is the same as the ETH sent from Ethereum to Aptos to Polygon. Transfers work by locking native assets (or burning wrapped assets) on the source chain and emitting a Wormhole message that allows the minting of wrapped assets (or unlocking of native assets) on the destination chain.

It is unfortunately not practical to synchronize the state of the token bridge across all chains: If a transfer of Portal-wrapped ETH happens from Solana to Polygon, the token bridge contract on Ethereum is not aware of that transfer and therefore doesn’t know on which chain the wrapped assets currently are.

If any connected chain gets compromised, e.g. through a bug in the rpc nodes, this could cause the arbitrarily mint of “unbacked” wrapped assets, i.e. wrapped assets that are not backed by native assets. Those unbacked wrapped assets could then be transferred to the native chain. Because wrapped assets are fungible, the smart contract on the native chain cannot distinguish those unbacked wrapped assets and would proceed with unlocking the native assets, effectively draining the bridge.

The [Governor](0007_governor.md) limits the maximal impact of such an attack, but cannot fully prevent it.

The core of the problem is that the various L1s operate under different trust assumptions and have differently mature security programs. By design, the Token Bridge is subject to the weakest link. But a holder of a wrapped asset should only be subject to security risks from the chain where the asset is native as well as the chain where they are holding the wrapped asset on. They should not be subject to security risks from other chains connected to the bridge.

## Goals

### Primary Goals

- Localize cross-chain risk by ensuring that no more wrapped tokens can be moved out of a chain than have been sent to that chain, even if that chain was compromised.

### Nice-to-haves

- Minimize smart contract changes.  Ideally, we won’t need to change the Token Bridge smart contracts at all.
- Support any token bridge, not just Portal.
- Provide ground-truth statistics for locked and minted assets per chain, transfer activity, etc.

### Non-Goals

- Protect against any exploit that doesn’t go cross-chain. E.g. if there is a bug in a smart contract that allows minting unbacked wrapped assets, the attacker might be able to swap these wrapped assets into other valuable assets through other protocols or through exchanges without going through Wormhole.

## Overview

The `global-accountant` contract on wormchain acts as an Integrity Checker. This means that Guardians will submit pre-observations to it and only finalize their observations if the `global-accountant` gives the go-ahead.

`global-accountant`keeps track of the tokens locked and minted on each connected L1.

Once a Token Bridge message has reached a quorum of pre-observations (i.e. 13 guardians have submitted a pre-observation), it will try to apply the transfer to the account balances for each chain. If the transfer doesn’t result in a negative account, it gives the go-ahead.

## Detailed Design

### Accountant Smart Contract

#### Query methods for signature verification

Since the global-accountant is a cosmwasm-based smart contract rather than a built-in native cosmos-sdk module, we will need to expose some query methods from the native wormhole module to cosmwasm (like getting the members of a guardian set).  Luckily, extending the contract interface is [fairly straightforward](https://docs.cosmwasm.com/docs/1.0/integration#extending-the-contract-interface).  The `x/wormhole` native module will expose the following query methods:

```rust
#[cw_serde]
#[derive(QueryResponses)]
pub enum WormholeQuery {
    /// Verifies that `data` has been signed by a quorum of guardians from `guardian_set_index`.
    #[returns(Empty)]
    VerifyQuorum {
        data: Binary,
        guardian_set_index: u32,
        signatures: Vec<Signature>,
    },

    /// Verifies that `data` has been signed by a guardian from `guardian_set_index`.
    #[returns(Empty)]
    VerifySignature {
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

When a guardian observes a token transfer message from the registered tokenbridge on a particular chain, it will submit that observation directly to the accountant.  The contract will keep track of pending token transfers and emit an event once it has received a quorum of observations.  If a quorum of guardians submit their observations within a single tendermint block then the token transfer only needs to wait for one round of tendermint consensus and one round of guardian consensus.

When a guardian submits an observation to the contract, the contract will perform several checks:

- Check that the signature is valid and from a guardian in the current guardian set.
- If a transfer with the same `(emitter_chain, emitter_address, sequence)` tuple has already been committed, then check that the digest of the observation matches the digest of the committed transfer.  If they match return a `Committed` status; otherwise return an `Error` status and emit an `Error` event.
- If the transfer has not been committed, mark the observation as having a signature from the guardian.
- If the observation does not yet have a quorum of signatures, returning a `Pending` status.
- If the observation has a quorum of signatures, parse the tokenbridge payload and commit the transfer.
- If committing the transfer fails (for example, if one of the accounts does not have a sufficient balance) then return an `Error` status and emit an `Error` event.  Otherwise return a `Committed` status and emit a `Committed` event.

If the guardian receives a `Committed` status for a transfer then it can immediately proceed to signing the VAA.  If it receives an `Error` status, then it should halt further processing of the transfer (see the [Handling Rejections](https://www.notion.so/tbjump-copy-Accounting-v3-11572f647daf46f1a1e7d3ac80050ac0) section for more details).  If it receives a `Pending` status, then it should add the observation to a local database of pending transfers and wait for an event from the accountant.

#### Observing Events from Wormhole Chain

In addition to submitting observations to the accountant, each guardian must also set up a watcher to watch for transfers that complete asynchronously, which can happen when no observation has reached quorum for a particular `(emitter_chain, emitter_address, seuence)` tuple.

There are 2 types of events that the guardian must handle:

- Transfer committed - This indicates that an observation has reached quorum and the transfer has been committed.  The guardian must independently verify the contents of the event and then sign the VAA if they match.  See [this section](https://www.notion.so/tbjump-copy-Accounting-v3-11572f647daf46f1a1e7d3ac80050ac0) for more details.
- Error - This indicates that the accountant ran into an error while processing an observation.  The guardian should halt further processing of the transfer.  See the [Handling Rejections](https://www.notion.so/tbjump-copy-Accounting-v3-11572f647daf46f1a1e7d3ac80050ac0) section for more details

#### Handling stuck transfers

It may be possible that some transfers don’t receive enough observations to reach quorum.  This can happen if a number of guardians miss the transaction on the origin chain.  Since the accountant runs before the p2p gossip layer, it cannot take advantage of the existing automatic retry mechanisms that the guardian network uses.  To deal with this the accountant contract provides a `MissingObservations` query that returns a list of pending transfers for which the accountant does not have an observation from a particular guardian.

Guardians are expected to periodically run this query and re-observe txs for which the accountant is missing observations from them.

#### Contract Upgrades

Upgrading the contract will be done via a governance action from the guardian network with the same mechanism we use for the existing wormhole cosmwasm contracts.

### Account Management

The accountant will keep track of the various token balances on each L1 and ensure that they never become negative.  When committing a transfer, the accountant will parse the tokenbridge payload to figure out the emitter chain, recipient chain, token chain, and token address.  These values will then be used to look up the relevant accounts and adjust the balances as needed.

We will have the following accounts:

- Accounts for native tokens on L1 chains ($SOL on solana, $ETH or $wETH on ethereum).  These are classified as assets because they represent actual tokens held by the smart contracts on the various chains.  All tokens that are not issued by the token bridge itself are considered native tokens, even if they are wrapped tokens issued by another bridge.  If the token is held by the bridge contract rather than burned, then it is an asset.  We’ll call these “custody accounts”.
- Accounts for wrapped tokens (wrapped SOL on ethereum, wrapped ETH on solana).  These are classified as liabilities.  These tokens are not held by the smart contract and instead represent a liability for the bridge: if a user returns a wrapped token to the bridge, the bridge **must** be able to transfer the native token to that user on its native chain.  We’ll call these “wrapped accounts”.

Each account is identified via a `(chain_id, token_chain, token_address)` tuple.  Custody accounts are those where `chain_id == token_chain`, while the rest are wrapped accounts.  Locking a native token or minting a wrapped token will increase the balance of the relevant account while unlocking a native token or burning a wrapped token will decrease the balance.  Transfers that would cause the balance of an account to become negative will be rejected.

#### Governance

The account balances recorded by the accountant  may end up becoming inaccurate due to bugs in the program or exploits of the token bridge.  In these cases, the guardians may issue governance actions to manually modify the balances of the accounts.  These governance actions must be signed by a quorum of guardians.

#### Examples

Example 1: A user sends SOL from solana to ethereum.  In this transaction the  `(Solana, Solana, SOL)` account is debited and the `(Ethereum, Solana, SOL)` account is credited, increasing the balances of both accounts.

Example 2: A user sends wrapped SOL from ethereum to solana.  In this transaction the `(Ethereum, Solana, SOL)` account is debited and the `(Solana, Solana, SOL)` account is credited, decreasing the balances of both accounts.

Example 3: An attacker exploits a bug in the solana contract to print fake wrapped ETH and transfers it over the bridge, draining real ETH from the ethereum contract. Thanks to the accounting contract, the total amount drained is limited by the total number of wrapped ETH issued by the solana token bridge contract.  Since the token bridge is still liable for all the legitimately issued wrapped ETH the account balances no longer reflect reality.

Wormhole decides to cover the cost of the hack and transfers the ETH equivalent to the stolen amount to the token bridge contract on ethereum.  To update the accounts, the guardians must issue 2 governance actions: one to increase the balance of the `(Ethereum, Ethereum, ETH)` custody account and one to increase the balance of the `(Solana, Ethereum, ETH)` wrapped account.

Example 4: A user sends wrapped SOL from ethereum to polygon.  Even though SOL is not native to either chain, the transaction is still recorded the same way: the `Ethereum/SOL` mint account is debited and the `Polygon/SOL` mint account is credited.  The total liability of the token bridge has not changed and instead we have just moved some liability from ethereum to polygon.

#### Creating accounts

Since users can send any arbitrary token over the token bridge we need to be able to automatically create accounts any time we see a new token.  In order to determine whether the newly created account should be a custody account or a mint account, we follow these steps:

- If `emitter_chain == token_chain` and a custody account doesn’t already exist, create one.
- If `emitter_chain != token_chain` and a wrapped account doesn’t already exist, reject the transaction.  The wrapped account should have been created when the tokens were originally transferred to `emitter_chain`.
- If `token_chain != recipient_chain` and a wrapped account doesn’t already exist, create one.
- If `token_chain == recipient_chain` and a custody account doesn’t already exist, reject the transaction.  The custody account should have been created when the tokens were first transferred from `recipient_chain`.

### Bootstrapping / Backfills

In order for the accountant to have accurate values for the balances of the various accounts we will need to replay all transactions since the initial launch of the token bridge.  Similarly it may be necessary to backfill missing transfers if the accountant is disabled for any period of time.

To handle both these issues, the accountant will have the ability to directly process tokenbridge transfer VAAs.  As long as the VAAs are signed by a quorum of guardians from the latest guardian set, the accountant will use them to update its on-chain state of token balances.

Submitting VAAs will not disable any of the core checks though: a VAA that would cause the balance of an account to become negative will still be rejected even if it has a quorum of signatures.  This can be used to catch problematic transfers after the fact.

### Handling Rejections

Legitimate transactions should never be rejected so a rejection implies either a bug in the accountant or an actual attack on the bridge.  In both cases manual intervention is required.

#### Alerting

Whenever a guardian observes an error (either via an observation response or via an error event), it will increase a prometheus error counter.  Guardians will be encouraged to set up alerts whenever these metrics change so that they can be notified when an error occurs.

## Deployment Plan

- Deploy wormchain and have a quorum of guardians bring their nodes online.
- Deploy the accountant contract to wormchain and backfill it with all the tokenbridge transfer VAAs that have been emitted since inception.
- Enable the accountant in log-only mode.  In this mode, each guardian will submit its observations to the accountant but will not wait for the transfer to commit before signing the VAA.  This will give us some time to shake out any remaining issues with the implementation without impacting the network.
- Once we have been running in log-only mode for some time and have built up confidence that it is operating correctly, switch to enforcing-mode.  In this mode, guardians will not sign VAAs for token transfers until the transfer is committed by the accountant.  As long as at least a quorum of guardians submit their observations to the accountant, we only need a super-minority (1/3) to turn on enforcing-mode for it to be effective.

## Security Considerations

### Threat Model

Since the Global Accountant is implemented as an Integrity Checker, it inherits the threat model of Integrity Checkers. Therefore, it is only able to block messages, but it cannot create or modify messages.

### Susceptibility to wormchain downtime

Once all token transfers become gated on approval from the accountant, the uptime of the guardian network for those transfers will become dependent on the uptime of wormchain itself.  Any downtime for wormchain would halt all token transfers until the network is back online (or 2/3 of the guardians choose to disable the accountant).

In practice, cosmos chains are relatively stable and wormchain is not particularly complex.  However, once the accountant is live attacking wormchain will be a viable way to shut down token transfers for the entire bridge.

### Token Bridge messages cannot reach quorum if 7 or more, but less than 13 Guardians decide to disable Accountant

Due to the nature of the accountant, a guardian that chooses to enable it in enforcing-mode will refuse to sign a token transfer until it sees that the transfer is committed by the accountant.  If a super-minority of guardians choose to not enable the accountant at all (i.e. stop sending their observations to the contract) then we will end up with a stalemate: the guardians with the feature disabled will be unable to reach quorum because they don’t have enough signatures while the guardians in enforcing-mode will never observe the transfer being committed by the accountant (and so will never sign the observation).  This is very different from the governor feature, where only a super-minority of guardians have to enable it for it to be effective.