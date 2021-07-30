#!/usr/bin/env bash

(
  cd tools/
  npm ci
)

rm -rf explorer/src/proto
mkdir -p explorer/src/proto

tools/bin/buf generate --template buf.gen.web.yaml
