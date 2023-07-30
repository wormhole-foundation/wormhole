#!/bin/bash
set -o errexit -o nounset -o pipefail
command -v shellcheck > /dev/null && shellcheck "$0"

if [ $# -ne 1 ]; then
  echo "Usage: ./download_releases.sh RELEASE_TAG"
  exit 1
fi

tag="$1"

# From CosmosContracts/token-bindings

url="https://github.com/CosmWasm/token-bindings/releases/download/$tag/token_reflect.wasm"
echo "Downloading $url ..."
wget -O "token_reflect.wasm" "$url"

rm -f version.txt
echo "$tag" >version.txt