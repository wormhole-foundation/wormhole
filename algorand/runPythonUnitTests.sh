#!/usr/bin/env bash
#python3 -m pip install -r requirements.txt
if [ ! -d _sandbox ]; then
  echo We need to create it...
  git clone https://github.com/algorand/sandbox.git _sandbox
  cd _sandbox
  git checkout 4e613dcff61523693c18584894ee6de9bd469ec1
  cd ..
fi

sed -i -e 's@export ALGOD_URL=""@export ALGOD_URL="https://github.com/algorand/go-algorand"@' \
       -e 's/export ALGOD_CHANNEL="stable"/export ALGOD_CHANNEL=""/'   \
       -e 's/export ALGOD_BRANCH=""/export ALGOD_BRANCH="v3.16.2-stable"/'   \
       -e 's/export INDEXER_BRANCH="master"/export INDEXER_BRANCH="2.15.4"/'   \
       -e 's/export INDEXER_ENABLE_ALL_PARAMETERS="false"/export INDEXER_ENABLE_ALL_PARAMETERS="true"/'  _sandbox/config.dev

cd _sandbox

# NOTE: This is a workaround for a bug. It's already fixed in `d8e60ed1a6203f02d3b4702e2e2eefdb7f246f92` in the sandbox
# repository, but we're not ready to upgrade. This allows docker to work in the meantime.
# These lines can be removed when we update the commit hash.
sed -i -e 's/docker compose help/docker compose --help/' ./sandbox
sed -i -e 's/-eq 16/-eq 0/' ./sandbox

# NOTE: workaround for the algod image's Debian bullseye base going EOL:
# security.debian.org no longer reliably serves bullseye-security packages,
# so apt-get install fails inside the algod build. Move it to a Go image
# built on a currently-supported Debian release instead.
sed -i -e 's/ARG GO_VERSION=1.17.5/ARG GO_VERSION=1.24-bookworm/' ./images/algod/Dockerfile

./sandbox clean
./sandbox up -v dev
cd ..
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
cd ../_sandbox
./sandbox down
