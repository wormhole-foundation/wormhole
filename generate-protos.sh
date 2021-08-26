#!/usr/bin/env bash
set -euo pipefail

(
  cd tools/
  ./build.sh
)

# TODO(leo): remove after a while
rm -rf bridge

rm -rf node/pkg/proto

tools/bin/buf lint
tools/bin/buf generate
