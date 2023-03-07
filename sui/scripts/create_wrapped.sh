#!/bin/bash -f

set -euo pipefail

cd "$(dirname "$0")"

. ../env.sh

echo "Creating wrapped asset..."
echo "$COIN_PACKAGE::coin_witness::COIN_WITNESS"

sui client call --function register_new_coin \
    --module wrapped --package $TOKEN_PACKAGE --gas-budget 20000 \
    --args "$WORM_STATE" "$TOKEN_STATE" "$NEW_WRAPPED_COIN" \
    --type-args "$COIN_PACKAGE::coin_witness::COIN_WITNESS"
