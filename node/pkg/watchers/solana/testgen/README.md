# testgen - Solana watcher replay fixtures

The Solana watcher turns on-chain transactions into Wormhole message publications. `testgen`
builds transaction bundles that the regression test replays through the watcher. For each bundle,
the test compares the emitted VAA signing digests with a recorded baseline. This pins behavior for
the covered corpus; it does not prove injectivity for every possible Solana transaction.

The fixtures live in `testdata/` in two files:

- **`static_bundles.json`** - synthetic bundles generated in code. A matrix that
  sweeps the watcher's inputs (post_message / shim / close, outer / inner, and edge cases).
- **`live_bundles.json`** - real on-chain transactions collected from a Solana RPC node,
  covering what synthetic cases can't (e.g. versioned transactions that resolve accounts
  through Address Lookup Tables).

## Regenerating the fixtures

Run from the `node/` directory:

```sh
go run ./pkg/watchers/solana/testgen/cmd static                        # synthetic matrix (no network)
go run ./pkg/watchers/solana/testgen/cmd live --rpc "$SOLANA_RPC_URL"  # live on-chain txs
go run ./pkg/watchers/solana/testgen/cmd all  --rpc "$SOLANA_RPC_URL"  # both
```

## Recording the expected output

The generator emits bundles with no `expected` block. The replay test produces the digests and
fails until they are recorded:

```sh
UPDATE_SOLANA_REPLAY_FIXTURES=1 go test -v ./pkg/watchers/solana/ -run TestReplayGeneratedBundles
go test ./pkg/watchers/solana/ -run TestReplayGeneratedBundles   # assert against them
```

The env var discards every existing `expected` block and re-records all of them from current
behavior. There is no y/N prompt because `go test` gives the test binary `/dev/null` on stdin.
Setting it is the confirmation; `-v` is what makes the warning banner visible. Review the digest
diff before committing.
