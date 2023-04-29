#!/bin/bash

source $HOME/.cargo/env

git clone https://github.com/MystenLabs/sui.git --branch devnet
cd sui
# Corresponds to https://github.com/MystenLabs/sui/releases/tag/testnet-0.33.1
git reset --hard c525ba6489261ff6db65e87bf9a3fdda0a6c7be3

cargo --locked install --path crates/sui
cargo --locked install --path crates/sui-faucet
cargo --locked install --path crates/sui-gateway
cargo --locked install --path crates/sui-node
