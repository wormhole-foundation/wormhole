#!/bin/bash

source $HOME/.cargo/env

git clone https://github.com/MystenLabs/sui.git --branch devnet
cd sui
git reset --hard 82c9c80c11488858f1d3930f47ec9f335a566683

cargo --locked install --path crates/sui
cargo --locked install --path crates/sui-faucet
cargo --locked install --path crates/sui-gateway
cargo --locked install --path crates/sui-node
