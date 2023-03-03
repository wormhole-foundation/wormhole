#!/usr/bin/env bash

# This dev script funds the given account. It defaults to the account that is 
# created in `import_default_account.sh`.

sui client transfer-sui --to 0x13b3cb89cf3226d3b860294fc75dc6c91f0c5ecf --sui-coin-object-id `sui client objects | grep sui::SUI | tail -1 | sed -e 's/|.*//'` --gas-budget 10000
