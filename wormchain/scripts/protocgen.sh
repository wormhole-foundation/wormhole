#!/bin/bash

# end early if any command fails
set -eo pipefail

# generate gogo proto code
echo "Generating gogo proto code"
cd proto
buf mod update
cd ..
buf generate

# move proto to x/ directory
cp -r ./github.com/wormhole-foundation/wormchain/x/* x/

# remove github.com directory
rm -rf ./github.com