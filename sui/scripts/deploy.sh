#!/bin/bash -f

set -x

#Transaction Kind : Publish
#----- Transaction Effects ----
#Status : Success
#Created Objects:
#  - ID: 0x069b6d8ea50a0b0756518cb08ddbbad2babf8ae0 <= STATE , Owner: Account Address ( 0xe6a09658743da40b0f48c4da1f3fa0d34797d0d3 <= OWNER )
#  - ID: 0x73fc05ae6f172f90b12a98cf3ad0b669d6b70e5b <= PACKAGE , Owner: Immutable

cd wormhole
sed -i -e 's/wormhole = .*/wormhole = "0x0"/' Move.toml
make build
sui client publish --gas-budget 10000 | tee publish.log
grep ID: publish.log  | head -2 > ids.log

wa="`grep "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
sed -i -e "s/wormhole = .*/wormhole = \"$wa\"/" Move.toml
state="`grep -v "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
owner="`grep -v "Immutable" ids.log | sed -e 's/^.*( \(.*\) )/\1/'`"
echo "export WORM_PACKAGE=\"$wa\"" > ../env.sh
echo "export WORM_STATE=\"$state\"" >> ../env.sh
echo "export WORM_OWNER=\"$owner\"" >> ../env.sh

