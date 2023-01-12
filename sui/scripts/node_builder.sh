#!/bin/bash

source $HOME/.cargo/env

git clone https://github.com/MystenLabs/sui.git --branch devnet
cd sui

cargo --locked install --path crates/sui
cargo --locked install --path crates/sui-faucet
cargo --locked install --path crates/sui-gateway
cargo --locked install --path crates/sui-node
