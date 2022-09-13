# Governor

[TOC]

## Objective

Provide an optional security layer that enables Guardians to limit the amount of notional value that can be transferred out of a given chain within a sliding time period, with the aim of protecting against external risk such as smart contract exploits or runtime vulnerabilities.

## Background

Bridge security is incredibly high stakes — beyond core trust assumptions and high code quality, it is important to have defense in depth to minimize the potential for user harm. Under the assumption of smart contract bugs, the Governor is designed to be a passive security check that individual Guardians can implement to rate limit the notional value of assets that can be transferred out of a given chain to ensure the integrity of the value stored within a token bridge.

## Goals

- Implement an optional security check for Guardians to incorporate in a message verification process based on notional value processed by chain
- Limit the notional movement of value out of a given chain over a period of time

## Non-Goals

- Set a blanket rate limiting on all supported chains for all tokens
- Prevent any single "bad actor" from blocking other value transfer by intentionally exceeding the transfer limit for the given time period

## Overview

Each individual Guardian within the Guardian network should employ a set of strategies to verify the validity of a VAA. The Governor is designed to check VAAs that transfer tokens by enforcing limits on the notional value that can be transferred from a given chain over a specific period of time.

The current implementation works on two classes of transaction (large and small) and current configuration can be found [here](https://github.com/wormhole-foundation/wormhole/blob/dev.v2/node/pkg/governor/mainnet_chains.go):

- **Large Transactions**
  - A transaction is large if it is greater than or equal to the `bigTransactionSize` for a given origin chain.
  - All large transactions will have a mandatory 24-hour finality delay and will have no affect on the `dailyLimit`.
- **Small Transactions**
  - A transaction is small if it is less than the `bigTransactionSize` for a given origin chain.
  - All small transactions will have no additional finality delay up to the `dailyLimit` defined within a 24hr sliding window.
  - If a small transaction exceeds the `dailyLimit`it will be delayed until it either
    - fits inside the `dailyLimit` and will be counted toward the `dailyLimit`
    - has been delayed for 24-hours and will have no affect on the `dailyLimit`.

## Detailed Design

The Governor is implemented as an additional package that defines (1) a `ChainGovernor` object, (2) `mainnet_tokens.go`, a single map of tokens that will be monitored, and (3) `mainnet_chains.go`, a map of chains governed by the chain governor.

The `mainnet_tokens.go` maps a list of tokens with the maximum price between a hard-coded token floor price and the latest price read from CoinGecko.

If a node level config parameter is enabled to indicate that the chain governor is enabled, all VAAs will be passed through the `ChainGovernor` to perform a series of additional checks to indicate whether the message can be published or if it should not and be dropped by the processor.

The checks performed include:

1. Is the source chain of the message one that is listed within `mainnet_chains.go`?
2. Is the message sent from a goverened emitter?
3. Is the message a known type that transfers value?
4. Is the token transferred listed within `mainnet_tokens.go`?
5. Is the transaction a “large” transaction (ie. greater than or equal to `bigTransactionSize` for this chain)?
6. Is the transaction a “small” transaction (ie. less than `bigTransactionSize` for this chain)?

The above checks will produce 3 possible scenarios:

- **Non-Governed Message**: If a message does not pass checks (1-4), `ChainGovernor` will indicate that the message can be published.
- **Governed Message (Large)**: If a message is “large”, `ChainGovernor` will wait for 24hrs before signing the VAA and place the message in a queue.
- **Governed Message (Small)**: If a message is “small”, `ChainGovernor` will determine if it fits inside the `dailyLimit` for this chain. If it does fit, it will be signed immediately. If it does not fit, it will wait in the queue until it does fit. If it does not fit in 24hrs, it will be released from the queue.

While messages are enqueued, any Guardian has a window of opportunity to determine if a message is fraudulent using their own processes for fraud detection. If Guardians determine a message is fraudulent, they can delete the message from the queue from their own independently managed queue. If a super minority of Guardians (7 of 19) delete a message from their queues, this fraudulent message is effectively censored as it can no longer reach a super-majority quorum.

In this design, there are three mechanisms for enqueued messages to be published:

- A quorum (13/19) of Guardians can manually override the Governor and release any pending messages.
  - _Messages released through this mechanism WOULD NOT be added to the list of the processed transactions to avoid impacting the daily notional limit as maintained by the sliding window._
- Guardians will periodically check if a message can be posted without exceeding the daily notional limit as the sliding window and notional value of the transactions change.
  - _Messages released through this mechanism WOULD be added to the list of processed transactions and thus be counted toward the daily notional limit._
- Messages will be automatically released after a maximum time limit (this time limit can be adjusted through governance and is currently set to 24 hours).
  - _Messages released through this mechanism WOULD NOT be added to the list of the processed transactions to avoid impacting the daily notional limit as maintained by the sliding window._

## Potential Improvements

Right now, adding more governed emitters requires modifying guardian code. In the future, it would be ideal to be able to dynamically add new contracts for guardian nodes to observe.
