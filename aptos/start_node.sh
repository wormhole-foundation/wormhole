#!/bin/bash

set -x

sui genesis -f
sui start &
sleep 5
rpc-server --host 0.0.0.0
