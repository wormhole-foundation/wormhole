# Wormhole Software Development Kit

This directory contains libraries in various languages for developing software that interacts with
wormhole.

## Adding a New ChainID

To add a new ChainID to Wormhole:

1. **Add the constant** in `vaa/structs.go`:
   ```go
   // ChainIDNewChain is the ChainID of NewChain
   ChainIDNewChain ChainID = 99
   ```
   Keep constants in numerical order and follow the naming convention.

2. **Regenerate methods** by running:
   ```bash
   make go-generate
   ```
   This runs `chainid_generator.go` which auto-generates `String()`, `ChainIDFromString()`, and `GetAllNetworkIDs()` methods.

3. **Update other components** as needed:
   - Add to governor chain lists (`node/pkg/governor/mainnet_chains.go`)
   - Add manual tokens if required (`node/pkg/governor/manual_tokens.go`)
   - Update any chain-specific configuration files

# Directory Structure

 * [sdk/](./): Go SDK.  This package must live in this directory so that clients can use the
   `github.com/wormhole-foundation/wormhole/sdk` import path.
 * [vaa/](./vaa/): Go package for using VAAs (Verifiable Action Approval).
 * [js/](./js/README.md): Legacy JavaScript SDK (**Deprecated and Unsupported**)
   * Please use the new Wormhole TypeScript SDK instead: [`@wormhole-foundation/sdk`](https://github.com/wormhole-foundation/wormhole-sdk-ts)
 * [js-proto-node/](./js-proto-node/README.md): NodeJS client protobuf.
 * [js-proto-web/](./js-proto-web/README.md): Web client protobuf.
 * [js-wasm/](./js-wasm/README.md): WebAssembly libraries.
 * [rust/](./rust/): Rust SDK.
