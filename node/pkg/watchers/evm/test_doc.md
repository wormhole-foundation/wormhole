# EVM watcher postMessage fixture tests

This test data is used by `TestPostMessageGeneratedFixture` in `watcher_test.go`.
The goal is regression coverage for the EVM watcher message path: given a historical
or generated `LogMessagePublished` event, the watcher should either not publish a
message, or publish the same message hash every time.

## Files

- `testdata/generated_data.json`: artificial cases generated from the prompt rules.
- `testdata/generated_receipts.json`: artificial Ethereum transaction receipts built
  from `generated_data.json`.
- `testdata/real_data.json`: real Ethereum Wormhole Core logs scraped from chain data.
- `testdata/real_receipts.json`: full Ethereum transaction receipts fetched for the
  transactions in `real_data.json`.

## Fixture fields

Each case contains the decoded event fields (`Sender`, `Sequence`, `Nonce`,
`Payload`, `ConsistencyLevel`), the original raw EVM log under `Raw`, and test
expectations:

- `BlockTime`: Unix timestamp used as the message timestamp.
- `messageSent`: whether the watcher should publish to `msgC`.
- `hash`: expected `MessagePublication.CreateDigest()` output when `messageSent`
  is true.

If `messageSent` or `hash` is missing, the test computes it and writes it back to
the fixture. If it already exists, the test compares the current output with the
fixture and does not rewrite it.

## Generated data

The artificial data was created to cover common valid and invalid watcher inputs
without making every failure the same:

- Most invalid cases have only one or two issues, never more than three.
- The event topic is correct in about 90 percent of cases.
- The raw contract address is correct in about 90 percent of cases.
- Non-empty topic lists contain only the event topic and indexed sender, so the
  logs can be ABI-parsed during re-observation.
- A small number of cases use `removed: true`.
- `BlockTime` values are deterministic pseudo-random timestamps.

The generated receipt file is derived mechanically from the generated message
cases. Each receipt is a successful geth `types.Receipt` JSON object with one log,
and its transaction hash, block hash, block number, transaction index, and log
index match the corresponding `Raw` log.

After changing `BlockTime` or any event field, remove `hash` from affected cases
and run the fixture test to regenerate expected hashes.

## Real data

The real data came from Ethereum Wormhole Core `LogMessagePublished` logs. For
these cases, `BlockTime` was filled from Wormholescan VAA timestamps for the
matching transaction hash, emitter address, and sequence. This matters because
the VAA signing digest includes timestamp; using the old test timestamp `1234`
produced hashes that did not match historical VAA digests.

Historical non-immediate consistency levels such as `1` are treated as finalized
unless they are explicitly latest (`200`) or safe (`201`).

The real receipt file is keyed by transaction hash. Each fixture log in
`real_data.json` must be present in the matching receipt from `real_receipts.json`.

## Test flow

For each fixture case, the test:

1. Builds an `AbiLogMessagePublished` from the fixture.
2. Installs a successful mocked receipt for the transaction.
3. Calls `postMessage` with the case `BlockTime`.
4. For non-immediate finality, calls `processNewBlock` with the log block at the
   needed finality level.
5. Checks whether a message was sent to `msgC`.
6. Compares or seeds `messageSent` and `hash`.

The generated fixture also checks the expected input distribution so accidental
changes do not turn the data set into mostly one failure mode.

`TestFixtureObservationMatchesReobservation` uses the receipt fixtures to run both
paths for the same transaction: the normal observation flow from the fixture log,
and the re-observation flow from transaction hash plus receipt. The test requires
both paths to agree on whether a message is sent; when one is sent, it finds the
matching message ID and requires the same `CreateDigest()` hash.
