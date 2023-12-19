# Batch VAAs (EVM)

[TOC]

## Objective

Add batch VAAs to Wormhole to allow for efficient verification of multiple messages, as well as allowing for better patterns for xDapps to compose with each other.

## Background

Currently, composing between different cross-chain applications is often leading to design patterns where developers are nesting all actions into a common VAA that contains all instructions to save gas and ensure atomic execution. This pattern is sub-optimal as it often requires multiple composing xDapps to integrate each other and extend their functionality for new use-cases and delivery methods. This is undesirable as it adds more code-complexity over time and slows down integration efforts.

## Goals

Extend Wormhole with the core-primitives needed to build better composability patterns by leveraging batching.

- Individual VAAs included in a batch should stay backwards compatible with existing smart contract integrations.
- Batch VAAs should be usable in a gas-efficient manner.
- Allow for cheaper verification of individual VAAs included in a batch.

## Non-Goals

This design document focuses only on the extension of the current implementation of Wormhole’s generic message passing ([0001_generic_message_passing.md](https://github.com/wormhole-foundation/wormhole/blob/main/whitepapers/0001_generic_message_passing.md)) and does not attempt to solve the following problems, leaving them for future design iterations:

- Replacing VAAv1s with batch VAAs (VAAv2) containing only a single observation.
- Ensuring backwards compatibility of batch VAAs (VAAv2) with only a single observation using the existing Wormhole APIs.
- Verifying batch VAAs with a subset of the original observations.
- The specifics of implementing xDapps leveraging batch VAAs, other than ensuring the right APIs are provided.

## Overview

For now, all Wormhole messages that are emitted during a transaction will continue to receive individual signatures (VAAv1). This is mainly to ensure backwards compatibility and might be deprecated in the future.

Guardians will start producing batch-signatures for messages emitted within the same transaction and that share the same nonce. However, messages with a nonce of zero will not receive batch-signatures, as this is a way of opting out of including messages in a batch VAA.

The number of messages within a batch is constrained by the maximum `uint8` value of 255, due to the [VAAv2 Payload Encoding format](#payloads-encoded-messages). If a transaction produces more than 255 messages with the same nonce a batch-signature will not be produced because it would not fit within the binary encoding understood by `parseAndVerifyBatchVM`, and therefore could not be successfully verified on-chain. Transactions may include messages to produce multiple batch-signatures, which would be independent of each other verified individually.

We will add support for two new VAA payload types to the Wormhole core contract to allow handling of these:

- The VAAv2 payload that holds the batch-signatures, an array of the signed hashes and an array of observations (the [Structs](#structs) section of this paper defines `observation`). This VAAv2 payload can be verified using a new Wormhole core contract endpoint `verifyBatchVM`. This payload will also be produced for individual messages with a nonce greater than zero to offer integrators flexibility when deciding which Wormhole core endpoint to verify messages with.
- The VAAv3 payload, which is a “headless” payload that only carries an observation. This payload can only be verified when its hash is cached by the Wormhole core contract during VAAv2 signature verification. This payload type is created when a VAAv2 is parsed using the Wormhole core endpoint `parseBatchVM` by prepending the version type to the observation bytes. Although the payload format for VAAv3 is new, it will be parsed and verified using the existing Wormhole core endpoints and parsed into the existing `VM` struct (`signatures[]` and `guardianSetIndex` will be null) to ensure backwards compatibility.

## Detailed Design

### VAAv2

To create a VAAv2 payload (which is eventually parsed into the `VM2` struct) an xDapp will invoke the `publishMessage` method at least one time with a nonce greater than zero. The guardian will then produce a VAAv2 payload by grouping messages with the same `nonce` (in the same transaction) and create a batch-signature by signing the payload version (`uint8(2)` for batches) and hash of all hashes of the observations:

`hash(version, hash(hash(Observation1), hash(Observation2), ...))`

Once the batch is signed by the guardian, the VAAv2 can be parsed and verified by calling the new Wormhole core endpoint `parseAndVerifyBatchVM`. This method parses the VAAv2 into the `VM2` struct by calling `parseBatchVM`, calls `verifyBatchVM` to verify the batch-signatures, and stores the hash of each observation in a cache when specified by the caller (for reasons explained in the [VAAv3](#vaav3) section of this detailed design). `verifyBatchVM` also independently computes the hash of each observation and validates that each hash is stored in the `hashes` array, which is included in the VAAv2 payload. The structure of the VAAv2 payload can be found in the [Payloads](#payloads-encoded-messages) section of this design.

### VAAv3

When a VAAv2 payload is parsed into a `VM2` struct, each observation is stored as bytes in the `observations` byte array. The `uint8(3)` version type is prepended to the bytes to specify that they are considered a VAAv3 payload. Each VAAv3 payload can be parsed into the existing `VM` struct with `parseVM` to ensure backwards compatibility with existing smart contract integrations. Since VAAv3 payloads are considered “headless” and do not contain signatures, the `Signatures[]` and `guardianSetIndex` fields are left as null in the `VM` struct.

A parsed VAAv3 payload can then be verified by calling the existing method `verifyVM`. This method will check that the hash of the VAAv3 payload (hash of the observation) is stored in the `verifiedHashCache` and bypass signature verification (allowing for cheap verification of individual messages). The VAAv3 payload hash will only be stored in the `verifiedHashCache` if the caller sets the `cache` argument to `true` when verifying the associated VAAv2 payload with `verifyBatchVM`.

At the end of a batch execution, the handler contract should call `clearBatchCache` which will clear the `verifiedHashCache` of provided hashes and reduce the gas costs associated with storing the hashes in the Wormhole contract’s state. A parsed VAAv3 payload will no longer be considered a verified message once its hash is removed from the `verifiedHashCache`.

### API

```solidity
function parseAndVerifyBatchVM(bytes calldata encodedVM2, bool cache)
function verifyBatchVM(Structs.VM2 memory vm2, bool cache)
function parseBatchVM(bytes memory encodedVM2)
function clearBatchCache(bytes32[] memory hashesToClear)
```

### Structs

```solidity
struct Header {
    uint32 guardianSetIndex;
    Signature[] signatures;
    bytes32 hash;
}

// This struct exists already, but now has an additional version type 3.
struct VM {
    uint8 version; // Version = 1 or 3
    // The following fields constitute an `observation`. For compatibility
    // reasons we keep the representation inlined.
    uint32 timestamp;
    uint32 nonce;
    uint16 emitterChainId;
    bytes32 emitterAddress;
    uint64 sequence;
    uint8 consistencyLevel;
    bytes payload;
    // End of observation

    // Inlined Header
    uint32 guardianSetIndex;
    Signature[] signatures;

    // Hash of the observation
    bytes32 hash;
}

struct VM2 {
    uint8 version; // Version = 2

    // Inlined header
    uint32 guardianSetIndex;
    Signature[] signatures;

    // Array of observation hashes
    bytes32[] hashes;

    // Computed Batch Hash - `hash(version, hash(hash(Observation1), hash(Observation2), ...))`
    bytes32 hash;

    // Array of observation bytes with prepended version 3
    bytes[] observations;
}
```

### Payloads (Encoded Messages)

VAAv2:

```solidity
// Version uint8 = 2;
uint8 version;
// Guardian set index
uint32 guardianSetIndex;
// Number of signatures
uint8 signersLen;
// Signatures: 66 bytes per signature
Signature[] signatures;
// Number of hashes
uint8 hashesLen;
// Array of observation hashes
bytes32[] hashes;
// Number of observations, should be equal to hashesLen
uint8 observationsLen;

// Repeated for observationLen times, bytes[] observation
	// Index of the observation
	uint8 index;
	// Number of bytes in the observation
	uint32 observationBytesLen;
	// Encoded observation, see the Structs section
        // for details on the observation structure.
	bytes observation;
```

VAAv3:

```solidity
// Version uint8 = 3;
uint8 version;
// Observation bytes
bytes observation;
```
