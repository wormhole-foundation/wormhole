#!/bin/bash -f

. env.sh

sui client call --function get_new_emitter --module wormhole --package $WORM_PACKAGE  --gas-budget 20000 --args \"$WORM_STATE\"
