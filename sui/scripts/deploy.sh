#!/bin/bash -f

set -euo pipefail

cd "$(dirname "$0")"/..

#Transaction Kind : Publish
#----- Transaction Effects ----
#Status : Success
#Created Objects:
#  - ID: 0x069b6d8ea50a0b0756518cb08ddbbad2babf8ae0 <= STATE , Owner: Account Address ( 0xe6a09658743da40b0f48c4da1f3fa0d34797d0d3 <= OWNER )
#  - ID: 0x73fc05ae6f172f90b12a98cf3ad0b669d6b70e5b <= PACKAGE , Owner: Immutable

cd wormhole
sed -i -e "s/wormhole = \"_\"/wormhole = \"0x0\"/" Move.toml
echo "build wormhole"
make build
echo "publishing"
sui client publish --gas-budget 10000 | tee publish.log
echo "published"
grep "ID:" publish.log  | head -2 > ids.log

WORM_PACKAGE=$(grep "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/')
echo "yeah buddy?"
sed -i -e "s/wormhole = \"0x0\"/wormhole = \"${WORM_PACKAGE}\"/" Move.toml
echo "okay"
WORM_DEPLOYER_CAPABILITY=$(grep -v "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/')
WORM_OWNER=$(grep -v "Immutable" ids.log | sed -e 's/^.*( \(.*\) )/\1/')

cd ../token_bridge
sed -i -e 's/token_bridge = "_"/token_bridge = "0x0"/' Move.toml
echo "build token_bridge"
make build
echo "publishing"
sui client publish --gas-budget 10000 | tee publish.log
echo "published"
grep ID: publish.log  | head -2 > ids.log

cd ..
echo "wrap up"
sed -i -e "s/wormhole = \"${WORM_PACKAGE}\"/wormhole = \"_\"/" wormhole/Move.toml
sed -i -e "s/token_bridge = \"0x0\"/token_bridge = \"_\"/" token_bridge/Move.toml

exit 0

TOKEN_PACKAGE=$(grep "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/')
#sed -i -e "s/token_bridge = .*/token_bridge = \"$TOKEN_PACKAGE\"/" Move.toml
TOKEN_DEPLOYER_CAPABILITY=$(grep -v "Immutable" ids.log  | sed -e 's/^.*: \(.*\) ,.*/\1/')
TOKEN_OWNER=$(grep -v "Immutable" ids.log | sed -e 's/^.*( \(.*\) )/\1/')

sui client call --function init_and_share_state --module state --package $WORM_PACKAGE  --gas-budget 20000 --args \"$WORM_DEPLOYER_CAPABILITY\" 0 0 "[190,250,66,157,87,205,24,183,248,164,217,26,45,169,171,74,240,93,15,190]" "[[190,250,66,157,87,205,24,183,248,164,217,26,45,169,171,74,240,93,15,190]]" | tee wormhole.log
WORM_STATE=$(grep Shared wormhole.log | head -1 | sed -e 's/^.*: \(.*\) ,.*/\1/')

sui client call --function get_new_emitter --module wormhole --package $WORM_PACKAGE --gas-budget 20000 --args \"$WORM_STATE\" | tee emitter.log
TOKEN_EMITTER_CAPABILITY=$(grep ID: emitter.log | head -1 | sed -e 's/^.*: \(.*\) ,.*/\1/')

sui client call --function init_and_share_state --module bridge_state --package $TOKEN_PACKAGE  --gas-budget 20000 --args "$TOKEN_DEPLOYER_CAPABILITY" "$TOKEN_EMITTER_CAPABILITY" | tee token.log
TOKEN_STATE=$(grep Shared token.log | head -1 | sed -e 's/^.*: \(.*\) ,.*/\1/')

{ echo "export WORM_PACKAGE=$WORM_PACKAGE";
  echo "export WORM_DEPLOYER_CAPABILITY=$WORM_DEPLOYER_CAPABILITY";
  echo "export WORM_OWNER=$WORM_OWNER";
  echo "export TOKEN_PACKAGE=$TOKEN_PACKAGE";
  echo "export TOKEN_DEPLOYER_CAPABILITY=$TOKEN_DEPLOYER_CAPABILITY";
  echo "export TOKEN_OWNER=$TOKEN_OWNER";
  echo "export WORM_STATE=$WORM_STATE";
  echo "export TOKEN_EMITTER_CAPABILITY=$TOKEN_EMITTER_CAPABILITY";
  echo "export TOKEN_STATE=$TOKEN_STATE";
} > ../env.sh
