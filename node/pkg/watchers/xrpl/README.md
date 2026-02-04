# XRPL Guardian Watcher - Tech Spec

# Objective

As of this writing [XRPL](https://xrpl.org/) does not feature any smart contract-like functionality. In order to support XRP and other XRPL assets as NTTs, a service must artificially generate NTT transfer payloads from payment data.

# **Background**

NTT messages are wrapped in an outermost [TransceiverMessage](https://github.com/wormhole-foundation/native-token-transfers/blob/main/docs/Transceiver.md#transceivermessage) format by the configured Transceiver. An NTT Manager’s transfer payload will be encoded in the `ntt_manager_payload` field following the [NativeTokenTransfer](https://github.com/wormhole-foundation/native-token-transfers/blob/main/docs/NttManager.md#nativetokentransfer) format. Together, for a Wormhole Transceiver, this looks like

```go
[4]byte  prefix = 0x9945FF10           //
[32]byte source_ntt_manager_address    // WH universal address of the source manager
[32]byte recipient_ntt_manager_address // WH universal address of the destination manager
uint16   ntt_manager_payload_length    // 143
--- (payload)
[32]byte  id                           // a unique message identifier
[32]byte  sender                       // original message sender address
[4]byte   prefix = 0x994E5454          // 0x99'N''T''T'
uint8     decimals                     // number of decimals for the amount
uint64    amount                       // amount being transferred
[32]byte  source_token                 // source chain token address
[32]byte  recipient_address            // the address of the recipient
uint16    recipient_chain              // the Wormhole Chain ID of the recipient
---
uint16   transceiver_payload_length    // 0
[]byte   transceiver_payload           // n/a
```

There are also standard messages for [Initialize Transceiver](https://github.com/wormhole-foundation/native-token-transfers/blob/main/docs/Transceiver.md#initialize-transceiver) and [Transceiver (Peer) Registration](https://github.com/wormhole-foundation/native-token-transfers/blob/main/docs/Transceiver.md#transceiver-peer-registration).

In the Wormhole context, these would be in the `payload` field of a [VAA](https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0001_generic_message_passing.md#detailed-design).

XRP Ledger documentation can be found [here](https://xrpl.org/docs). Notably

- [Accounts](https://xrpl.org/docs/concepts/accounts) are identified by their [address](https://xrpl.org/docs/references/protocol/data-types/basic-data-types#addresses), which contains 20-bytes derived from the account’s master public key.
- [Payment](https://xrpl.org/docs/references/protocol/transactions/types/payment) transactions are used for direct XRP payments as well as [Multi-Purpose Tokens (MPTs)](https://xrpl.org/docs/concepts/tokens/fungible-tokens/multi-purpose-tokens).
- MPTs have issuer accounts that use payment transactions to mint new tokens and cannot hold their own tokens. Payments of tokens made to its issuer account are burned. In this way, a payment back to an issuer account is the same as a transfer back to a mint/burn configured NTT manager that burns the tokens as part of the transaction.

# **Goals**

- Support artificially generating each of the standard Wormhole NTT messages from XRPL.
- Support permissionless onboarding of new NTT deployments.

# Non-Goals

- Relaying, which will be handled by [Executor](https://github.com/wormholelabs-xyz/example-messaging-executor).
- Unlocking / minting tokens on XRPL, which will be handled by a separate design leveraging the [Manager Service](https://github.com/wormholelabs-xyz/wormhole/blob/doge/whitepapers/0016_manager_service.md).

# **Overview**

The Wormhole guardian node will implement a new [watcher](https://github.com/wormhole-foundation/wormhole/tree/main/node/pkg/watchers) for XRPL which will allow for the initialization, registration, and native token transfers of XRPL assets.

# Detailed Design

Until this writing, a watcher has had the sole purpose of generating VAA bodies based on an artifact from a Core smart contract, such as the [LogMessagePublished](https://github.com/wormhole-foundation/wormhole/blob/e11926a849391e8a035c69fc52f4efb3205258fd/ethereum/contracts/Implementation.sol#L12) event in the EVM implementation, and confirming a certain [Consistency Level](https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0001_generic_message_passing.md#consistency-levels) had been reached. However, without smart contracts, the XRPL watcher will have to take a different approach and implement some of the logic that took place on chain.

For NTT, this requires establishing a protocol of XRPL [Payments](https://xrpl.org/docs/references/protocol/transactions/types/payment#payment) that can be transformed into the applicable NTT Wormhole Transceiver messages - Initialize Transceiver, Transceiver (Peer) Registration, and NativeTokenTransfer TransceiverMessage.

## Technical Details

On any transaction, the watcher MUST verify that the transaction result is [`tesSUCCESS`](https://xrpl.org/docs/references/protocol/transactions/transaction-results/tes-success) in a validated ledger.

For the purposes of NTT, any XRPL Account can potentially be an NTT Manager. However,

<aside>
🚧

This watcher also requires tracking of registered **Manager Accounts** - i.e. XRPL accounts which have been transferred to the multisig of the [Delegated Manager Set](https://github.com/wormholelabs-xyz/wormhole/blob/doge/whitepapers/0016_manager_service.md) for XRPL - and their **registered token**. The details of this may change if the [Batch](https://xrpl.org/resources/known-amendments#batch) amendment is accepted. For now, consider that this will be a hardcoded list during the “alpha” phase of this feature rollout.

</aside>

<aside>
💡

Remember that all fields must be generated deterministically based on the transaction information!

</aside>

### Initialize Transceiver

<aside>
🚧

TODO

</aside>

### Transceiver (Peer) Registration

<aside>
🚧

TODO

</aside>

### NativeTokenTransfer TransceiverMessage

The watcher must be able to generate a valid VAA body containing a NativeTokenTransfer TransceiverMessage payload for any properly registered Manager Account.

Upon a Payment to a registered Manager Account, the guardian watcher MUST

- Verify that the transaction is `validated`
- Verify that the transaction result is `tesSUCCESS`
- Verify that the transaction type is `Payment`
- Verify that the `from_decimals` matches…
  - `6` for XRP
  - [no check for Trust Line tokens, this may be arbitrarily specified by the sender]
  - `AssetScale` for MPT tokens
- Scale / trim the amount to `decimals = min(min(8, fromDecimals), toDecimals)`.
- Ensure that the resulting amount does not exceed `u64` max.

Further verification of tokens or registrations is not needed, as each manager account + token pair will emit messages from a different transceiver.

Then the watcher must generate the VAA body and NativeTokenTransfer Transceiver message payload.

```go
// VAA Body
uint32   timestamp                     // ledger close time
uint32   nonce                         // 0
uint16   emitter_chain                 // watcher's chain ID
[32]byte emitter_address               // keccak256(source_ntt_manager_address + source_token)
uint64   sequence                      // (ledgerIndex << 32) | txIndex
uint8    consistency_level             // 0
// Payload
[4]byte  prefix = 0x9945FF10           //
[32]byte source_ntt_manager_address    // same as destination account
[32]byte recipient_ntt_manager_address // ** memo data
uint16   ntt_manager_payload_length    // 143
// - Manager Payload
[32]byte  id                           // sequence
[32]byte  sender                       // account (payment sender)
[4]byte   prefix = 0x994E5454          // 0x99'N''T''T'
uint8     decimals                     // ** memo data
uint64    amount                       // delivered amount * see note
[32]byte  source_token                 // * see note
[32]byte  recipient_address            // ** memo data
uint16    recipient_chain              // ** memo data
// - Transceiver payload
uint16   transceiver_payload_length    // 0
```

For the `amount` and `source_token` fields, XRPL supports effectively 3 different [types of payments](https://xrpl.org/docs/references/protocol/transactions/types/payment#types-of-payments) for our purposes with 3 different token / payment standards. XRPL NTT will enumerate these types by the first byte of the `source_token`:

- `0` - Direct XRP Payments ([example](https://livenet.xrpl.org/transactions/03C818B49391BB09B8C245DA7DB8397473D4C97A7E39B6BB08590731D7BF072C/raw))
  - `delivered_amount` is a string
  - The XRP amount is represented as a string whole number with `6` decimals. e.g. the `delivered_amount: "10"` should be interpreted as `0.000010` XRP
  - The `source_token` MUST be the **zero address**
- `1` - Classic Issued Currencies a.k.a [Trust Line Tokens](https://xrpl.org/docs/concepts/tokens/fungible-tokens#trust-line-tokens) ([example](https://livenet.xrpl.org/transactions/10CBD4248D521E86FFF38F1FD79911E0D0030E5DE98AD59B269B96226E23DB02/raw))
  - `delivered_amount` is a [Token Amount](https://xrpl.org/docs/references/protocol/data-types/currency-formats#token-amounts) object with `currency`, `issuer`, and `value`
  - `value` is a **string number with decimals**
    > Quoted decimal representation of the amount of the token. This can include scientific notation, such as `1.23e11` meaning 123,000,000,000. Both `e` and `E` may be used. This can be negative when displaying balances, but negative values are disallowed in other contexts such as specifying how much to send.
    <!-- cspell:ignore rMxCKbEDwqr76QuheSUMdEGf4B9xJ8m5De -->
  - The `issuer` and `currency` form a unique fungible token, such as [RLUSD](https://livenet.xrpl.org/token/524C555344000000000000000000000000000000.rMxCKbEDwqr76QuheSUMdEGf4B9xJ8m5De) which is issuer Ripple `rMxCKbEDwqr76QuheSUMdEGf4B9xJ8m5De` and currency `RLUSD` or `524C555344000000000000000000000000000000` in hex - this will always be a 40-character hex string
  - The `source_token` MUST be `1` + the **last** 31 bytes of the hash of the [currency code](https://xrpl.org/docs/references/protocol/data-types/currency-formats#currency-codes)’s canonical 20-byte internal format and the issuer - `keccak256(normalizedCurrency + decodeAccountID(issuerBytes))`.
    ```go
    Standard codes:    [0x00][ASCII bytes][trailing zeros]
    Non-standard codes: [raw 160-bit value]
    ```
  - Note, per [the docs](https://xrpl.org/docs/references/protocol/data-types/currency-formats#standard-currency-codes), `XRP` (all-uppercase) is disallowed.
- `2` - [Multi-Purpose Tokens](https://xrpl.org/docs/concepts/tokens/fungible-tokens/multi-purpose-tokens) (MPT)
  - `delivered_amount` requires only the `mpt_issuance_id` and the `value`
  - `value` is a string whole number, similar to XRP but the decimals are defined on the metadata during [creation](https://xrpl.org/docs/references/protocol/ledger-data/ledger-entry-types/mptokenissuance#mptokenissuance-fields) as `AssetScale`
  - The `source_token` MUST be `2` + the 31-byte left padded `mpt_issuance_id`, which is 24 bytes.

Every field can be determined from the existing XRPL Payment transaction fields with the exception of the cross-chain recipient information and the appropriate truncated decimal amount. For this, the following [MemoData](https://xrpl.org/docs/references/protocol/transactions/common-fields#memos-field) must be provided (72 bytes, encoded as a hexadecimal string).

```go
[4]byte   prefix = 0x994E5454            // 0x99'N''T''T'
[32]byte  recipient_ntt_manager_address  // WH universal address of the destination manager
[32]byte  recipient_address              // the address of the recipient
uint16    recipient_chain                // the Wormhole Chain ID of the recipient
uint8     from_decimals                  // decimals on XRPL
uint8     to_decimals                    // decimals on the destination chain
```

The `to_decimals` must be passed in off-chain as this field is typically [trimmed](https://github.com/wormhole-foundation/native-token-transfers/blob/fbe42df37ba19d3c05db8bb77b56c47fc0467c0e/evm/src/libraries/TrimmedAmount.sol#L219-L231) by the smart contract implementation to be `decimals = min(min(8, fromDecimals), toDecimals)` and this is the only data unavailable on XRPL. With these restrictions enforced in the watcher, the only risk should be to an individual transfer which loses dust by transferring an amount that is not rounded to the above calculation. The `value` must then be scaled to the resulting `decimals`. The watcher must also enforce that this resulting value fits into a u64.

Having the sender specify the `from_decimals` ensures that the amount is generated deterministically and as expected by the sender. While the `AssetScale` is immutable in the current XRPL specification, this can be considered as a defense-in-depth measure.

## Protocol Integration

Other than the watcher implementation described above, this design requires no other changes to Wormhole software or protocols.

## **API / database schema**

TODO

# **Caveats**

## Batches

This design was written without the assumption of the [Batch amendment](https://xrpl.org/resources/known-amendments#mainnet-status) which as of 2026-02-02 has 76.47% of the required 80% vote (held for 2 weeks). If the Batch amendment passes, this would greatly simplify the watcher implementation

## Decimals

XRP has a hardcoded 6 decimals. MPT tokens have an `AssetScale` determined during creation. However, [Trust Line Tokens](https://xrpl.org/docs/concepts/tokens/fungible-tokens#trust-line-tokens) have no specified number of decimals. They [use decimal math](https://xrpl.org/docs/concepts/tokens/fungible-tokens/trust-line-tokens#properties) with 15 digits of precision and an exponent.

The NTT spec encodes the decimals used to represent a transfer on each individual transfer payload. Therefore, we can allow the sender to specify the decimal representation in the transfer, up to 8 decimals, and expect it to be handled correctly on the receiving side.

This design does rely on the fact that MPT `AssetScale` is immutable in order to adhere to the determinism of message generation, since the amount on the transfer will be scaled based on `min(min(8, fromDecimals), toDecimals)`.

## Partial Payments

XRPL supports [Partial Payments](https://xrpl.org/docs/concepts/payment-types/partial-payments) (aka `DeliverMax`). When processing any Payment, use the `delivered_amount` metadata field, not the `Amount` field. The `delivered_amount` is the amount a payment actually delivered.

# **Alternatives Considered**

## Watcher Registration Lookups

The reliance on the XRPL watcher having to look up registrations on Solana was explicitly decided against so that the message emission on XRPL can be deterministic based solely on the transaction information.

## Using the Manager Address as Emitter

The calculation `keccak256(source_ntt_manager_address + source_token)` is used as the emitter address to prevent any possible confusion between different tokens being sent to the same manager. While, as of this writing, accounts [must opt in](https://xrpl.org/docs/concepts/tokens/fungible-tokens/multi-purpose-tokens#core-properties-of-mpts-on-xrpl) to receiving tokens, this is not true of XRP and we don’t know what future changes may be made to the protocol. If the `source_ntt_manager_address` were used directly, the only differentiation the guardian watcher would make (without performing any initialization message lookups) would be the `source_token` field which is [**unchecked**](https://github.com/wormhole-foundation/native-token-transfers/blob/fbe42df37ba19d3c05db8bb77b56c47fc0467c0e/evm/src/NttManager/NttManager.sol#L237) by other implementations. The approach in this design maintains safe compatibility with existing NTT deployments.

# **Security Considerations**

## Permissionless Memo Field

All fields passed via the memo MUST have no security impact and be properly validated or mitigated by the relevant implementations. In other words, these fields must be completed untrusted since they can be arbitrarily generated.

## Decimals

This design relies on other NTT implementations handling the decimals on the payload appropriately.

# Test Plan

TODO

# Performance Impact

TODO

# Rollout

TODO

## Acceptance Criteria

TODO

## Rollback

TODO
