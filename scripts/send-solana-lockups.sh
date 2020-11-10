#!/bin/bash
set -e

kubectl exec -it solana-devnet-0 -c setup ./lockups.sh
