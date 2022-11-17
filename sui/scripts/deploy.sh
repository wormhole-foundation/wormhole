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
deployer_cap="`grep -v "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
owner="`grep -v "Immutable" ids.log | sed -e 's/^.*( \(.*\) )/\1/'`"
echo "export WORM_PACKAGE=\"$wa\"" > ../env.sh
echo "export WORM_DEPLOYER_CAPABILITY=\"$deployer_cap\"" >> ../env.sh
echo "export WORM_OWNER=\"$owner\"" >> ../env.sh

cd ../token_bridge
sed -i -e 's/token_bridge = .*/token_bridge = "0x0"/' Move.toml
make build
sui client publish --gas-budget 10000 | tee publish.log
grep ID: publish.log  | head -2 > ids.log

wa="`grep "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
sed -i -e "s/token_bridge = .*/token_bridge = \"$wa\"/" Move.toml
deployer_cap="`grep -v "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/'`"
owner="`grep -v "Immutable" ids.log | sed -e 's/^.*( \(.*\) )/\1/'`"
echo "export TOKEN_PACKAGE=\"$wa\"" >> ../env.sh
echo "export TOKEN_DEPLOYER_CAPABILITY=\"$deployer_cap\"" >> ../env.sh
echo "export TOKEN_OWNER=\"$owner\"" >> ../env.sh

. ../env.sh
sui client call --function init_and_share_state --module state --package $WORM_PACKAGE  --gas-budget 20000 --args \"$WORM_DEPLOYER_CAPABILITY\" 0 0 [190,250,66,157,87,205,24,183,248,164,217,26,45,169,171,74,240,93,15,190] [[190,250,66,157,87,205,24,183,248,164,217,26,45,169,171,74,240,93,15,190]] | tee wormhole.log
wormhole=`grep Shared wormhole.log | head -1 | sed -e 's/^.*: \(.*\) ,.*/\1/'`
echo "export WORM_STATE=\"$wormhole\"" >> ../env.sh

. ../env.sh
sui client call --function get_new_emitter --module wormhole --package $WORM_PACKAGE --gas-budget 20000 --args \"$WORM_STATE\" | tee emitter.log
emitter=`grep ID: emitter.log | head -1 | sed -e 's/^.*: \(.*\) ,.*/\1/'`
echo "export TOKEN_EMITTER_CAPABILITY=\"$emitter\"" >> ../env.sh

. ../env.sh
sui client call --function init_and_share_state --module bridge_state --package $TOKEN_PACKAGE  --gas-budget 20000 --args \"$TOKEN_DEPLOYER_CAPABILITY\" \"$TOKEN_EMITTER_CAPABILITY\" | tee token.log
token_bridge=`grep Shared token.log | head -1 | sed -e 's/^.*: \(.*\) ,.*/\1/'`
echo "export TOKEN_STATE=\"$token_bridge\"" >> ../env.sh

