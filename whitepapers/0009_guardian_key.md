# Guardian Key Usage

## Objective

- Describe how guardian keys are used and how message confusion is avoided.

## Background

Message confusion could occur when a Guardian signs a message and an attacker replays that message elsewhere where it is interpreted as a different message type, which could lead to unintended behavior.

## Overview

The Guardian Key is used to:

1. Sign gossip messages
   1. heartbeat
   1. governor config and governor status
   1. observation request
1. Sign Observations
   1. Version 1 VAAs
1. Sign Guardian identification
   1. Wormchain account registration
1. Sign Accountant observations
   1. Token Bridge
   1. NTT
1. Sign Query responses

## Detailed Design

Signing of gossip messages:

1. Prepend the message type prefix to the payload
2. Compute Keccak256Hash of the payload.
3. Compute ethcrypto.Sign()

Signing of Observations:

- v1 VAA: `double-Keccak256(observation)`.

Rationale

- Gossip messages cannot be confused with other gossip messages because the message type prefix is prepended.
- Gossip messages cannot be confused with observations because observations utilize a double-Keccak256 and the payload is enforced to be `>=34` bytes.

## Prefixes Used

<!-- cspell:disable -->

```go
acct_sub_obsfig_000000000000000000| // token bridge accountant observation
ntt_acct_sub_obsfig_00000000000000| // ntt accountant observation
governor_config_000000000000000000| // gossip governor config
governor_status_000000000000000000| // gossip governor status
heartbeat|                          // gossip heartbeat
signed_observation_request|         // gossip signed observation request
mainnet_query_request_000000000000| // query request (mainnet, not signed by guardian)
testnet_query_request_000000000000| // query request (testnet, not signed by guardian)
devnet_query_request_0000000000000| // query request (devnet, not signed by guardian)
query_response_0000000000000000000| // query response
query_response_0000000000000000000| // query response
signed_wormchain_address_00000000|  // wormchain register account as guardian
```

<!-- cspell:enable -->
