# Governor

[TOC]

## Objective

Limit the impact of certain exploits by giving Guardians the option to delay Wormhole messages from registered token bridges if their aggregate notional value is extraordinarily large.

## Background

A single integrity failure of the core messaging bridge can have disastrous consequences for users of token bridges built ontop of Wormhole, if no additional safety mitigations are in place. For example, on Feb 2, 2022 [a vulnerability in the Solana core smart contract was exploited](https://wormholecrypto.medium.com/wormhole-incident-report-02-02-22-ad9b8f21eec6) to maliciously mint wETH on Solana and it was subsequently bridged back to Ethereum.

There are multiple potential failure modes of the bridge:
* In scope of the Wormhole security program:
  * Bugs in the smart contract
  * Bugs in the Guardian software
  * Guardian key compromise
* Out of scope of the Wormhole security program:
  * Bugs in the blockchain smart contract runtime or rpc nodes
  * Forks

Even if Wormhole's code and operations are flawless, it might still produce "invalid" messages if there is an exploit of the origin chain: "Bugs" in the origin-chain smart contract runtime that produced undesirable Wormhole messages could be patched by the community, effectively leading to a fork that reverts these Wormhole messages. And bugs in the rpc nodes could lead to the Guardians not having an accurate view of the on-chain state.

If token bridge transfers are unlimited and instantaneous, a bug in a single connected chain could cause the entire token bridge to be drained.

## Goals

If a Guardian decides to enable this feature:
* Delay Wormhole messages from registered token bridges for 24h, if their notional value is excessively large.
* This gives the Guardian the opportunity to delete pending messages, if they were created through a software bug and not accurately represent the state of the origin chain.
* Protect against sybil attacks, i.e. this feature should work even if an attacker tries to make one large transfer look like organic activity by splitting it into many small transfers.

## Non-Goals

* Synchronize state between Guardians. Each Guardian may have a slightly different view of the network, including Governor configuration, ordering and completeness of token bridge transfers, etc. The Governor is a local feature that makes decisions based on each Guardian's individual view of the network.
* Prevent quality-of-service degradation attacks where one bad actor causes a majority of transfers to be delayed.

## High-Level Design

### Delay Decision Logic
*Configuration*:
* For each chain, a _single-transaction-threshold_ and a *24h-threshold* denominated in a base currency (U.S. Dollar) is specified in [mainnet_chains.go](https://github.com/wormhole-foundation/wormhole/blob/main/node/pkg/governor/mainnet_chains.go).
* A list of prominent tokens is specified in [manual_tokens.go](https://github.com/wormhole-foundation/wormhole/blob/main/node/pkg/governor/manual_tokens.go) and [generated_mainnet_tokens.go](https://github.com/wormhole-foundation/wormhole/blob/main/node/pkg/governor/generated_mainnet_tokens.go). Tokens that are not on this list are not being tracked by the Governor. This list is opt-in in order to prevent thinly-traded tokens with unreliable price feeds to count towards the thresholds.

Governor divides token-based transactions into two categories: small transactions, and large transactions.

- **Small Transactions:** Transactions smaller than the single-transaction threshold of the chain where the transfer is originating from are considered small transactions.  During any 24h sliding window, the Guardian will sign token bridge transfers in aggregate value up to the 24h threshold with no finality delay.  When small transactions exceed this limit, they will be delayed until sufficient headroom is present in the 24h sliding window. A transaction either fits or is delayed, they are not artifically split into multiple transactions. If a small transaction has been delayed for more than 24h, it will be released immediately and it will not count towards the 24h threshold.
- **Large Transactions:** Transactions larger than the single-transaction threshold of the chain where the transfer is originating from are considered large transactions.  All large transactions have an imposed 24h finality delay before Wormhole Guardians sign them. These transactions do not affect the 24h threshold counter.

### Asset pricing

Since the thresholds are denominated in the base currency, the Governor must know the notional value of transfers in this base currency. To determine the price of a token it uses the *maximum* of:
1. **Hardcoded Floor Price**: This price is hard coded into the governor and is based on a fixed point in time (usually during a Wormhole Guardian release) which polls CoinGecko for a known set of known tokens that are governed.
2. **Dynamic Price:** This price is dynamically polled from CoinGecko at 5-10min intervals.

The token configurations are in [manual_tokens.go](https://github.com/wormhole-foundation/wormhole/blob/main/node/pkg/governor/manual_tokens.go) and [generated_mainnet_tokens.go](https://github.com/wormhole-foundation/wormhole/blob/main/node/pkg/governor/generated_mainnet_tokens.go).

If CoinGecko was to provide an erroneously low price for a token, the Governor errs on the side of safety by using the hardcoded floor price instead.

### Visibility
Each Guardian publishes its Governor configuration and status on the Wormhole gossip network, which anyone can subscribe to [instructions](https://github.com/wormhole-foundation/wormhole/blob/dev.v2/relayer/spy_relayer/README.md). Some Guardians also make the Governor status available through a public API, which can be visualized on the [Wormhole Dashboard](https://wormhole-foundation.github.io/wormhole-dashboard/). A more feature-rich [Wormhole Explorer](https://github.com/wormhole-foundation/wormhole-explorer) that will aggregate Governor status across all Guardians is work-in-progress.

### Security Considerations
* The Governor can only reduce the impact of an exploit, but not prevent it.
* Excessively high transfer activity, even if manufactured and not organic, will cause transactions to be delayed by up to 24h.
* If CoinGecko reports an unreasonably high price for a token, the 24h threshold will be exhausted sooner.
* Guardians need to manually respond to erroneous messages within the 24h time window. It is expected that all Guardians operate collateralization monitoring for the protocol, taking into account the Governor queue. All Guardians should have alerting and incident response procedures in case of an undercollateralization.
* An attacker could utilize liquidity pools and other bridges to launder illicitly minted wrapped assets.

## User FAQ
- **Who is the Governor?**
    - The Governor is not a person, but a software feature built into the guardian reference implementation.
- **When does the Governor step in?**
    - The Governor delays messages for up to 24 hours if the aggregate value of messages originating from one chain in the past 24 hours exceeds a threshold that is set per chain, which happens very rarely.
- **In what direction are Governor thresholds for delays imposed?**
    - Transfer volume is calculated *from* each chain.
- **Are Governor thresholds per wallet?**
    - Governor thresholds are imposed globally from each chain, to address [Sybil-attacks](https://en.wikipedia.org/wiki/Sybil_attack), in which an attacker could split a large transfer into many small transfers.
- **How can the Governor help protect Wormhole users?**
    - The Governor can help reduce the impact of a hack or fork by giving the Guardians additional time to respond to any extraordinary events. E.g., if a chain forks, while the Governor and the new fork delay a Wormhole message, do not include that Wormhole message anymore, the Guardians have time to decide which fork to adopt and whether or not to confirm the pending messages, to ensure that Wormhole messages represent the state of the origin chain as accurately as possible.
- **How does the Governor *exactly* decide which messages to confirm immediately and which to delay?**
    - The Governor has two configurable values for each chain: The 24h threshold and the single-transaction threshold. These two thresholds only apply to messages originating **from** a given chain. Messages are processed differently if their value exceeds the single-transaction threshold or not. We call these small transfers and large transfers. Large transfers will always be delayed for 24 hours. Small transfers will be confirmed immediately unless the sum of all small transfers in the last 24 hours, plus the transfer in question, exceeds the 24h-threshold (this is to protect against an attacker splitting a large transfer into many small transfers). Those small transfers will be delayed as long as the 24h threshold is exceeded, but never for longer than 24 hours. If a small transfer would still cause the 24h threshold to be exceeded after 24 hours (e.g., because there was a constant flow of smaller transfers), it will be confirmed regardless and not count towards the 24h-threshold.
- **Is it possible to have my transfer delayed by Governor?**
    - The above answer explains that transactions can be delayed during high utilization times.  However, the maximum possible wait time is 24hrs, though most transactions have no delay.
- **How do I know if my transfer will be delayed by the Governor?**
    - If you are using the Portal Bridge UI, you will get a warning pop-up if your transfer is likely to be delayed. Alternatively, you can receive each Guardian’s Governor config and status  on the Wormhole gossip network ([instructions](https://github.com/wormhole-foundation/wormhole/blob/dev.v2/relayer/spy_relayer/README.md)), which contains a list of tracked tokens, their prices, configured thresholds and available headroom. Based on this data, you can calculate if a transfer is likely to go through or not, assuming there is no other transfer in the meantime. Some Guardians also make the Governor status available through a public API, which can be visualized on the [Wormhole Dashboard](https://wormhole-foundation.github.io/wormhole-dashboard/). The [Wormhole Explorer](https://github.com/wormhole-foundation/wormhole-explorer) is work-in-progress and will provide an easy API to get the aggregated Governor status across all Guardians.
- **How does Governor determine the notional value of a transfer?**
    - Two sources determine the notional value of the transfer.  One source is the hard coded floor price shipped with the Guardian software (which is polled from Coingecko at release time) and the other is a dynamic polling of Coingecko on 5-10min intervals.  The notional value of the transfer will be the higher of the two sources.  This is designed in such a way to ensure that if Coingecko reports a nil, zero, or meager price for an asset, it won’t allow an attacker to move effectively infinite amounts of a governed token.
- **Why doesn’t the Governor use a price oracle like Pyth or Chainlink for notional value computation?**
    - This is mainly because neither Chainlink or Pyth currently support price feeds for all the registered tokens the Wormhole Guardians wish to Governor.  As price oracles mature to support a broader set of tokens, this may become more viable to use on-chain price oracles for this computation.

## Operator FAQ
- **I have identified a potentially inaccurate message in the Governor queue. Can I extend the release time to give me more time to investigate?**
    - Yes. You can use the `ChainGovernorResetReleaseTimer` admin RPC or `governor-reset-release-timer [VAA_ID]` admin command to reset the delay to 24h. Another option would be to drop the message first with the `ChainGovernorDropPendingVAA` admin RPC or `governor-drop-pending-vaa [VAA_ID]` admin command and after the investigation is concluded and it is determined that the message was (and still is) an accurate representation of the source chain state, you can issue a re-observation request.

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
