#!/bin/bash

set -x

source $HOME/.cargo/env

sui start &
sleep 5
sui-faucet --host-ip 0.0.0.0&
sleep 2
curl -X POST -d '{"FixedAmountRequest":{"recipient": "'"0x2acab6bb0e4722e528291bc6ca4f097e18ce9331"'"}}' -H 'Content-Type: application/json' http://127.0.0.1:5003/gas
sui-node --config-path ~/.sui/sui_config/fullnode.yaml

#sleep infinity
