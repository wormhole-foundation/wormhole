# Wormhole on NEAR

Unless stated otherwise, run each command block from the repository root. The
blocks use subshells so that one step does not change the working directory for
the next step.

## Building contracts

Contracts are Rust crates compiled to WASM. Build them with:

```sh
(cd near && make build)
```

This invokes `cargo build --target wasm32-unknown-unknown --release` for each contract
under `contracts/`. Requires the Rust toolchain specified in `rust-toolchain.toml` (1.63,
wasm32 target).

To build via Docker (no local Rust needed), install and start Docker, then run:

```sh
(cd near && make artifacts)
```

This writes the exported WASM files and checksums to `near/artifacts/`. The
pinned base image is currently `linux/amd64`, so Docker uses emulation on ARM
hosts. The first build downloads a large image and, because `near/` has no
`.dockerignore`, the build context can be several gigabytes when local build
outputs are present.

## Running tests locally

The integration test (`near/test/test.ts`) exercises the Wormhole core and token
bridge contracts against a local NEAR sandbox. It generates VAAs client-side
using local guardian keys from `testlib.ts` and exercises governance upgrades,
attestations, transfers, and the `ft_transfer_call` flow. NFT bridge coverage is
in the separate `near/test/nft.ts` script; `make test` does not run it.

### Prerequisites

Use Node.js from the repository's `.nvmrc`. Native sandbox builds additionally
require Rust via `rustup`, a C/C++ build toolchain, Git, Make, Python 3, and
`jq`. The pinned nearcore revision and its Rust 1.60 dependencies are known to
build on Linux/x86_64. The native build does not currently complete on Apple
Silicon; use the Tilt devnet described below on unsupported hosts.

1. **Build the SDK** (the test imports from `@certusone/wormhole-sdk`):
   ```sh
   (cd sdk/js && npm ci && npm run build-all)
   ```

   `npm run build-all` is equivalent to `npm run build-deps && npm run
   build-lib`. npm 11 may warn that dependency install scripts have not been
   approved; the SDK and NEAR builds currently complete despite those warnings.

2. **Build NEAR contracts** (required for the WASM files the test deploys):
   ```sh
   (cd near && make build)
   ```

3. **Build and start a NEAR sandbox** locally on Linux/x86_64:
   ```sh
   (cd near && make nearcore) # Clones and builds nearcore; allow at least 10 minutes.
   (cd near && make run)      # Runs the sandbox on :3030 and key server on :3031.
   ```

   Run `make run` in a dedicated terminal and leave it running while the test
   executes. The current target calls `killall -q Python` before startup, which
   can terminate unrelated Python processes. Do not use it on a shared host or
   while other Python workloads are running; prefer Tilt in those environments.

   If `make nearcore` fails after creating `near/nearcore/`, Make considers the
   directory complete and a retry only reports `nearcore is up to date`.
   Remove the partial `near/nearcore/` directory before retrying. Cargo 1.60's
   bundled libgit2 can also reject newer Git configuration values such as
   `merge.conflictStyle=zdiff3`; use an isolated build environment with
   compatible Git defaults or use Tilt rather than changing user-wide Git
   settings.

4. **Run the test**:
   ```sh
   (cd near && npm ci && make test)
   ```

The test first reads `validator_key.json` from the key server on
`localhost:3031`, so it fails immediately if the sandbox is not running. Later,
it calls `getSignedVAAWithRetry` against the guardian spy on `localhost:7071`
and fails if no guardian is running.

For full end-to-end testing, use the intended Tilt environment instead of the
native sandbox commands:

```sh
tilt up -- --near
```

## CI coverage

See `.github/workflows/build.yml` for the full CI matrix.

| What | Where | Status |
|---|---|---|
| Contract compilation | `tilt` job (via `near/Dockerfile.deploy` → `build-contracts.sh`) | Runs on every PR |
| Contract deployment | `tilt` job (via `devnet/near-devnet.yaml` → `devnet_deploy.ts`) | Deploys wormhole, token bridge, nft bridge |
| Integration test (`test/test.ts`) | Not invoked anywhere | The test suite never runs in CI |
| Standalone build-only check | No dedicated job in `build.yml` | Not covered |

The `tilt` job runs NEAR deployment by default in CI mode
(`near = cfg.get("near", ci)` in the root `Tiltfile`). This verifies contracts compile and
deploy. The tests under `sdk/js` perform some testing against NEAR.

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
