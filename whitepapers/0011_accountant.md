# Global Accountant

This document describes the motivation, goals, and technical design of the "Global Accountant".

The Global Accountant is a safety feature for registered token bridges and Native Token Transfer (NTT) deployments. It is implemented as an [Integrity Checker](0010_integrity_checkers.md) on Wormchain.

## Objectives

Tightly limit the impact an exploit in a connected chain can have on registered token bridges and NTT deployments. Ensure the number of wrapped tokens capable of being moved out of a chain never exceeds the number sent to that chain, even if that chain was compromised.

## Background

### Token Bridges
Token Bridges built on Wormhole (e.g. [Portal](0003_token_bridge.md)) allow transferring tokens between any of the various blockchains connected to the bridge.

A core feature of these systems is that xAssets are fungible across all chains. ETH sent from Ethereum to Solana to Polygon is the same as the ETH sent from Ethereum to Aptos to Polygon. Transfers work by locking native assets (or burning wrapped assets) on the source chain and emitting a Wormhole message that allows the minting of wrapped assets (or unlocking of native assets) on the destination chain.

It is unfortunately not practical to synchronize the state of token bridges and NTT protocols across all chains: If a transfer of Portal-wrapped ETH happens from Solana to Polygon, the token bridge contract on Ethereum is not aware of that transfer and therefore doesn't know where the wrapped assets are currently located.

If any connected chain gets compromised, e.g. through a bug in the RPC nodes, this could cause arbitrary minting of "unbacked" wrapped assets, i.e. wrapped assets that are not backed by native assets. Those unbacked wrapped assets could then be transferred to the native chain. Because wrapped assets are fungible, the smart contract on the native chain cannot distinguish those unbacked wrapped assets and would proceed with unlocking the native assets, effectively draining the bridge.

The [Governor](0007_governor.md) limits the maximal impact of such an attack on the token bridges, but cannot fully prevent it.

The core of the issue is the differing trust assumptions and security program maturity of various blockchains. By design, Token Bridges are subject to the weakest link. A holder of a wrapped asset should only be subject to security risks from the asset's native chain as well as the chain where the wrapped asset lives. They should not be subject to security risks from other chains connected to the bridge.

### Native Token Transfers
Native Token Transfer (NTT) deployments allow certain assets to be transferred across a subset of chains, according to the deployment details.
Additional details can be found in that project's [README](https://github.com/wormhole-foundation/native-token-transfers/blob/main/README.md)

## Goals

### Primary Goals

- Localize cross-chain risk by ensuring no more wrapped tokens can be moved out of a chain than have been sent to that chain, even if that chain was compromised.
- Support both Token Bridge and Native Token Transfer (NTT) protocols with unified accounting principles.

### Nice-to-haves

- Support any token bridge, not just Portal.
- Provide ground-truth statistics for locked and minted assets per chain, transfer activity, etc.
- Minimize smart contract changes. Ideally, we won't need to change the Token Bridge or NTT smart contracts at all.

### Non-Goals

- Protect against any exploit that doesn't go cross-chain e.g. a bug in a smart contract allowing minting of unbacked wrapped assets, the attacker might swap these wrapped assets into other valuable assets through other protocols or through exchanges without going through Wormhole.

## Overview

The Global Accountant consists of two parallel accounting systems:

1. **Base Accountant**: Handles traditional Token Bridge transfers (Portal)
2. **NTT Accountant**: Handles Native Token Transfer protocol messages

