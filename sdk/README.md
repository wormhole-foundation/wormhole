# Wormhole Software Development Kit

This directory contains libraries in various languages for developing software that interacts with
wormhole.

# Directory Structure

 * [sdk/](./): Go SDK.  This package must live in this directory so that clients can use the
   `github.com/wormhole-foundation/wormhole/sdk` import path.
 * [vaa/](./vaa/): Go package for using VAAs (Verifiable Action Approval).
 * [js/](./js/README.md): Javascript SDK.
 * [js-proto-node/](./js-proto-node/README.md): NodeJS client protobuf.
 * [js-proto-web/](./js-proto-web/README.md): Web client protobuf.
 * [js-wasm/](./js-wasm/README.md): WebAssembly libraries.
 * [rust/](./rust/): Rust SDK.
