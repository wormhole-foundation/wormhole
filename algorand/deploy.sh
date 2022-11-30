#!/bin/bash

# need netcat 

# Wait for node to start
while ! netcat -z localhost 4001; do
  sleep 1
done

while ! wget http://localhost:4001/genesis -O genesis.json ; do
    sleep 15
done

if [ ! -f genesis.json ]; then
    echo "Failed to create genesis file!"
    exit 1
fi

sleep 2

pipenv run python3 admin.py --devnet --boot --fundDevAccounts

