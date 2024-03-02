#!/usr/bin/env bash

set -euo pipefail

# This is the address corresponding to the private key in the networks.ts file of the worm cli (for the sui devnet wallet)
# The faucet doesn't allow us to specify how many tokens we need, so we just run the command 20 times.
for i in {1..20}
do
  sui client faucet --url http://localhost:5003/gas --address 0xed867315e3f7c83ae82e6d5858b6a6cc57c291fd84f7509646ebc8162169cf96
done

sleep 10
