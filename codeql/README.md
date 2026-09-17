# Wormhole Go CodeQL Lints

This directory is a CodeQL query pack with Wormhole-specific Go lint rules. Run the commands below from this directory (`codeql/`). The pack lock file (`codeql-pack.lock.yml`) pins the Go query dependencies; run `codeql pack install` once to download them.

CI compiles the queries, runs the query unit tests, and analyzes the `node` and `sdk` Go modules with this pack (see `.github/workflows/codeql.yml`). Findings are uploaded to GitHub code scanning and appear as annotations on pull requests.

## Rules

- [`wormhole/go/already-locked-receiver-mutex`](docs/rules/already-locked-receiver-mutex.md): require documented `AlreadyLocked` helpers to be called while holding the exact receiver mutex.
- [`wormhole/go/algorand-publication-field-length-check`](docs/rules/algorand-publication-field-length-check.md): require exact length checks before decoding Algorand publication nonce and sequence fields.
- [`wormhole/go/canonical-chain-id-parsing`](docs/rules/canonical-chain-id-parsing.md): use Wormhole SDK chain-ID conversion helpers at modeled external boundaries.
- [`wormhole/go/canonical-vaa-address-parsing`](docs/rules/canonical-vaa-address-parsing.md): use Wormhole SDK address parsers for external address values.
- [`wormhole/go/canonical-vaa-id-parsing`](docs/rules/canonical-vaa-id-parsing.md): parse complete VAA IDs with the canonical parser instead of reconstructing components manually.
- `wormhole/go/delegate-consensus-canonical-digest`: key delegate observation quorum buckets by the reconstructed `MessagePublication` VAA signing digest, not by serialized observations or composite keys.
- `wormhole/go/delegated-guardian-config-validation`: strictly parse guardian addresses, reject duplicate canonical keys, and enforce non-empty threshold/quorum before governance serialization.
- [`wormhole/go/evm-finality-release-and-reorg-checks`](docs/rules/evm-finality-release-and-reorg-checks.md): require finality, receipt refetch, and reorg-provenance checks before releasing pending EVM observations.
- [`wormhole/go/evm-require-successful-receipt-before-observation`](docs/rules/evm-require-successful-receipt-before-observation.md): require a local successful-receipt proof before EVM log observation or publication.
- [`wormhole/go/evm-verify-and-publish-gate`](docs/rules/evm-verify-and-publish-gate.md): route EVM watcher publication through `verifyAndPublish`.
- `wormhole/go/evm-ccl-signed-message-immutability`: preserve signed `MessagePublication` fields after observation and update only release metadata such as `effectiveCL` or `additionalBlocks`.
- `wormhole/go/governance-vaa-typed-payload`: production governance VAA construction must pass `CreateGovernanceVAA` a payload from a checked SDK typed governance serializer or `EmptyPayloadVaa`.
- `wormhole/go/guardian-signer-exact-digest-length`: `GuardianSigner.Sign` implementations must reject non-32-byte digest input before signing.
- [`wormhole/go/message-publication-canonical-timestamp`](docs/rules/message-publication-canonical-timestamp.md): use `vaa.TimeFromUnix` for modeled chain-derived publication timestamps.
- [`wormhole/go/message-publication-safe-serialization`](docs/rules/message-publication-safe-serialization.md): avoid deprecated publication serialization helpers that omit security-relevant fields.
- [`wormhole/go/near-finalized-receipt-outcome-before-publication`](docs/rules/near-finalized-receipt-outcome-before-publication.md): require same-outcome NEAR finality and finalized-header provenance before processing receipt logs for publication.
- [`wormhole/go/run-with-scissors-error-return`](docs/rules/run-with-scissors-error-return.md): return runnable errors instead of writing directly to the same `errC`.
- `wormhole/go/solana-alt-owner-before-decode`: prove an RPC-fetched address lookup table account exists and is owned by the ALT program before decoding its bytes.
- [`wormhole/go/solana-commitment-match-before-publication`](docs/rules/solana-commitment-match-before-publication.md): require decoded Solana message commitment to match the exact watcher before scheduling or publication.
- [`wormhole/go/solana-message-account-validation`](docs/rules/solana-message-account-validation.md): require validated constructor provenance for Solana message account data.
- [`wormhole/go/solana-require-successful-transaction-meta`](docs/rules/solana-require-successful-transaction-meta.md): require successful Solana transaction metadata before parsing or processing observations.
- `wormhole/go/untrusted-vaa-use-before-verification`: parsed signed VAAs from untrusted boundaries must be verified with the complete guardian set before storage or external delivery; `vaa.Unmarshal` only checks wire format.
- [`wormhole/go/xrpl-derived-generated-emitter`](docs/rules/xrpl-derived-generated-emitter.md): derive collision-resistant emitters for XRPL-generated messages.
- [`wormhole/go/xrpl-first-memo-only`](docs/rules/xrpl-first-memo-only.md): inspect only the first XRPL memo for Wormhole Core and NTT messages.
- [`wormhole/go/xrpl-require-validated-transaction`](docs/rules/xrpl-require-validated-transaction.md): require a validated-ledger proof before parsing an XRPL transaction.

## Compile

Compile every query in the pack:

```sh
codeql query compile src
```

Compile the registered suite:

```sh
codeql query compile suites/wormhole-go.qls
```

## Test

Run every rule fixture:

```sh
codeql test run test
```

Run one rule fixture:

```sh
codeql test run test/xrpl-first-memo-only
```

## Analyze Wormhole

Create separate Go databases for `node` and `sdk`. CodeQL extracts Go code while the database is created, so use a single source root with subdirectory-scoped build commands rather than creating one database from the whole repository. This avoids unrelated SDK subdirectories such as `sdk/js`, `sdk/js-proto-node`, `sdk/js-proto-web`, `sdk/js-wasm`, and `sdk/rust`.

Create the `node` database:

```sh
codeql database create wormhole-node-go-db \
  --language=go \
  --source-root=.. \
  --command='cd node && go build -a ./...' \
  --overwrite
```

Create the Go SDK database:

```sh
codeql database create wormhole-sdk-go-db \
  --language=go \
  --source-root=.. \
  --command='cd sdk && go build -a ./...' \
  --overwrite
```

Analyze both databases with the registered suite:

```sh
codeql database analyze wormhole-node-go-db \
  suites/wormhole-go.qls \
  --format=sarif-latest \
  --output=wormhole-node-go-lints.sarif

codeql database analyze wormhole-sdk-go-db \
  suites/wormhole-go.qls \
  --format=sarif-latest \
  --output=wormhole-sdk-go-lints.sarif
```

If you already have finalized databases, skip the `database create` commands and run `database analyze` against those database paths.

Alternatively, use the `Makefile`, which defaults the source root to the enclosing repository:

```sh
make scan
```
