# EVM watcher tests

This package has two complementary groups of tests: unit and integration tests
covering the watcher's behavior, and hash regression tests that pin the exact
digests the watcher produces for checked-in transaction receipts.

## Unit and integration tests

| File | What it covers |
| ---- | -------------- |
| `watcher_test.go` | Message processing: `postMessage` dispatch, immediate publication, pending-message queueing, `processNewBlock` finality handling, `verifyAndPublish`, consistency-level handling, and block-time retry logic. |
| `reobserve_test.go` | Reobservation request handling: invalid chain IDs, receipt errors, failed transaction status, skipped logs, deterministic ordering, large receipts, and transfer-verifier integration. |
| `by_transaction_test.go` | Core bridge log validation (`isValidCoreBridgeMessagePublicationLog`): wrong contract, wrong topic, removed logs, malformed topics. |
| `custom_consistency_level_test.go` | Custom consistency level (CCL) config parsing and effective-consistency-level handling. |
| `chain_config_test.go` | Per-chain configuration: finality support, EVM chain IDs, mainnet contract addresses. |
| `blocks_by_timestamp_test.go` | The block-by-timestamp cache used for CCQ. |
| `ccq_test.go`, `ccq_backfill_test.go` | Cross-chain query (CCQ) request handling and backfill. |
| `tron_integration_test.go` | Live integration against the Tron Nile testnet (requires network access). |
| `watcher_test_helpers_test.go` | Shared mock connector and test helpers. |

## Hash regression tests

These fixtures protect the message hash used to match live observations with
re-observations. The expected hashes are checked in and are constructed without
calling `MessagePublication.CreateDigest()`, so an accidental change to watcher
message construction is caught by the tests.

### Source of truth

`TestObservationReobservationParity` in
`observation_reobservation_parity_test.go` loads the canonical receipt vectors
and sends each receipt through both watcher paths:

- live observation: `runMessageProcessor` / `postMessage`
- re-observation: `runReobservationHandler` /
  `handleReobservationRequest`

For every receipt, the test requires both paths to emit exactly the same
digests - compared without regard to order, but with every duplicate counted -
and requires those digests to match the checked-in `expectedMessages[].hash`
values. It also asserts that `CreateDigest()` and `VAAHash()` remain
equivalent.

The digest is the double Keccak-256 hash of the serialized VAA body fields:

1. timestamp as a big-endian `uint32`
2. nonce as a big-endian `uint32`
3. Wormhole chain ID as a big-endian `uint16`
4. 32-byte emitter address
5. sequence as a big-endian `uint64`
6. consistency level as a `uint8`
7. payload bytes

Receipt metadata such as transaction hash, block hash, block number,
transaction index, log index, gas fields, bloom, and `IsReobservation` must not
change the digest. `TestGeneratedReceiptGoldenVectorMetadataIndependence`
checks this explicitly.

### Receipt fixtures

The tests load only these canonical files:

- `testdata/generated_receipts.json`: 210 deterministic synthetic receipt
  vectors containing 211 expected messages.
- `testdata/real_receipts.json`: 200 Ethereum mainnet receipt vectors containing
  202 expected messages. Two receipts contain two Wormhole Core events.

Each vector has this shape:

```json
{
  "name": "real-000-tx-2461b4bce349",
  "comment": "real: Ethereum Wormhole Core ...",
  "wormholeChainId": 2,
  "blockTime": 1749806543,
  "receipt": {},
  "expectedMessages": [
    {
      "logIndex": 645,
      "hash": "cfdc8c25256503257c9d66a22b43e00a1ce42445592e5e8653c684e219cae1b5"
    }
  ]
}
```

`hash` is the observation-matching digest, encoded as 32 lowercase hexadecimal
bytes without a `0x` prefix. `logIndex` makes message selection explicit for
receipts containing unrelated logs or multiple Wormhole events.

### Data provenance and hash construction

The synthetic vectors were built from explicit event fields, ABI-encoded into
`LogMessagePublished` receipt logs, and paired with independently calculated
hashes. The real vectors were derived from full Ethereum receipts. Historical
block timestamps were matched to the corresponding Wormholescan VAAs by
transaction hash, emitter, and sequence because timestamp is part of the signed
body.

For both data sets, each expected hash was calculated by serializing the VAA
body fields in protocol order and applying Keccak-256 twice. This calculation
did not call `MessagePublication.CreateDigest()`, `VAAHash()`, or another watcher
hash helper. All 200 previously recorded historical hashes matched this
independent calculation. The full receipts also revealed two additional valid
Wormhole Core events, which is why 200 real receipts contain 202 expected
messages.

The fixtures are therefore checked from two independent directions: the stored
hashes come from direct protocol serialization, while the tests decode the
receipt logs through the production ABI and construct `MessagePublication`
objects through the watcher paths. A mistake in watcher field mapping,
serialization, or event selection causes the produced digest to differ from the
checked-in value.

The generated corpus covers:

- empty, binary, all-zero, 1-, 31-, 32-, 33-, and 4096-byte payloads
- leading-zero emitter addresses
- timestamps with non-round and high-bit values
- chain ID `4004`
- sequence boundaries including `0`, `uint64` max, and `2^53 + 1`
- maximum `uint32` nonce
- immediate, safe, finalized, custom, and historical consistency levels
- unrelated logs, wrong-contract logs, and multiple Wormhole events in one
  receipt

`TestGeneratedReceiptGoldenVectorsCoverage` pins these properties so fixture
changes cannot silently weaken the corpus.

### Tests to run

From `node`:

```sh
go test ./pkg/watchers/evm -run TestObservationReobservationParity -count=1
go test ./pkg/watchers/evm -run 'TestGeneratedReceiptGolden|TestConsistencyLevelMatches' -count=1
go test ./pkg/watchers/evm -count=1
```
