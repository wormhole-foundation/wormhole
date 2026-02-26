# Stacks Integration

Wormhole integration for the [Stacks](https://www.stacks.co/) Bitcoin L2 blockchain.

## Overview

Stacks is a Bitcoin Layer 2 that settles transactions to Bitcoin through Proof-of-Transfer (PoX) consensus. This integration enables cross-chain messaging using Bitcoin blocks as the anchor point for finality.

| Component         | Location                    | Description              |
| ----------------- | --------------------------- | ------------------------ |
| Smart Contracts   | `stacks/contracts/`         | Wormhole Core in Clarity |
| Watcher           | `node/pkg/watchers/stacks/` | Go blockchain monitor    |
| Integration Tests | `stacks/test/`              | Vitest test suite        |
| Devnet Config     | `devnet/stacks-*.yaml`      | Kubernetes manifests     |

## Architecture

```
Bitcoin Block → Stacks Blocks → Transactions → Wormhole Events → Guardian Network
```

The watcher polls the Stacks node for **stable** Bitcoin blocks (via `/v2/info`). Once a block is marked stable by the node, all Stacks blocks anchored to it are processed for Wormhole message events.

### Watcher

The watcher monitors `print` events from the Wormhole Core state contract.

**Key files:**

- `watcher.go` — Polling loop and message processing
- `fetch.go` — Stacks RPC client (v2/v3 endpoints)
- `clarity.go` — Clarity value parsing
- `config.go` — Configuration

**API endpoints used:**

- `/v2/info` — Node info including `stable_burn_block_height`
- `/v2/pox` — PoX epoch info (Stacks 3.0 start height)
- `/v3/tenures/blocks/height/{height}` — Stacks blocks by Bitcoin height
- `/v3/blocks/replay/{blockHash}` — Block transactions with events
- `/v3/transaction/{txID}` — Transaction lookup (re-observation)

## Docker Images

| Image                | Dockerfile                      | Purpose                 |
| -------------------- | ------------------------------- | ----------------------- |
| `stacks-node`        | `stacks/Dockerfile`             | Stacks Core node        |
| `stacks-signer`      | `stacks/Dockerfile`             | Stacks 3.0 signer       |
| `stacks-broadcaster` | `stacks/broadcaster/Dockerfile` | Transaction broadcaster |
| `stacks-stacker`     | `stacks/stacker/Dockerfile`     | PoX stacking service    |
| `stacks-test`        | `stacks/test/Dockerfile`        | Integration test runner |

The main `Dockerfile` builds both `stacks-node` and `stacks-signer` from [stacks-core](https://github.com/stacks-network/stacks-core).

## Devnet

Kubernetes manifests in `devnet/`:

- `stacks-bitcoin.yaml` — Bitcoin test node
- `stacks-node.yaml` — Stacks node
- `stacks-signer.yaml` — Stacks 3.0 signer
- `stacks-stacker.yaml` — PoX stacking
- `stacks-broadcaster.yaml` — Transaction distribution

**Endpoints (when port-forwarded):**

- Stacks RPC: `http://localhost:20443`
- Bitcoin RPC: `http://localhost:18443`

## Resources

- [Stacks Docs](https://docs.stacks.co/)
- [Clarity Reference](https://docs.stacks.co/clarity)
- [stacks-core](https://github.com/stacks-network/stacks-core)
