#!/bin/bash -f

. env.sh

sui client call --function publish_message --module wormhole --package $WORM_ACCOUNT --gas-budget 20000 --args \"$WORM_STATE\" 400 [2] \"0x29651a048dda2a8e07e793cc0f4274b256293436\"
