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
./sandbox up -v dev
echo "run the tests"
cd test
python3 test.py
echo "bring the sandbox down"
cd ..
./sandbox down
