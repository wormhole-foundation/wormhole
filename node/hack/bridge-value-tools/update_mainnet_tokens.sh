#!/bin/sh

rm -rf generated_mainnet_tokens.go
wget https://github.com/wormhole-foundation/wormhole/raw/main/node/pkg/governor/generated_mainnet_tokens.go
sed -i 's/package\ governor/package\ main/g' generated_mainnet_tokens.go

rm -rf manual_tokens.go
wget https://github.com/wormhole-foundation/wormhole/raw/main/node/pkg/governor/manual_tokens.go
sed -i 's/package\ governor/package\ main/g' manual_tokens.go
