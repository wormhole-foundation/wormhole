#!/bin/bash
goal app delete --app-id $1 --from "$OWNER_ADDR" -o delete.txn
algokey -t delete.txn -o delete.stxn sign  -m "$OWNER_MNEMO"
goal clerk rawsend -f delete.stxn
