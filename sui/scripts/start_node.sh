#!/bin/bash

set -x

sui start &
sleep 10
#sleep infinity
sui client object --id 0x5
#sui-faucet --host-ip 0.0.0.0&
#sleep 2
#curl -X POST -d '{"FixedAmountRequest":{"recipient": "'"0x2acab6bb0e4722e528291bc6ca4f097e18ce9331"'"}}' -H 'Content-Type: application/json' http://127.0.0.1:5003/gas
sed -i -e 's/:9000/:9002/' ~/.sui/sui_config/fullnode.yaml
sui-node --config-path ~/.sui/sui_config/fullnode.yaml

#sleep infinity
