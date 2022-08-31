#!/bin/bash

curl --location --request GET 'https://eth-rpc-acala.aca-api.network/' \
--header 'Content-Type: application/json' \
--data-raw '{
    "jsonrpc": "2.0",
    "method": "eth_getEthGas",
    "params": [
        {
            "gasLimit": 21000000,
            "storageLimit": 64100
        }
    ],
    "id": 1
}'
printf "\n"
