# Cross-Chain Queries (Proposal)

## Objective

Provide a mechanism for integrators to request information and an attestation from the guardians about a chain they are connected to.

## Background

Wormhole currently only supports "push" attestations. For example, in order to get the state of some contract on Ethereum, a smart contract would need to be written and deployed to Ethereum explicitly to read that state and call `publishMessage` on the Core Bridge. Furthermore, any time that data was needed elsewhere, a costly and time-consuming transaction would need to made.

This design proposes a mechanism to "pull" for attestations. For integrator flexibility and speed, along with decentralization support, these should be able to originate on _or_ off chain.

## Goals

- Establish a mechanism for initiating requests on-chain and off-chain
- Provide a solution for responding to requests
- Provide a generic, extensible solution for describing requests and responses
- Propose a form of request replay-protection
- Propose a form of DoS mitigation
- Propose a format for serialization

## Non-Goals

- Data-availability of query responses
- Describe all possible implementation-specific query requests or response formats
- Attest to the finality status of the result
- Resolve a tag (`latest`, `safe`, `finalized`, etc.) to a particular block hash or number pre-query
- Batch query requests
- Relaying of query responses

## Overview

Wormhole guardians run full nodes for many of the connected chains. In the current design, any information desired to be consumed from one of these chains on another chain must be "pushed" by a specially-developed contract on the chain where the data resides. This results in a lag time of landing a transaction as well the cost to execute that transaction on chain. For applications which may only require attestation of state changes cross chain on-demand, the additional complexity and cost to always publish messages is inefficient.

Consider how a token attestation from Ethereum for the [Token Bridge](./0003_token_bridge.md) could be different with cross-chain queries. Instead of having to make an Ethereum transaction to the token bridge to call `decimals()`, `symbol()`, and `name()` for a given `tokenAddress` and wait for finality on that transaction, one could make a cross-chain query for the three calls to that contract on Ethereum via the guardians. The cross-chain query could be significantly faster (likely seconds instead of 15-20 minutes) and avoid the need to pay gas on Ethereum.

## Detailed Design

### Requests

The request format should be extensible in order to support querying data across heterogenous chains and even future batching of those requests together.

```
QueryRequest
  chain_id  // Wormhole chainId
  nonce     // for repeat requests from off-chain
  request   // type and body of request
```

Where `request` is one of the supported types of cross-chain queries. Initially, this might be `eth_call`.

```
EthCallQueryRequest
  to     // contract to call
  data   // ABI packed call data
  block  // block tag, number, or hash
```

#### On-Chain

Requests can be made on-chain from supported chains via a new cross-chain query contract. This contract could construct a payload representing the requestor and request, and publish it via the core bridge, generating a standard VAA. Guardians could have a pre-defined list of these emitters to treat as cross-chain query requests and process the requests accordingly.

#### Off-Chain

Requests can be made off-chain by sending a new type of gossip message. This message should be signed with a private key separate from the p2p key so that requests can be relayed to the gossip network by a third-party service.

In order to differentiate signatures and prevent replay attacks of requests intended for devnet/testnet from mainnet ones, the following prefixes should be used when signing.

```
mainnet_query_request_000000000000|
testnet_query_request_000000000000|
devnet_query_request_0000000000000|
```

### Guardian

The guardian nodes should independently process requests by

- Detecting and channeling requests initiated via the query contract (on-chain) or gossip network (off-chain) to the new query module.
- Validating the request and verifying that the sender is authorized
  - Initially, this could be an allow-list of public keys and emitters
  - Later, there should be a permission-less mechanism to "sign up" for query access
- De-duplicating the requests
  - A malicious peer should not be able to replay off-chain requests
- Performing the corresponding query
- Serializing, signing, and gossiping the response

The response should be signed with the prefix `query_response_0000000000000000000|`

In order to reduce the storage burden on the guardian node, full responses should not need to persist in the guardian. However, to facilitate de-duplication and authorization, some cross-chain query information may be committed to Wormchain.

### Response

In order for the query results to be usable, they should be paired with the corresponding request. Therefore a full response might look like.

```
  // Sender information
  SenderChainId  uint16    // 0 = off-chain
  Signature      [65]byte  // or vaaHash for on-chain requests [32]bytes

  // Request
  RequestType    uint8     // 1 = eth_call
  RequestChainId uint16
  RequestNonce   uint32
  To             [20]byte  // contract address
  DataLen        uint32
  Data           bytes     // call data
  BlockLen       uint32
  Block          bytes     // block tag, number, or hash

  // Response
  BlockNumber    uint64
  BlockHash      [32]byte
  BlockTime      uint32
  ResultLen      uint32
  Result         bytes
```

### On-Chain Verification

Updated core bridge smart contract code should be provided to verify a quorum of response signatures for a given hash. This can provide a cross-call-minimized, gas-optimized solution for integrators.

## Example Data Flow

### Using a protocol-provided endpoint / service (off-chain)

In this example, imagine a hub-and-spoke cross-chain borrow / lend protocol based on Avalanche which requires state from Avalanche on another chain to fulfill a request.

> Prerequisite: Provision an Eth key, and authorize the public key with Wormhole

1. End-user wants to make a borrow on Moonbeam, clicks button on website, hits protocol REST endpoint.
2. Endpoint signs request for latest Avalanche block and `eth_call` to their deposit contract `getLockedAssets(addr, token)` and gossips it.
3. Guardians receive the request, make the call to Avalanche, sign and gossip the response.
4. Endpoint gathers enough signatures for quorum, returns status `200` with the response bytes and signatures.
5. User is prompted to submit the resulting response and signatures to the borrow contract on Moonbeam.
6. The contract verifies the signatures with the core bridge contract, checks the request data was for the correct chain, contract, and call data, then abi decodes the result.

## Future Considerations

### DoS Mitigation

To prevent requests from creating undue load on guardians' RPC nodes, a mechanism may be imposed to rate-limit or impose service fees upon requestors.

### Batch Requests

To reduce verification costs for integrations which may require multiple responses within one transaction (like the token bridge example above), the guardian code could be modified to accept multiple query types within one request and subsequently batch the corresponding results into one response.
