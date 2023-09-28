#!/usr/bin/env bash
# This script submits a VAA to devnet
set -e

webHost=$1
vaaHex=${2}
devnetRPC="http://${webHost}:8545"
devnetCoreAddress=0xCfEB869F69431e42cdB54A4F4f105C19C080A601
key=4f3edf983ac636a65a842ce7c78d9aa706d3b113bce9c46f30d7d21715b23b1d # one of the Ganche defaults

cd ./clients/eth

go run main.go execute_governance --contract=${devnetCoreAddress} --rpc=${devnetRPC} --key=${key} ${vaaHex}

echo "done executing_governance."
