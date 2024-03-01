#!/bin/bash

git clone https://github.com/MystenLabs/sui.git --branch devnet
cd sui
# Corresponds to https://github.com/MystenLabs/sui/releases/tag/mainnet-v1.19.1
git reset --hard 041c5f2bae2fe52079e44b70514333532d69f4e6

cargo --locked install --path crates/sui
cargo --locked install --path crates/sui-faucet
cargo --locked install --path crates/sui-gateway
cargo --locked install --path crates/sui-node
