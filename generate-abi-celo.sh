#!/usr/bin/env bash
# Regenerate node/pkg/celo/abi.

set -euo pipefail

# Getting this error when building the docker image:
#10 40.92 /usr/lib/gcc/x86_64-alpine-linux-musl/9.3.0/../../../../x86_64-alpine-linux-musl/bin/ld: /go/pkg/mod/github.com/celo-org/celo-bls-go@v0.2.4/bls/../libs/x86_64-unknown-linux-gnu/libbls_snark_sys.a(std-f14aca24435a5414.std.54pte2sm-cgu.0.rcgu.o): in function `std::sys::unix::net::on_resolver_failure':
#10 40.92 /rustc/18bf6b4f01a6feaf7259ba7cdae58031af1b7b39//library/std/src/sys/unix/net.rs:376: undefined reference to `__res_init'
#10 40.92 collect2: error: ld returned 1 exit status

# (
#   cd third_party/abigen-celo
#   docker build -t localhost/certusone/wormhole-abigen-celo:latest .
# )

cd third_party/abigen-celo
go build -o abigen github.com/celo-org/celo-blockchain/cmd/abigen
cd -

function gen() {
  local name=$1
  local pkg=$2

  cd ethereum
  mkdir -p abigenBindings
  npm run abigen -- ${name}
  cat abigenBindings/abi/Implementation.abi | ../third_party/abigen-celo/abigen --abi - --pkg ${pkg} > ../node/pkg/celo/${pkg}/abi.go
  #cat abigenBindings/abi/Implementation.abi | docker run --rm -i localhost/certusone/wormhole-abigen-celo:latest /bin/abigen --abi - --pkg ${pkg} > ../node/pkg/celo/${pkg}/abi.go
  rm -rf abigenBindings
}

gen Implementation abi
#gen ERC20 erc20
