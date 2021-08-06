#!/usr/bin/env bash
# Regenerate bridge_ui/rust_modules
set -euo pipefail

(
  cd solana
  mkdir -p ../bridge_ui/rust_modules/core
  docker build -t localhost/certusone/wormhole-wasmpack:latest -f Dockerfile.wasm .
  docker run --rm -it --workdir /usr/src/bridge/bridge/program \
    -v $(pwd)/../bridge_ui/rust_modules/core:/usr/src/bridge/bridge/program/pkg \
    -e EMITTER_ADDRESS=11111111111111111111111111111115 \
    localhost/certusone/wormhole-wasmpack:latest \
    /usr/local/cargo/bin/wasm-pack build --target nodejs -- --features wasm
  cp $(pwd)/../bridge_ui/rust_modules/core/. $(pwd)/../clients/solana/pkg/ -R
  docker run --rm -it --workdir /usr/src/bridge/modules/token_bridge/program \
    -v $(pwd)/../bridge_ui/rust_modules/token:/usr/src/bridge/modules/token_bridge/program/pkg \
    -e EMITTER_ADDRESS=11111111111111111111111111111115 \
    localhost/certusone/wormhole-wasmpack:latest \
    /usr/local/cargo/bin/wasm-pack build --target nodejs -- --features wasm
  cp $(pwd)/../bridge_ui/rust_modules/. $(pwd)/../clients/token_bridge/pkg -R
)
