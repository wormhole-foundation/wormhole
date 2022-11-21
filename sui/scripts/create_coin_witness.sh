#!/bin/bash -f

. env.sh

cd coin
sui client publish --gas-budget 20000 | tee publish.log
grep ID: publish.log  | head -2 > ids.log
witness_container="`grep "Account Address" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
coin_package="`grep "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
echo "export WITNESS_CONTAINER=\"$witness_container\"" >> ../env.sh
echo "export COIN_PACKAGE=\"$coin_package\"" >> ../env.sh
