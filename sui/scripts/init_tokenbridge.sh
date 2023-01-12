#!/bin/bash -f

. env.sh

sui client call --function init_and_share_state --module bridge_state --package $TOKEN_PACKAGE --gas-budget 20000 --args \"$TOKEN_STATE\" \"$EMITTER_CAP\"
