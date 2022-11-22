#!/bin/bash -f

. env.sh

cd coin
sui client publish --gas-budget 20000 | tee publish.log
grep ID: publish.log  | head -2 > ids.log
treasury_cap="`grep "Account Address" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
coin_package="`grep "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
echo "export TREASURY_CAP=\"$treasury_cap\"" >> ../env.sh
echo "export COIN_PACKAGE=\"$coin_package\"" >> ../env.sh
cd ../