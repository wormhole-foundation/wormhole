#!/bin/bash

source $HOME/.cargo/env

git clone https://github.com/MystenLabs/sui.git --branch devnet
cd sui
git reset --hard 9ea7599fe5ca95454e43038ef41884753cee753c

cargo --locked install --path crates/sui
cargo --locked install --path crates/sui-faucet
cargo --locked install --path crates/sui-gateway
cargo --locked install --path crates/sui-node
