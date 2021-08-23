#!/usr/bin/env bash
# Regenerate sdk/js/src/solana
set -euo pipefail

(
  cd solana
  mkdir -p ../sdk/js/src/solana/core
  mkdir -p ../sdk/js/src/solana/token
  docker build -t localhost/certusone/wormhole-wasmpack:latest -f Dockerfile.wasm .
  docker run --rm -it --workdir /usr/src/bridge/bridge/program \
    -v $(pwd)/../sdk/js/src/solana/core:/usr/src/bridge/bridge/program/pkg \
    -e EMITTER_ADDRESS=11111111111111111111111111111115 \
    -e BRIDGE_ADDRESS=11111111111111111111111111111115 \
    localhost/certusone/wormhole-wasmpack:latest \
    /usr/local/cargo/bin/wasm-pack build --target bundler -- --features wasm
  docker run --rm -it --workdir /usr/src/bridge/modules/token_bridge/program \
    -v $(pwd)/../sdk/js/src/solana/token:/usr/src/bridge/modules/token_bridge/program/pkg \
    -e EMITTER_ADDRESS=11111111111111111111111111111115 \
    -e BRIDGE_ADDRESS=11111111111111111111111111111115 \
    localhost/certusone/wormhole-wasmpack:latest \
    /usr/local/cargo/bin/wasm-pack build --target bundler -- --features wasm
  docker run --rm -it --workdir /usr/src/bridge/bridge/program \
    -v $(pwd)/../clients/solana/pkg:/usr/src/bridge/bridge/program/pkg \
    -e EMITTER_ADDRESS=11111111111111111111111111111115 \
    -e BRIDGE_ADDRESS=11111111111111111111111111111115 \
    localhost/certusone/wormhole-wasmpack:latest \
    /usr/local/cargo/bin/wasm-pack build --target nodejs -- --features wasm
  cp $(pwd)/../clients/solana/pkg/. $(pwd)/../clients/token_bridge/pkg/core -R
  docker run --rm -it --workdir /usr/src/bridge/modules/token_bridge/program \
    -v $(pwd)/../clients/token_bridge/pkg/token:/usr/src/bridge/modules/token_bridge/program/pkg \
    -e EMITTER_ADDRESS=11111111111111111111111111111115 \
    -e BRIDGE_ADDRESS=11111111111111111111111111111115 \
    localhost/certusone/wormhole-wasmpack:latest \
    /usr/local/cargo/bin/wasm-pack build --target nodejs -- --features wasm
)
