#!/bin/bash

curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
source $HOME/.cargo/env
rustup default 1.60.0
rustup update
rustup target add wasm32-unknown-unknown
