#!/bin/bash -f

set -x

cd wormhole
sed -i -e 's/wormhole = .*/wormhole = "0x0"/' Move.toml
make build
sui client publish --gas-budget 10000 | tee publish.log
#wa="`grep "The newly published package" publish.log | sed -e 's/^.*: //'`"
#sed -i -e "s/wormhole = .*/wormhole = \"$wa\"/" Move.toml
#state="`grep "Move Object" publish.log | sed -e 's/^.*(\(.*\)\[.*$/\1/'`"
#echo "WORM_PACKAGE=\"$wa\"" > ../env.sh
#echo "WORM_STATE=\"$state\"" >> ../env.sh
#
