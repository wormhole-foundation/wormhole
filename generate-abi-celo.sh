#!/usr/bin/env bash
# Regenerate node/pkg/celo/abi.

set -euo pipefail

(
  cd third_party/abigen-celo
  docker build -t localhost/wormhole-foundation/wormhole-abigen-celo:latest .
)

function gen() {
  local name=$1
  local pkg=$2
  
  kubectl exec -c tests eth-devnet-0 -- npx truffle@5.4.1 run abigen $name

  kubectl exec -c tests eth-devnet-0 -- cat abigenBindings/abi/${name}.abi | \
    docker run --rm -i localhost/wormhole-foundation/wormhole-abigen-celo:latest /bin/abigen --abi - --pkg ${pkg} > \
    node/pkg/celo/${pkg}/abi.go
}

gen Implementation abi
