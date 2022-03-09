#!/usr/bin/env bash

chmod 700 /opt/algorand/node/data/kmd-v0.5

/opt/algorand/node/goal kmd start -d /opt/algorand/node/data
/opt/algorand/node/algod -d /opt/algorand/node/data
