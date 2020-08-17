#!/bin/bash
# Regenerate bridge/pkg/ethereum/abi using a running eth-devnet's state.

(
  cd third_party/abigen
  docker build -t localhost/certusone/wormhole-abigen:latest .
)

kubectl exec -c tests eth-devnet-0 -- npx truffle run abigen Wormhole

kubectl exec -c tests eth-devnet-0 -- cat abigenBindings/abi/Wormhole.abi | \
  docker run --rm -i localhost/certusone/wormhole-abigen:latest /bin/abigen --abi - --pkg abi > \
  bridge/pkg/ethereum/abi/abi.go
