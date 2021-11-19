#!/bin/bash
export STATELESS_ADDR=$(goal clerk compile vaa-verify.teal | cut -f 2 -d ':' | cut -c 2- )
goal app call --app-id $1 --from "$OWNER_ADDR" --app-arg "str:setvphash" --app-arg "addr:$STATELESS_ADDR" -o setvphash.txn 
algokey -t setvphash.txn -o setvphash.stxn sign -m "$OWNER_MNEMO" 
goal clerk rawsend -f setvphash.stxn