Both systems act as [Integrity Checkers](0010_integrity_checkers.md) on wormchain. Guardians submit [pre-observations](0010_integrity_checker.md#pre-observations) to the appropriate accountant and only finalize their observations if the accountant gives the go-ahead.

Each accountant keeps track of the tokens locked and minted on each connected blockchain for their respective message type.

Once a transfer message has reached a quorum of pre-observations (i.e. 13 guardians have submitted a pre-observation), the appropriate accountant attempts to apply the transfer to the account balances for each chain. If the transfer doesn't result in a negative account balance, it is allowed to proceed.

## Detailed Design

### Architecture

The Global Accountant supports both Token Bridge and NTT protocols through separate but parallel accounting systems:

#### Base Accountant
- Handles Token Bridge transfers (Portal)
- Uses existing Token Bridge emitter detection logic
- Maintains separate contract: `accountantContract`
- Uses `accountantKeyPath` and `accountantKeyPassPhrase` for authentication

#### NTT Accountant  
- Handles Native Token Transfer protocol messages
- Detects NTT messages through payload analysis and known emitter configurations
- Maintains separate contract: `accountantNttContract`
- Uses `accountantNttKeyPath` and `accountantNttKeyPassPhrase` for authentication
- Supports both direct NTT endpoint transfers and Automatic Relayer (AR) forwarded transfers

### Message Detection and Routing

The accountant includes message detection logic to route transfers to the appropriate accounting system:

#### Token Bridge Detection
- Uses known Token Bridge emitter mappings
- Validates payload format for Token Bridge transfers
- Routes to Base Accountant for processing

#### NTT Detection
- **Direct NTT Transfers**: Detects messages from known NTT endpoint contracts by:
  - Checking emitter chain/address against configured NTT emitters
  - Validating NTT payload format (WH_PREFIX + NTT_PREFIX at specific offsets)
- **Automatic Relayer Transfers**: Detects NTT messages forwarded through Automatic Relayers by:
  - Checking if emitter is a known Automatic Relayer
  - Parsing AR payload to extract original sender address
  - Validating that sender is a known NTT emitter
  - Extracting and validating embedded NTT payload

### Accountant Smart Contracts

#### Query methods for signature verification

Both accountants are cosmwasm-based smart contracts rather than built-in native cosmos-sdk modules. They expose query methods from the native wormhole module to cosmwasm. The `x/wormhole` native module exposes the following query methods:

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

When a guardian observes a transfer message, it first determines whether it's a Token Bridge or NTT transfer, then submits the observation to the appropriate accountant:

- **Token Bridge transfers** → submitted to Base Accountant
- **NTT transfers** → submitted to NTT Accountant

Each accountant maintains separate:
- Processing workers (`baseWorker` / `nttWorker`)
- Event watchers (`baseWatcher` / `nttWatcher`) 
- Submission channels and message prefixes
- Per-emitter enforcement configurations

The contract processing logic is the same for both accountants:

- Check that the signature is valid and from a guardian in the current guardian set.
- If the transfer does not yet have a quorum of signatures, return a `Pending` status and mark the transfer as having a signature from the guardian.
- If a transfer with the same `(emitter_chain, emitter_address, sequence)` tuple has already been committed, then check that the digest of the transfer matches the digest of the committed transfer. If they match return a `Committed` status; otherwise return an `Error` status and emit an `Error` event.
- If the transfer has a quorum of signatures, parse the payload (Token Bridge or NTT format) and commit the transfer.
- If committing the transfer fails (for example, if one of the accounts does not have a sufficient balance), return an `Error` status and emit an `Error` event. Otherwise return a `Committed` status and emit a `Committed` event.

If the guardian receives a `Committed` status for a transfer then it can immediately proceed to signing the VAA. If it receives an `Error` status, then it should halt further processing of the transfer. If it receives a `Pending` status, then it should add the observation to a local database of pending transfers and wait for an event from the accountant.

#### Observing Events from Wormchain

Each guardian sets up watchers for both accountants, watching for events emitted signalling determinations on transfers. Events can come from either the Base Accountant or NTT Accountant depending on the transfer type.

There are 2 types of events that the guardian must handle from each accountant:

- Transfer committed - This indicates that an observation has reached quorum and the transfer has been committed. The guardian must verify the contents of the event and then sign the VAA if they match.
- Error - This indicates that the accountant ran into an error while processing an observation. The guardian should halt further processing of the transfer.

#### Handling stuck transfers

It may be possible that some transfers don't receive enough observations to reach quorum. This can happen if a number of guardians miss the transaction on the origin chain. Since the accountant runs before the peer to peer gossip layer, it cannot take advantage of the existing automatic retry mechanisms that the guardian network uses. To deal with this, both accountant contracts provide a `MissingObservations` query that returns a list of pending transfers for which the accountant does not have an observation from a particular guardian.

Guardians are expected to periodically run this query against both accountants and re-observe transactions for which the accountants are missing observations from them.

#### Configuration and Deployment Flexibility

The dual accountant system supports flexible deployment configurations:

- **Base Only**: Only Token Bridge accounting enabled (`accountantContract` configured, `accountantNttContract` empty)
- **NTT Only**: Only NTT accounting enabled (`accountantNttContract` configured, `accountantContract` empty)  
- **Both**: Full accounting for both protocols (both contracts configured)
- **Neither**: Accountant disabled entirely (both contracts empty)

This allows for gradual rollout and protocol-specific deployments.

#### Contract Upgrades

Upgrading either contract is performed with a migrate contract governance action from the guardian network. Once quorum+1 guardians (13) have signed the migrate contract governance VAA, the [wormchain client migrate command](https://github.com/wormhole-foundation/wormhole/blob/a846036b6ebff3af6f12ff375f5c3801ada20291/wormchain/x/wormhole/client/cli/tx_wasmd.go#L148) can be ran by anyone with an authorized wallet to submit the valid governance vaa and updated wasm contract to wormchain.

### Account Management

Both the Base and NTT Accountants use the same account management principles. Each accountant keeps track of token balances for each blockchain ensuring they never become negative. When committing a transfer, the accountant parses the respective payload format (Token Bridge or NTT) to get the emitter chain, recipient chain, token chain, and token address. These values are used to look up the relevant accounts and adjust the balances as needed.

Both accountants maintain the following accounts:

- Accounts for native tokens -- not to be confused with Native Token Transfers -- include e.g. $SOL on Solana, and $ETH or $wETH on Ethereum. These are classified as assets because they represent actual tokens held by smart contracts on the various chains. All tokens not issued by the token bridge are considered native tokens, even if they are wrapped tokens issued by another bridge. If the token is held by the bridge contract rather than burned, it is an asset. We'll call these "custody accounts".
- Accounts for wormhole wrapped tokens (wrapped SOL on Ethereum, wrapped ETH on Solana). These are classified as liabilities. These tokens are not held by the smart contract and instead represent a liability for the bridge: if a user returns a wrapped token to the bridge, the bridge **must** be able to transfer the native token to that user on its native chain. We'll call these "wrapped accounts".

Transferring a native token across either protocol (locking) increases the balance on both the custody account and wrapped account on the destination chain. Transferring a wrapped token back across either protocol (burning) and subsequently unlocking its native asset decreases the balance on both the wrapped and custody accounts. Any transfer causing the balance of an account to become negative is invalid and gets rejected.

Each account is identified via a `(chain_id, token_chain, token_address)` tuple. Custody accounts are those where `chain_id == token_chain`, while the rest are wrapped accounts. See the [governance](#governance) section below for further clarification.

#### Governance

Account balances recorded by either accountant may end up inaccurate due to bugs in the program or exploits of the respective protocols. In these cases, the guardians may issue governance actions to manually modify the balances of the accounts in either the Base or NTT Accountant. These governance actions must be signed by a quorum of guardians.

#### Examples

Example 1: A user sends SOL from Solana to Ethereum. In this transaction the `(Solana, Solana, SOL)` account is debited and the `(Ethereum, Solana, SOL)` account is credited, increasing the balances of both accounts.

Example 2: A user sends wrapped SOL from Ethereum to Solana. In this transaction the `(Ethereum, Solana, SOL)` account is debited and the `(Solana, Solana, SOL)` account is credited, decreasing the balances of both accounts.

Example 3: An attacker exploits a bug in the Solana contract to print fake wrapped ETH and transfers it over the bridge. This could drain real ETH from the Ethereum contract. Thanks to the accounting contract, the total amount drained is limited by the total number of wrapped ETH issued by the solana token bridge contract. Since the token bridge is still liable for all the legitimately issued wrapped ETH the account balances no longer reflect reality.

Wormhole decides to cover the cost of the hack and transfers the ETH equivalent to the stolen amount to the token bridge contract on Ethereum. To update the accounts, the guardians must issue 2 governance actions: one to increase the balance of the `(Ethereum, Ethereum, ETH)` custody account and one to increase the balance of the `(Solana, Ethereum, ETH)` wrapped account.

Example 4: A user sends wrapped SOL from Ethereum to Polygon. Even though SOL is not native to either chain, the transaction is still recorded the same way: the `(Ethereum, Solana, SOL)` wrapped account is debited and the `(Polygon, Solana, SOL)` wrapped account is credited. The total liability of the token bridge has not changed and instead we have just moved some liability from Ethereum to Polygon.

#### Creating accounts

Since users can send any arbitrary token over the token bridge, the accountant should automatically create accounts for any new token. To determine whether the newly created account should be a custody account or a wrapped account, these steps are followed:

- If `emitter_chain == token_chain` and a custody account doesn't already exist, create one.
- If `emitter_chain != token_chain` and a wrapped account doesn't already exist, reject the transaction. The wrapped account should have been created when the tokens were originally transferred to `emitter_chain`.
- If `token_chain != recipient_chain` and a wrapped account doesn't already exist, create one.
- If `token_chain == recipient_chain` and a custody account doesn't already exist, reject the transaction. The custody account should have been created when the tokens were first transferred from `recipient_chain`.

### Enforcement Configuration

Both accountants support per-emitter enforcement configuration:

- **Enforcing mode**: Guardians will not sign VAAs until the transfer is committed by the accountant
- **Log-only mode**: Observations are submitted but VAAs are signed regardless of accountant status

This allows for:
- Gradual rollout of enforcement per protocol
- Different security postures for different emitters
- Testing and validation in production environments

### Bootstrapping / Backfills

In order for each accountant to have accurate values for the balances of the various accounts, it needs to replay all transactions since the initial launch of the respective protocol. Similarly, it may be necessary to backfill missing transfers if either accountant is disabled for any period of time.

To handle both issues, each accountant has the ability to directly process transfer VAAs for their respective protocols. As long as the VAAs are signed by a quorum of guardians from the latest guardian set, the accountant uses them to update its on-chain state of token balances.

Submitting VAAs does not disable any core checks: a VAA causing the balance of an account to become negative will still be rejected even if it has a quorum of signatures. This can be used to catch problematic transfers after the fact.

### Handling Rejections

Legitimate transactions should never be rejected so a rejection implies either a bug in the accountant or an actual attack on the bridge. In both cases manual intervention is required.

#### Alerting

Whenever a guardian observes an error (either via an observation response or via an error event), it will increase a Prometheus error counter. Guardians will be encouraged to set up alerts whenever these metrics change so that they can be notified when an error occurs from either accountant.

## Deployment Plan

- Deploy Wormchain and have a quorum of guardians bring their nodes online.
- Deploy both accountant contracts to Wormchain and backfill them with their respective transfer VAAs that have been emitted since inception.
- Enable both accountants in log-only mode. In this mode, each guardian will submit observations to the appropriate accountant but will not wait for transfers to commit before signing VAAs. This allows time to shake out any remaining implementation issues without impacting the network.
- After confidence is achieved that both accountants are running correctly in log-only mode, switch to enforcing mode per protocol as needed. As long as at least a quorum of guardians submit their observations to the accountants, only a super-minority (1/3) is needed to turn on enforcing-mode for it to be effective.

## Security Considerations

### Threat Model

Since both Global Accountants are implemented as Integrity Checkers, they inherit the threat model of Integrity Checkers. Therefore, they are only able to block messages and not create or modify messages.

### Protocol Isolation

The dual accountant architecture provides protocol isolation benefits:

- The Base and NTT accountants can be enabled independently
- Token Bridge exploits cannot affect NTT accounting and vice versa
- Each protocol can have different enforcement policies
- Issues in one accountant don't necessarily impact the other

### Susceptibility to wormchain downtime

Once transfers are gated on approval from the respective accountants, the uptime of the guardian network for those transfers becomes dependent on the uptime of wormchain itself. Any downtime for wormchain would halt transfers until the network is back online (or 2/3+ of the guardians choose to disable the relevant accountant).

In practice, cosmos chains are stable and wormchain is not particularly complex. However, once the accountants are live, attacking wormchain could shut down transfers for both protocols.

### Token Bridge and NTT messages cannot reach quorum if 7 or more, but less than 13 Guardians decide to disable Accountant

Due to the nature of the accountants, a guardian that chooses to enable them in enforcing-mode will refuse to sign a transfer until it sees that the transfer is committed by the appropriate accountant. If a super-minority of guardians choose to not enable an accountant at all (i.e. stop sending their observations to the contract) then we will end up with a stalemate: the guardians with the feature disabled will be unable to reach quorum because they don't have enough signatures while the guardians in enforcing-mode will never observe the transfer being committed by the accountant (and so will never sign the observation). This is very different from [the governor feature](0007_governor.md), where only a super-minority of guardians have to enable it to be effective.

### Configuration Complexity

The dual accountant system introduces configuration complexity:

- Guardians must properly configure both accountants
- Message routing logic must correctly identify transfer types
- Monitoring and alerting must cover both systems
