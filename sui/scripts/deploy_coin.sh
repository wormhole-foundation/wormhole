#!/bin/bash -f

set -euo pipefail

cd "$(dirname "$0")"/..

. env.sh

sui client publish --gas-budget 20000 --path coin | tee publish.log
grep ID: publish.log  | head -2 > ids.log
NEW_WRAPPED_COIN=$(grep "Account Address" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/')
COIN_PACKAGE=$(grep "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/')
echo "export NEW_WRAPPED_COIN=$NEW_WRAPPED_COIN" >> env.sh
echo "export COIN_PACKAGE=$COIN_PACKAGE" >> env.sh
