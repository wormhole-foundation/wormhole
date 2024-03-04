#!/bin/bash

network="$1"
valid_networks=("devnet" "testnet" "mainnet" "reset")

usage() {
    echo "Usage: $0 {devnet|testnet|mainnet|reset}" >&2
    exit 1
}

if [[ ! " ${valid_networks[@]} " =~ " ${network} " ]]; then
    echo "Error: Unrecognized network '${network}'."
    usage
fi

git ls-files | grep 'Move.toml' | while read -r file; do
    if [[ "$network" == "reset" ]]; then
        echo "Resetting $file"
        git checkout "$file" --quiet
    else
        dir=$(dirname "$file")
        base=$(basename "$file")
        new_file="${dir}/Move.$network.toml"
        if [ -f "$new_file" ]; then
            echo "Switching $file to $new_file"
            rm "$file"
            # Create a relative symlink
            (cd "$dir" && ln -s "$(basename "$new_file")" "$base")
        fi
    fi
done
