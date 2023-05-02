#!/bin/bash

source $HOME/.cargo/env

git clone https://github.com/MystenLabs/sui.git --branch devnet
cd sui
# Corresponds to https://github.com/MystenLabs/sui/releases/tag/testnet-1.0.0
git reset --hard 09b2081498366df936abae26eea4b2d5cafb2788

cargo --locked install --path crates/sui
cargo --locked install --path crates/sui-faucet
cargo --locked install --path crates/sui-gateway
cargo --locked install --path crates/sui-node
