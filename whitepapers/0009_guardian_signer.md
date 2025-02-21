# Guardian Signer

## Objective

Sign different kinds of messages within the Wormhole ecosystem for the purpose of attestation and guardian identification.

## Background

In order for guardians to attest to on-chain events or prove their identities when communicating among each other, digital signatures are required. On-chain smart contracts and guardians hold registries of public keys of trusted guardian nodes that are permitted to perform certain actions within the guardian ecosystem. Without this system, it would not be possible to distinguish legitimate behavior from malicious.

The guardian signer is responsible for providing signatures, and supports different mechanisms for [producing signatures](../docs/guardian_signer.md).

## Overview

The guardian signer is used to sign numerous messages within the Wormhole ecosystem:

* Gossip Messages - Messages that are sent between guardians, such as heartbeats, governor configs, governor status updates and observation requests.
* On-Chain Observations - Events that occur on-chain that need to be attested to and delivered to different chains, bundled in VAAs (Version 1).
* Guardian Identification - Wormchain account registration.
* Accountant Observations - Sign observations relevant to token bridge and NTT. 
* Cross-Chain Query Responses - Attest to states on other chains.

## Detailed Design

The process for signing gossip messages are as follows:

1. Prepend the message type prefix to the payload.
    - This is to ensure uniqueness of the message, and prevent two gossip messages from being used interchangeably for different operations.
2. Compute the `keccak256` hash of the payload.
3. Compute a signature of the hash using `ethcrypto.Sign()`.

On-chain observations are signed by performing a double-`keccak256` hashing operation on the observation and signing the result. The resulting data structure, which primarily contains information about the observation and the signature, is called a VAA (Verifiable Action Approval). 

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
