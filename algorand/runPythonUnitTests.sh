#!/usr/bin/env bash
#python3 -m pip install -r requirements.txt
if [ ! -d _sandbox ]; then
  echo We need to create it...
  git clone https://github.com/algorand/sandbox.git _sandbox
fi

sed -i -e 's@export ALGOD_URL=""@export ALGOD_URL="https://github.com/algorand/go-algorand"@' \
       -e 's/export ALGOD_CHANNEL="stable"/export ALGOD_CHANNEL=""/'   \
       -e 's/export ALGOD_BRANCH=""/export ALGOD_BRANCH="v3.6.2-stable"/'   \
       -e 's/export INDEXER_ENABLE_ALL_PARAMETERS="false"/export INDEXER_ENABLE_ALL_PARAMETERS="true"/'  _sandbox/config.dev

./sandbox clean
./sandbox up -v dev
echo running the tests...
cd test
python3 test.py
rv=$?
echo rv = $rv
if [ $rv -ne 0 ]; then
	echo tests in test.py failed
	exit 1
fi
echo bringing the sandbox down...
cd ..
./sandbox down
