#!/bin/bash

set -x

sui start &
sleep 5
sui-faucet --host-ip 0.0.0.0
