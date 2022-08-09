# Governor
[TOC]

## Objective
Provide an optional security layer that enables Guardians to limit the amount of notional value that can be transferred out of a given chain within a sliding time period.

## Background
Bridge security is incredibly high stakes — beyond core trust assumptions and high code quality, it is important to have defense in depth to minimize the potential for user harm. Under the assumption of smart contract bugs, the Governor is designed to be a passive security check that individual Guardians can implement to rate limit the notional value of assets that can be transferred out of a given chain to ensure the integrity of the value stored within a token bridge.

## Goals
* Implement an optional security check for Guardians to incorporate in a message verification process based on notional value processed by chain
* Limit the notional movement of value out of a given chain over a period of time

## Non-Goals
* Set a blanket rate limiting on all supported chains for all tokens
* Prevent any single "bad actor" from blocking other value transfer by generating one large transfer

## Overview
Each individual Guardian within the Guardian network can employ a set of strategies to verify the validity of a VAA. The Governor is designed to be one of those checks by proposing a notional limit on the value that can be transferred from a given chain within a certain time frame. 

There are many other potential variations on the notional value limit and time frame considered (i.e. 4 hour window, 12 hour window, max single transaction size) — this initial implementation is for a 24-hour window with a custom limit per chain that is informed by data-driven analysis from recent chain activity.

## Detailed Design
The Governor is implemented as an additional package that defines (1) a `ChainGovernor` object, (2) `mainnet_tokens.go`, a single map of tokens that will be monitored, and (3) `mainnet_chains.go`, a map of chains governed by the chain governor.

The `mainnet_tokens.go` maps a list of tokens with the maximum price between a hard-coded token floor price and the latest price read from CoinGecko.

If a node level config parameter is enabled to indicate that the chain governor is enabled, all VAAs will be passed through the `ChainGovernor` to perform a series of additional checks to indicate whether the message can be published or if it should not and be dropped by the processor.

The checks performed include: 

1. Is the source chain of the message one that is listed within `mainnet_chains.go`?
2. Is the message sent from a goverened emitter?
3. Is the message a known type that transfers value?
4. Is the token transferred listed within `mainnet_tokens.go`?
5. Will the transfer amount bring the total notional value transferred within a specified time frame over the limit?

If a message does not apply to or passes these checks, `ChainGovernor` will indicate that the message can be published.

If a message fails these checks, it will be added to a pending list and `ChainGovernor` will indicate that the message should not be published.

Messages in this pending list will periodically be checked again to see if they can be posted without exceeding the limit. Guardians can also manually override the Governor and release any pending VAA.

## Potential Improvements
Right now, adding more governed emitters requires modifying guardian code. In the future, it would be ideal to be able to dynamically add new contracts for guardian nodes to observe.
