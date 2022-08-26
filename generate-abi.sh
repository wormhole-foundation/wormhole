#!/usr/bin/env bash
# Regenerate node/pkg/ethereum/abi using a running eth-devnet's state.
set -euo pipefail

(
  cd third_party/abigen
  docker build -t localhost/wormhole-foundation/wormhole-abigen:latest .
)

function gen() {
  local name=$1
  local pkg=$2

  kubectl exec -c tests eth-devnet-0 -- npx truffle@5.4.1 run abigen $name

  kubectl exec -c tests eth-devnet-0 -- cat abigenBindings/abi/${name}.abi | \
    docker run --rm -i localhost/wormhole-foundation/wormhole-abigen:latest /bin/abigen --abi - --pkg ${pkg} > \
    node/pkg/ethereum/${pkg}/abi.go
}

gen Wormhole abi
gen ERC20 erc20
