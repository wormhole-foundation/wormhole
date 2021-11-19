#!/bin/bash
goal app create --creator "$OWNER_ADDR" --global-ints 4 --global-byteslices 20 --local-byteslices 0 --local-ints 0 --approval-prog vaa-processor-approval.teal --clear-prog vaa-processor-clear.teal  --app-arg "b64:$GKEYSBASE64" --app-arg int:0 --app-arg int:0 -o create.txn
algokey -t create.txn -o create.stxn sign  -m "$OWNER_MNEMO" 
goal clerk rawsend -f create.stxn

