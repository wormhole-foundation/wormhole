#!/bin/bash -f

. env.sh

sui client call --function create_wrapped --module wrapped --package $TOKEN_PACKAGE --gas-budget 20000 \
--args \"$WORM_STATE\" \"$TOKEN_STATE\" \ --type-args 

