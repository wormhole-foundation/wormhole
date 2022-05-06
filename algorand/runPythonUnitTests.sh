#!/usr/bin/env bash
python3 -m pip install -r requirements.txt
if [ ! -d _sandbox ]; then
  echo We need to create it...
  git clone https://github.com/algorand/sandbox.git _sandbox
fi
if [ "`grep enable-all-parameters _sandbox/images/indexer/start.sh | wc -l`" == "0" ]; then
  echo the indexer is incorrectly configured
  sed -i -e 's/dev-mode/dev-mode --enable-all-parameters/'  _sandbox/images/indexer/start.sh
  echo delete all the existing docker images
  ./sandbox clean
fi
./sandbox clean
./sandbox up dev
if [ $? -ne 0 ]; then
	echo failed to bring up the sandbox
	exit 1
fi
echo running the tests...
cd test
python3 test.py
if [ $? -ne 0 ]; then
	echo tests in test.py failed
	exit 1
fi
echo bringing the sandbox down...
cd ..
./sandbox down
