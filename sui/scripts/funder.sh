#!/bin/bash -f

sui client transfer-sui --to 0x2acab6bb0e4722e528291bc6ca4f097e18ce9331 --sui-coin-object-id `sui client objects | grep sui::SUI | tail -1 | sed -e 's/|.*//'` --gas-budget 10000
