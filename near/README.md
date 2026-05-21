# Wormhole on NEAR

## Building contracts

Contracts are Rust crates compiled to WASM. Build them with:

```sh
cd near
make build
```

This invokes `cargo build --target wasm32-unknown-unknown --release` for each contract
under `contracts/`. Requires the Rust toolchain specified in `rust-toolchain.toml` (1.63,
wasm32 target).

To build via Docker (no local Rust needed):

```sh
make artifacts
```

## Running tests locally

The integration test (`near/test/test.ts`) exercises wormhole, token bridge, and NFT bridge
contracts against a local NEAR sandbox. It generates VAAs client-side using local guardian keys
from `testlib.ts` and exercises governance upgrades, attestations, transfers, and the
`ft_transfer_call` flow.

### Prerequisites

1. **Build the SDK** (the test imports from `@certusone/wormhole-sdk`):
   ```sh
   cd sdk/js
   npm ci
   npm run build-all    # or: npm run build-deps && npm run build-lib
   ```

2. **Build NEAR contracts** (required for the WASM files the test deploys):
   ```sh
   cd near
   make build
   ```

3. **Start a NEAR sandbox** locally:
   ```sh
   cd near
   make nearcore    # clones and builds nearcore from source (~10 min, one-time)
   make run         # starts sandbox on :3030, key server on :3031
   ```

4. **Run the test**:
   ```sh
   cd near
   npm ci
   make test
   ```

> **Note:** The test makes a `getSignedVAAWithRetry` call against `localhost:7071` (a
> guardian spy). Without a guardian running, the test will fail at this step.
> A full Tilt devnet (`tilt up -- --near`) is the intended environment for end-to-end
> testing.

## CI coverage

| What | Where | Status |
|---|---|---|
| Contract compilation | `tilt` job (via `near/Dockerfile.deploy` → `build-contracts.sh`) | Runs on every PR |
| Contract deployment | `tilt` job (via `devnet/near-devnet.yaml` → `devnet_deploy.ts`) | Deploys wormhole, token bridge, nft bridge |
| Integration test (`test/test.ts`) | Not invoked anywhere | The test suite never runs in CI |
| Standalone build-only check | No dedicated job in `build.yml` | Not covered |

The `tilt` job runs NEAR deployment by default in CI mode
(`near = cfg.get("near", ci)` in the root `Tiltfile`). This verifies contracts compile and
deploy, but runs no behavioral assertions. See `.github/workflows/build.yml` for the full
CI matrix.

## Docker images

| Dockerfile | Purpose |
|---|---|
| `Dockerfile.base` | Builds nearcore sandbox from source, creates base image `ghcr.io/wormhole-foundation/near:0.2` |
| `Dockerfile` | Production node image (pins the base image) |
| `Dockerfile.build` | Compiles contracts and exports WASM artifacts |
| `Dockerfile.contracts` | Bundles WASM files with deploy scripts |
| `Dockerfile.deploy` | Multi-stage: compiles contracts, then creates a deploy image with `devnet_deploy.ts` |

## Contract layout

| Contract | Path | Purpose |
|---|---|---|
| Wormhole core | `contracts/wormhole/` | VAA verification, guardian set management |
| Token bridge | `contracts/token-bridge/` | Cross-chain token transfers, attestations |
| NFT bridge | `contracts/nft-bridge/` | Cross-chain NFT transfers |
| NFT (wrapped) | `contracts/nft-wrapped/` | Wrapped NFT token implementation |
| Fungible token | `contracts/ft/` | Fungible token standard helpers |
| Mock bridge integration | `contracts/mock-bridge-integration/` | Test helpers for bridge integration |
| Mock bridge token | `contracts/mock-bridge-token/` | Test helpers for bridged tokens |

