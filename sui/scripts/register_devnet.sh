#!/bin/bash

set -e

DOTENV=../.env
[ -f $DOTENV ] || (echo "$DOTENV does not exist." >&2; exit 1)

# 1. load variables from .env file
. $DOTENV

# 2. next we get all the token bridge registration VAAs from the environment
# if a new VAA is added, this will automatically pick it up
VAAS=$(set | grep "REGISTER_.*_TOKEN_BRIDGE_VAA" | grep -v SUI | cut -d '=' -f1)

# 3. use 'worm' to submit each registration VAA
# we'll send the registration calls in parallel, but we want to wait on them at
# the end, so we collect the PIDs
registration_pids=()
for VAA in $VAAS
do
    VAA=${!VAA}
    worm submit "$VAA" --chain sui --network devnet &
    registration_pids+=( $! )
done

# wait on registration calls
for pid in "${registration_pids[@]}"; do
        wait "$pid"
done

echo "Registrations successful."
