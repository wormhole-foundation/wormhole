#!/bin/bash

set -e

DOTENV=$(realpath "$(dirname "$0")"/../.env)
[ -f $DOTENV ] || (echo "$DOTENV does not exist." >&2; exit 1)

# 1. load variables from .env file
. $DOTENV

# 2. next we get all the token bridge registration VAAs from the environment
# if a new VAA is added, this will automatically pick it up
VAAS=$(set | grep "REGISTER_.*_TOKEN_BRIDGE_VAA" | grep -v SUI | cut -d '=' -f1)

# 3. use 'worm' to submit each registration VAA
for VAA in $VAAS
do
    VAA=${!VAA}
    worm submit $VAA --chain sui --network devnet
done

echo "Registrations successful."
