#!/usr/bin/env bash
# Regenerate explorer
set -euo pipefail

(
  cd ../solana
  mkdir -p ../explorer/wasm/core
  mkdir -p ../explorer/wasm/token

  docker build -t localhost/certusone/wormhole-wasmpack:latest -f Dockerfile.wasm .

  docker run --rm -it --workdir /usr/src/bridge/bridge/program \
    -v $(pwd)/../explorer/wasm/core:/usr/src/bridge/bridge/program/pkg \
    -e EMITTER_ADDRESS=11111111111111111111111111111115 \
    localhost/certusone/wormhole-wasmpack:latest \
    /usr/local/cargo/bin/wasm-pack build --target bundler -- --features wasm

  docker run --rm -it --workdir /usr/src/bridge/modules/token_bridge/program \
    -v $(pwd)/../explorer/wasm/token:/usr/src/bridge/modules/token_bridge/program/pkg \
    -e EMITTER_ADDRESS=11111111111111111111111111111115 \
    localhost/certusone/wormhole-wasmpack:latest \
    /usr/local/cargo/bin/wasm-pack build --target bundler -- --features wasm
)
