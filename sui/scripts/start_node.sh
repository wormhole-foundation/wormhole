#!/bin/bash

set -x

sui start >/dev/null 2>&1 &
sleep 5
sui-faucet --write-ahead-log faucet.log
