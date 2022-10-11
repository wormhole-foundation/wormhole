#!/bin/bash -f

. env.sh

sui client call --function init_and_share_state --module state --package $WORM_PACKAGE  --gas-budget 20000 --args \"$WORM_STATE\" 0 0 [190,250,66,157,87,205,24,183,248,164,217,26,45,169,171,74,240,93,15,190] [190,250,66,157,87,205,24,183,248,164,217,26,45,169,171,74,240,93,15,190]
