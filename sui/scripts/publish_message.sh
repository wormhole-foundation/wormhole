#!/bin/bash -f

. env.sh

sui client call --function publish_message_free --module wormhole --package $WORM_PACKAGE --gas-budget 20000 --args \"$1\" \"$WORM_STATE\" 400 [2]
