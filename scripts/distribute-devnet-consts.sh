#!/usr/bin/env bash
set -euo pipefail

# copy devnet-consts.json to chain dirs for local use, so we can keep docker
# build contexts scoped to the chain, rather than the root just to read this file.
file="./scripts/devnet-consts.json"
paths=(
    ./terra2/tools/
    ./wormchain/contracts/tools/
)

for dest in "${paths[@]}"; do
    dirname=$(dirname $dest)
    if [[ -d "$dirname" ]]; then
        echo "copying $file to $dest"
        cp $file $dest
    fi
done

echo "distribute devnet consts complete!"
