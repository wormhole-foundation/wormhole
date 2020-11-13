#!/bin/bash
set -e

kubectl exec -it -c tests eth-devnet-0 -- npx truffle exec src/send-lockups.js
