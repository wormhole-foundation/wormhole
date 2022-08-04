#!/bin/bash

apt-get update
apt-get -y install libclang-dev jq

git clone https://github.com/near/nearcore
cd nearcore
make sandbox-release



